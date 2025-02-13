package plugin

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestValidateInputs validates input arguments for correctness
func TestValidateInputs(t *testing.T) {
	tests := []struct {
		name      string
		args      Args
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid Inputs",
			args: Args{
				ReportDirectory:       "./testdata",
				ReportFileNamePattern: "robot_report.xml",
				PassThreshold:         5,
				UnstableThreshold:     10,
			},
			expectErr: false,
		},
		{
			name: "Missing Report Directory",
			args: Args{
				ReportFileNamePattern: "robot_report.xml",
			},
			expectErr: true,
			errMsg:    "report directory is required",
		},
		{
			name: "Negative Thresholds",
			args: Args{
				ReportDirectory:       "./testdata",
				ReportFileNamePattern: "robot_report.xml",
				PassThreshold:         -1,
			},
			expectErr: true,
			errMsg:    "threshold values must be non-negative",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInputs(tc.args)
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestLocateFiles tests locating files with valid and invalid paths
func TestLocateFiles(t *testing.T) {
	tests := []struct {
		name          string
		directory     string
		outputFile    string
		expectedErr   bool
		errMsg        string
		expectedFiles int
	}{
		{
			name:          "Valid Directory and File",
			directory:     "../testdata",
			outputFile:    "robot_report.xml",
			expectedErr:   false,
			expectedFiles: 1,
		},
		{
			name:          "Valid Directory with Glob Pattern",
			directory:     "../testdata",
			outputFile:    "*.xml",
			expectedErr:   false,
			expectedFiles: 2,
		},
		{
			name:        "Invalid Directory",
			directory:   "./invalid",
			outputFile:  "robot_report.xml",
			expectedErr: true,
			errMsg:      "no files found matching the report filename pattern",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tc.directory)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}
			t.Logf("Checking directory: %s", absPath)

			files, err := locateFiles(tc.directory, tc.outputFile)
			if tc.expectedErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else if len(files) != tc.expectedFiles {
				t.Errorf("Expected %d files, but got %d", tc.expectedFiles, len(files))
			}
		})
	}
}

