package plugin

import (
	"sync"
	"time"
)

func computeStats(robotOutput RobotOutput, onlyCritical, countSkipped bool) StatsResult {
	stats := StatsResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		processSuite(&robotOutput.Suite, &stats, onlyCritical, countSkipped, &mu)
	}()
	wg.Wait()

	// ✅ Compute failure & skipped rates safely (avoid division by zero)
	if stats.TotalTests > 0 {
		stats.FailureRate = (float64(stats.FailedTests) / float64(stats.TotalTests)) * 100
		stats.SkippedRate = (float64(stats.SkippedTests) / float64(stats.TotalTests)) * 100
	} else {
		stats.FailureRate = 0
		stats.SkippedRate = 0
	}

	return stats
}

func processSuite(suite *Suite, stats *StatsResult, onlyCritical, countSkipped bool, mu *sync.Mutex) {
	mu.Lock()
	if len(suite.Tests) > 0 || len(suite.Suites) > 0 { // ✅ Prevent empty suite from counting
		stats.TotalSuites++ // Increment suite count only if it has tests or sub-suites
	}
	mu.Unlock()

	// ✅ Extract suite execution time
	startTime, errStart := parseRobotTime(suite.Status.StartTime)
	endTime, errEnd := parseRobotTime(suite.Status.EndTime)

	// ✅ Only add execution time if parsing was successful
	if errStart == nil && errEnd == nil {
		suiteExecutionTime := endTime.Sub(startTime).Seconds() * 1000 // Convert to ms
		mu.Lock()
		stats.ExecutionTime += suiteExecutionTime
		mu.Unlock()
	}

	for _, test := range suite.Tests {
		// Skip non-critical tests if OnlyCritical is enabled
		if onlyCritical && test.Status.Critical != "yes" {
			continue
		}

		mu.Lock()
		stats.TotalTests++

		// ✅ Extract execution time for individual tests
		testStartTime, errStart := parseRobotTime(test.Status.StartTime)
		testEndTime, errEnd := parseRobotTime(test.Status.EndTime)

		if errStart == nil && errEnd == nil {
			stats.ExecutionTime += testEndTime.Sub(testStartTime).Seconds() * 1000 // Convert to ms
		}

		// ✅ Track critical tests
		if test.Status.Critical == "yes" {
			stats.TotalCritical++
		}

		// ✅ Extract error messages
		errorMsg := ""
		for _, msg := range test.Status.Messages {
			if msg.Level == "ERROR" {
				errorMsg = msg.Text
			}
		}

		// ✅ Count pass/fail/skip stats
		if test.Status.Status == "PASS" {
			stats.PassedTests++
			if test.Status.Critical == "yes" {
				stats.CriticalPassed++
			}
		} else if test.Status.Status == "FAIL" {
			stats.FailedTests++
			if test.Status.Critical == "yes" {
				stats.CriticalFailed++
			}

			// ✅ Store failed test details
			stats.FailedTestsDetails = append(stats.FailedTestsDetails, FailedTestDetails{
				Name:         test.Name,
				Suite:        suite.Name,
				Status:       "FAIL",
				ErrorMessage: errorMsg,
			})
		} else if test.Status.Status == "SKIP" {
			stats.SkippedTests++
		}

		// ✅ Process test-level keywords
		for _, kw := range test.Keywords {
			processKeyword(&kw, stats, mu)
		}

		mu.Unlock()
	}

	// Recursively process sub-suites
	for _, subSuite := range suite.Suites {
		processSuite(&subSuite, stats, onlyCritical, countSkipped, mu)
	}
}

// parseRobotTime converts Robot Framework timestamps to Go time.
func parseRobotTime(timestamp string) (time.Time, error) {
	layout := "20060102 15:04:05.000"
	return time.Parse(layout, timestamp)
}

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

	// Recursively process nested keywords
	for _, subKw := range kw.Keywords {
		processKeyword(&subKw, stats, mu)
	}
}
