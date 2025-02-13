package plugin

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
)

// Args represents the plugin's configurable arguments.
type Args struct {
	ReportDirectory       string `envconfig:"PLUGIN_REPORT_DIRECTORY"`
	ReportFileNamePattern string `envconfig:"PLUGIN_REPORT_FILE_NAME_PATTERN"`
	PassThreshold         int    `envconfig:"PLUGIN_PASS_THRESHOLD"`
	UnstableThreshold     int    `envconfig:"PLUGIN_UNSTABLE_THRESHOLD"`
	CountSkippedTests     bool   `envconfig:"PLUGIN_COUNT_SKIPPED_TESTS"`
	OnlyCritical          bool   `envconfig:"PLUGIN_ONLY_CRITICAL"`
	Level                 string `envconfig:"PLUGIN_LOG_LEVEL"`
}

// ValidateInputs ensures valid plugin arguments.
func ValidateInputs(args Args) error {
	if args.ReportDirectory == "" {
		return errors.New("output path is required")
	}
	if args.ReportFileNamePattern == "" {
		args.ReportFileNamePattern = "*.xml"
	}
	if args.PassThreshold < 0 || args.UnstableThreshold < 0 {
		return errors.New("threshold values must be non-negative")
	}
	return nil
}

// Exec processes Robot Framework output files and extracts statistics.
func Exec(ctx context.Context, args Args) error {
	files, err := locateFiles(args.ReportDirectory, args.ReportFileNamePattern)
	if err != nil {
		logrus.Errorf("Error locating files: %v", err)
		return fmt.Errorf("failed to locate files: %v", err)
	}

	if len(files) == 0 {
		return errors.New("no Robot Framework output files found. Check the report file pattern")
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	stats := StatsResult{}

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			fileStats, err := processFile(f, args.CountSkippedTests, args.OnlyCritical)
			if err != nil {
				logrus.Warnf("Failed to process file %s: %v", f, err)
				return
			}
			mu.Lock()
			aggregateStats(&stats, fileStats)
			mu.Unlock()
		}(file)
	}
	wg.Wait()

	logAggregatedResults(stats)
	writeTestStats(stats)

	// Validate against thresholds
	if err := validateThresholds(stats, args); err != nil {
		return err
	}

	return nil
}

// locateFiles finds output.xml files matching the given pattern.
func locateFiles(directory, fileName string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(directory, fileName))
	if err != nil {
		logrus.WithError(err).WithField("Pattern", fileName).Error("Error occurred while searching for files")
		return nil, fmt.Errorf("failed to search for files: %v", err)
	}

	logrus.Infof("Found %d files matching the pattern: %s", len(matches), fileName)

	if len(matches) == 0 {
		return nil, errors.New("no files found matching the report filename pattern")
	}

	validFiles := []string{}
	for _, file := range matches {
		if fileInfo, err := os.Stat(file); err == nil {
			if fileInfo.Mode().Perm()&(1<<(uint(7))) != 0 {
				validFiles = append(validFiles, file)
			} else {
				logrus.Warnf("File found but not readable: %s", file)
			}
		} else {
			logrus.Warnf("Error accessing file: %s. Error: %v", file, err)
		}
	}

	logrus.Infof("Number of readable files: %d", len(validFiles))

	if len(validFiles) == 0 {
		return nil, errors.New("no readable files found matching the report filename pattern")
	}

	return validFiles, nil
}

func processFile(filename string, countSkipped, onlyCritical bool) (StatsResult, error) {
	logrus.Infof("Processing file: %s", filename)

	fileContent, err := os.ReadFile(filename)
	if err != nil {
		logrus.Errorf("Error opening file: %s. Error: %v", filename, err)
		return StatsResult{}, fmt.Errorf("error opening file: %s. Error: %v", filename, err)
	}

	// ✅ Handle empty files properly
	if len(fileContent) == 0 {
		logrus.Warnf("Skipping empty file: %s", filename)
		return StatsResult{}, nil
	}

	var robotOutput RobotOutput
	err = xml.Unmarshal(fileContent, &robotOutput)
	if err != nil {
		logrus.Errorf("Failed to parse XML: %v", err)
		return StatsResult{}, fmt.Errorf("failed to parse output.xml: %v", err)
	}

	// ✅ Prevent empty suites from being counted
	if len(robotOutput.Suite.Tests) == 0 && len(robotOutput.Suite.Suites) == 0 {
		logrus.Warnf("Skipping suite with no tests: %s", filename)
		return StatsResult{}, nil
	}

	return computeStats(robotOutput, onlyCritical, countSkipped), nil
}

// validateThresholds checks test results against configured thresholds.
func validateThresholds(stats StatsResult, args Args) error {
	if stats.FailedTests > args.PassThreshold {
		return fmt.Errorf("failed tests count (%d) exceeds the pass threshold (%d)", stats.FailedTests, args.PassThreshold)
	}
	if stats.FailedTests > args.UnstableThreshold {
		logrus.Warnf("Warning: failed tests count (%d) exceeds the unstable threshold (%d)", stats.FailedTests, args.UnstableThreshold)
	}
	return nil
}