func TestProcessFile(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		expectErr bool
		errMsg    string
		expected  StatsResult
	}{
		{
			name:     "Valid Robot Framework XML Report",
			filePath: "../testdata/robot_report.xml",
			expected: StatsResult{
				TotalSuites:    1,
				TotalTests:     4,
				PassedTests:    1,
				FailedTests:    2,
				SkippedTests:   1,
				TotalCritical:  2,
				CriticalPassed: 1,
				CriticalFailed: 1,
				FailureRate:    50.00,
				SkippedRate:    25.00,
				ExecutionTime:  10960,
				FailedTestsDetails: []FailedTestDetails{
					{
						Name:         "Test Case 2 - Critical Fail",
						Suite:        "Advanced Test Suite",
						Status:       "FAIL",
						ErrorMessage: "Critical test failed: Major issue detected",
					},
					{
						Name:         "Test Case 3 - Non-Critical Fail",
						Suite:        "Advanced Test Suite",
						Status:       "FAIL",
						ErrorMessage: "Non-critical test failed",
					},
				},
			},
		},
		{
			name:     "Empty File",
			filePath: "../testdata/empty.xml",
			expected: StatsResult{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := processFile(tc.filePath, false, false)
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("Results mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestExec tests overall report execution process
func TestExec(t *testing.T) {
	tests := []struct {
		name      string
		args      Args
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid Execution",
			args: Args{
				ReportDirectory:       "../testdata",
				ReportFileNamePattern: "robot_report.xml",
				PassThreshold:         5,
			},
			expectErr: false,
		},
		{
			name: "No XML Reports Found",
			args: Args{
				ReportDirectory:       "../testdata",
				ReportFileNamePattern: "invalid.xml",
			},
			expectErr: true,
			errMsg:    "failed to locate files: no files found matching the report filename pattern",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Exec(context.Background(), tc.args)
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestValidateThresholds tests threshold validation logic
func TestValidateThresholds(t *testing.T) {
	tests := []struct {
		name      string
		results   StatsResult
		args      Args
		expectErr bool
		errMsg    string
	}{
		{
			name: "Passes All Thresholds",
			results: StatsResult{
				TotalTests:  10,
				FailedTests: 1,
			},
			args: Args{
				PassThreshold: 5,
			},
			expectErr: false,
		},
		{
			name: "Failed Tests Exceed Threshold",
			results: StatsResult{
				TotalTests:  10,
				FailedTests: 6,
			},
			args: Args{
				PassThreshold: 5,
			},
			expectErr: true,
			errMsg:    "failed tests count (6) exceeds the pass threshold (5)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateThresholds(tc.results, tc.args)
			if tc.expectErr {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error '%s', but got %v", tc.errMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestComputeStats validates the computation of test statistics.
func TestComputeStats(t *testing.T) {
	tests := []struct {
		name          string
		robotOutput   RobotOutput
		onlyCritical  bool
		countSkipped  bool
		expectedStats StatsResult
	}{
		{
			name: "All tests pass",
			robotOutput: RobotOutput{
				Suite: Suite{
					Tests: []Test{
						{Name: "Test 1", Status: Status{Status: "PASS", Critical: "yes"}},
						{Name: "Test 2", Status: Status{Status: "PASS", Critical: "no"}},
					},
				},
			},
			onlyCritical: false,
			countSkipped: false,
			expectedStats: StatsResult{
				TotalTests:     2,
				PassedTests:    2,
				FailedTests:    0,
				SkippedTests:   0,
				TotalCritical:  1,
				CriticalPassed: 1,
				CriticalFailed: 0,
				FailureRate:    0,
				SkippedRate:    0,
			},
		},
		{
			name: "Some tests fail",
			robotOutput: RobotOutput{
				Suite: Suite{
					Tests: []Test{
						{Name: "Test 1", Status: Status{Status: "FAIL", Critical: "yes"}},
						{Name: "Test 2", Status: Status{Status: "PASS", Critical: "no"}},
						{Name: "Test 3", Status: Status{Status: "FAIL", Critical: "no"}},
					},
				},
			},
			onlyCritical: false,
			countSkipped: false,
			expectedStats: StatsResult{
				TotalTests:     3,
				PassedTests:    1,
				FailedTests:    2,
				SkippedTests:   0,
				TotalCritical:  1,
				CriticalPassed: 0,
				CriticalFailed: 1,
				FailureRate:    66.67,
				SkippedRate:    0,
			},
		},
		{
			name: "Only critical tests counted",
			robotOutput: RobotOutput{
				Suite: Suite{
					Tests: []Test{
						{Name: "Test 1", Status: Status{Status: "PASS", Critical: "yes"}},
						{Name: "Test 2", Status: Status{Status: "FAIL", Critical: "no"}},
						{Name: "Test 3", Status: Status{Status: "FAIL", Critical: "yes"}},
					},
				},
			},
			onlyCritical: true,
			countSkipped: false,
			expectedStats: StatsResult{
				TotalTests:     2,
				PassedTests:    1,
				FailedTests:    1,
				SkippedTests:   0,
				TotalCritical:  2,
				CriticalPassed: 1,
				CriticalFailed: 1,
				FailureRate:    50,
				SkippedRate:    0,
			},
		},
		{
			name: "Skipped tests counted",
			robotOutput: RobotOutput{
				Suite: Suite{
					Tests: []Test{
						{Name: "Test 1", Status: Status{Status: "PASS", Critical: "yes"}},
						{Name: "Test 2", Status: Status{Status: "SKIP", Critical: "no"}},
						{Name: "Test 3", Status: Status{Status: "FAIL", Critical: "yes"}},
					},
				},
			},
			onlyCritical: false,
			countSkipped: true,
			expectedStats: StatsResult{
				TotalTests:     3,
				PassedTests:    1,
				FailedTests:    1,
				SkippedTests:   1,
				TotalCritical:  2,
				CriticalPassed: 1,
				CriticalFailed: 1,
				FailureRate:    33.33,
				SkippedRate:    33.33,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stats := computeStats(tc.robotOutput, tc.onlyCritical, tc.countSkipped)

			// Validate results
			if stats.TotalTests != tc.expectedStats.TotalTests {
				t.Errorf("Expected TotalTests: %d, got: %d", tc.expectedStats.TotalTests, stats.TotalTests)
			}
			if stats.PassedTests != tc.expectedStats.PassedTests {
				t.Errorf("Expected PassedTests: %d, got: %d", tc.expectedStats.PassedTests, stats.PassedTests)
			}
			if stats.FailedTests != tc.expectedStats.FailedTests {
				t.Errorf("Expected FailedTests: %d, got: %d", tc.expectedStats.FailedTests, stats.FailedTests)
			}
			if stats.SkippedTests != tc.expectedStats.SkippedTests {
				t.Errorf("Expected SkippedTests: %d, got: %d", tc.expectedStats.SkippedTests, stats.SkippedTests)
			}
			if stats.TotalCritical != tc.expectedStats.TotalCritical {
				t.Errorf("Expected TotalCritical: %d, got: %d", tc.expectedStats.TotalCritical, stats.TotalCritical)
			}
			if stats.CriticalPassed != tc.expectedStats.CriticalPassed {
				t.Errorf("Expected CriticalPassed: %d, got: %d", tc.expectedStats.CriticalPassed, stats.CriticalPassed)
			}
			if stats.CriticalFailed != tc.expectedStats.CriticalFailed {
				t.Errorf("Expected CriticalFailed: %d, got: %d", tc.expectedStats.CriticalFailed, stats.CriticalFailed)
			}
			// Update the test case
			if !almostEqual(stats.FailureRate, tc.expectedStats.FailureRate, 0.01) {
				t.Errorf("Expected FailureRate: %.2f, got: %.2f", tc.expectedStats.FailureRate, stats.FailureRate)
			}

			if !almostEqual(stats.SkippedRate, tc.expectedStats.SkippedRate, 0.01) {
				t.Errorf("Expected SkippedRate: %.2f, got: %.2f", tc.expectedStats.SkippedRate, stats.SkippedRate)
			}
		})
	}
}

// Helper function to compare floating-point numbers
func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) <= epsilon
}
