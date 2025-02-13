package plugin

import (
	"sync"
	"time"
)

// computeStats calculates all test statistics from the parsed XML.
func computeStats(robotOutput RobotOutput, onlyCritical, countSkipped bool) StatsResult {
	stats := StatsResult{}
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		processSuite(&robotOutput.Suite, &stats, onlyCritical, countSkipped)
	}()
	wg.Wait()

	// ✅ Compute failure & skipped rates safely (avoid division by zero)
	if stats.TotalTests > 0 {
		stats.FailureRate = (float64(stats.FailedTests) / float64(stats.TotalTests)) * 100
		stats.SkippedRate = (float64(stats.SkippedTests) / float64(stats.TotalTests)) * 100
	} else {
		stats.FailureRate, stats.SkippedRate = 0, 0
	}

	return stats
}

// processSuite extracts statistics recursively.
func processSuite(suite *Suite, stats *StatsResult, onlyCritical, countSkipped bool) {
	var mu sync.Mutex

	if len(suite.Tests) > 0 || len(suite.Suites) > 0 {
		mu.Lock()
		stats.TotalSuites++
		mu.Unlock()
	}

	// ✅ Extract suite execution time
	startTime, errStart := parseRobotTime(suite.Status.StartTime)
	endTime, errEnd := parseRobotTime(suite.Status.EndTime)
	if errStart == nil && errEnd == nil {
		executionTime := int(endTime.Sub(startTime).Milliseconds()) // ✅ Convert int64 to int
		mu.Lock()
		stats.ExecutionTime += float64(executionTime)
		mu.Unlock()
	}

	var wg sync.WaitGroup

	for _, test := range suite.Tests {
		if onlyCritical && test.Status.Critical != "yes" {
			continue // ✅ Skip non-critical tests if onlyCritical flag is enabled
		}

		wg.Add(1)
		go func(test Test) {
			defer wg.Done()
			processTest(test, suite.Name, stats, &mu, countSkipped)
		}(test)
	}

	for _, subSuite := range suite.Suites {
		wg.Add(1)
		go func(subSuite Suite) {
			defer wg.Done()
			processSuite(&subSuite, stats, onlyCritical, countSkipped)
		}(subSuite)
	}

	wg.Wait()
}

// processTest processes a single test case and updates statistics.
func processTest(test Test, suiteName string, stats *StatsResult, mu *sync.Mutex, countSkipped bool) {
	mu.Lock()
	stats.TotalTests++
	mu.Unlock()

	// ✅ Extract execution time for individual tests
	startTime, errStart := parseRobotTime(test.Status.StartTime)
	endTime, errEnd := parseRobotTime(test.Status.EndTime)
	if errStart == nil && errEnd == nil {
		executionTime := int(endTime.Sub(startTime).Milliseconds()) // ✅ Convert int64 to int
		mu.Lock()
		stats.ExecutionTime += float64(executionTime)
		mu.Unlock()
	}

	// ✅ Track critical tests
	if test.Status.Critical == "yes" {
		mu.Lock()
		stats.TotalCritical++
		mu.Unlock()
	}

	// ✅ Extract error messages
	errorMsg := ""
	for _, msg := range test.Status.Messages {
		if msg.Level == "ERROR" {
			errorMsg = msg.Text
		}
	}

	// ✅ Count pass/fail/skip stats
	mu.Lock()
	switch test.Status.Status {
	case "PASS":
		stats.PassedTests++
		if test.Status.Critical == "yes" {
			stats.CriticalPassed++
		}
	case "FAIL":
		stats.FailedTests++
		if test.Status.Critical == "yes" {
			stats.CriticalFailed++
		}
		stats.FailedTestsDetails = append(stats.FailedTestsDetails, FailedTestDetails{
			Name:         test.Name,
			Suite:        suiteName,
			Status:       "FAIL",
			ErrorMessage: errorMsg,
		})
	case "SKIP":
		if countSkipped {
			stats.SkippedTests++
		}
	}
	mu.Unlock()

	// ✅ Process test-level keywords
	for _, kw := range test.Keywords {
		processKeyword(&kw, stats, mu)
	}
}

// processKeyword processes a keyword inside a test case or suite.
func processKeyword(kw *Keyword, stats *StatsResult, mu *sync.Mutex) {
	mu.Lock()
	stats.TotalKeywords++

	switch kw.Status.Status {
	case "PASS":
		stats.PassedKeywords++
	case "FAIL":
		stats.FailedKeywords++
	case "SKIP":
		stats.SkippedKeywords++
	}

	mu.Unlock()

	// ✅ Recursively process nested keywords
	for _, subKw := range kw.Keywords {
		processKeyword(&subKw, stats, mu)
	}
}

// parseRobotTime converts Robot Framework timestamps to Go time.
func parseRobotTime(timestamp string) (time.Time, error) {
	layout := "20060102 15:04:05.000"
	return time.Parse(layout, timestamp)
}