// aggregateStats merges statistics from multiple files.
func aggregateStats(stats *StatsResult, fileStats StatsResult) {
	// Aggregate basic test and keyword counts
	stats.TotalSuites += fileStats.TotalSuites
	stats.TotalTests += fileStats.TotalTests
	stats.PassedTests += fileStats.PassedTests
	stats.FailedTests += fileStats.FailedTests
	stats.SkippedTests += fileStats.SkippedTests
	stats.TotalKeywords += fileStats.TotalKeywords
	stats.PassedKeywords += fileStats.PassedKeywords
	stats.FailedKeywords += fileStats.FailedKeywords
	stats.SkippedKeywords += fileStats.SkippedKeywords

	// Aggregate critical test counts
	stats.TotalCritical += fileStats.TotalCritical
	stats.CriticalPassed += fileStats.CriticalPassed
	stats.CriticalFailed += fileStats.CriticalFailed

	// Merge failed test details
	stats.FailedTestsDetails = append(stats.FailedTestsDetails, fileStats.FailedTestsDetails...)

	// Aggregate execution time
	stats.ExecutionTime += fileStats.ExecutionTime

	// Compute failure and skipped rates safely (avoid division by zero)
	if stats.TotalTests > 0 {
		stats.FailureRate = (float64(stats.FailedTests) / float64(stats.TotalTests)) * 100
		stats.SkippedRate = (float64(stats.SkippedTests) / float64(stats.TotalTests)) * 100
	} else {
		stats.FailureRate = 0
		stats.SkippedRate = 0
	}
}

// logAggregatedResults logs a detailed summary of the test execution.
func logAggregatedResults(stats StatsResult) {
	logrus.Infof("\n===============================================\n")
	logrus.Infof("Robot Framework Test Report Summary\n")
	logrus.Infof("===============================================\n")
	logrus.Infof("📂 Total Test Suites: %d\n", stats.TotalSuites)
	logrus.Infof("📄 Total Test Cases: %d\n", stats.TotalTests)
	logrus.Infof("✅ Passed Tests: %d\n", stats.PassedTests)
	logrus.Infof("❌ Failed Tests: %d\n", stats.FailedTests)
	logrus.Infof("⏸ Skipped Tests: %d\n", stats.SkippedTests)
	logrus.Infof("🔥 Critical Tests: %d\n", stats.TotalCritical)
	logrus.Infof("✅ Critical Passed: %d\n", stats.CriticalPassed)
	logrus.Infof("❌ Critical Failed: %d\n", stats.CriticalFailed)
	logrus.Infof("📌 Total Keywords: %d\n", stats.TotalKeywords)
	logrus.Infof("✅ Passed Keywords: %d\n", stats.PassedKeywords)
	logrus.Infof("❌ Failed Keywords: %d\n", stats.FailedKeywords)
	logrus.Infof("⏸ Skipped Keywords: %d\n", stats.SkippedKeywords)
	logrus.Infof("📉 Failure Rate: %.2f%%\n", stats.FailureRate)
	logrus.Infof("📉 Skipped Rate: %.2f%%\n", stats.SkippedRate)
	logrus.Infof("⏱️ Total Execution Time: %.2f ms\n", stats.ExecutionTime)
	logrus.Infof("===============================================\n")

	// Log failed test details if any
	if len(stats.FailedTestsDetails) > 0 {
		logrus.Infof("Failed Test Details:\n")
		logrus.Infof("-----------------------------------------------\n")
		for i, test := range stats.FailedTestsDetails {
			logrus.Infof("%d. Test Name: %s\n", i+1, test.Name)
			logrus.Infof("   Suite: %s\n", test.Suite)
			logrus.Infof("   Status: %s\n", test.Status)
			logrus.Infof("   Error Message: %s\n", test.ErrorMessage)
			logrus.Infof("-----------------------------------------------\n")
		}
	}
}

// writeTestStats writes test statistics to DRONE_OUTPUT.
func writeTestStats(stats StatsResult) {
	statsMap := map[string]string{
		"TOTAL_TESTS":      strconv.Itoa(stats.TotalTests),
		"PASSED_TESTS":     strconv.Itoa(stats.PassedTests),
		"FAILED_TESTS":     strconv.Itoa(stats.FailedTests),
		"SKIPPED_TESTS":    strconv.Itoa(stats.SkippedTests),
		"TOTAL_KEYWORDS":   strconv.Itoa(stats.TotalKeywords),
		"PASSED_KEYWORDS":  strconv.Itoa(stats.PassedKeywords),
		"FAILED_KEYWORDS":  strconv.Itoa(stats.FailedKeywords),
		"SKIPPED_KEYWORDS": strconv.Itoa(stats.SkippedKeywords),
		"TOTAL_CRITICAL":   strconv.Itoa(stats.TotalCritical),
		"CRITICAL_PASSED":  strconv.Itoa(stats.CriticalPassed),
		"CRITICAL_FAILED":  strconv.Itoa(stats.CriticalFailed),
		"FAILURE_RATE":     fmt.Sprintf("%.2f", stats.FailureRate),
		"SKIPPED_RATE":     fmt.Sprintf("%.2f", stats.SkippedRate),
	}

	for key, value := range statsMap {
		WriteEnvToFile(key, value)
	}
}

// WriteEnvToFile writes a key-value pair to DRONE_OUTPUT.
func WriteEnvToFile(key, value string) {
	outputFile, _ := os.OpenFile(os.Getenv("DRONE_OUTPUT"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer outputFile.Close()
	outputFile.WriteString(key + "=" + value + "\n")
}
