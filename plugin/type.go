package plugin

import "encoding/xml"

// RobotOutput represents the structure of Robot Framework's output.xml
type RobotOutput struct {
	XMLName xml.Name `xml:"robot"`
	Suite   Suite    `xml:"suite"`
	Errors  []Error  `xml:"errors>msg"`
}

// Suite represents a test suite, which contains tests and sub-suites.
type Suite struct {
	ID       string    `xml:"id,attr"`
	Name     string    `xml:"name,attr"`
	Source   string    `xml:"source,attr,omitempty"`
	Doc      string    `xml:"doc,omitempty"`
	Tests    []Test    `xml:"test"`
	Keywords []Keyword `xml:"kw"`
	Status   Status    `xml:"status"`
	Suites   []Suite   `xml:"suite"`
}

// Test represents a test case inside a suite.
type Test struct {
	ID       string    `xml:"id,attr"`
	Name     string    `xml:"name,attr"`
	Keywords []Keyword `xml:"kw"`
	Status   Status    `xml:"status"`
}

// Keyword represents a keyword inside a test case or suite.
type Keyword struct {
	Name      string    `xml:"name,attr"`
	Type      string    `xml:"type,attr,omitempty"` // Can be "setup", "teardown", etc.
	Library   string    `xml:"library,attr,omitempty"`
	Arguments []Arg     `xml:"arguments>arg"`
	Doc       string    `xml:"doc,omitempty"`
	Status    Status    `xml:"status"`
	Messages  []Msg     `xml:"msg"`
	Keywords  []Keyword `xml:"kw"`
}

// Status represents the execution status of a test, keyword, or suite.
type Status struct {
	Status    string `xml:"status,attr"`
	Critical  string `xml:"critical,attr,omitempty"` // Only present in test statuses
	StartTime string `xml:"starttime,attr,omitempty"`
	EndTime   string `xml:"endtime,attr,omitempty"`
	Messages  []Msg  `xml:"msg"`
}

// Arg represents arguments passed to a keyword.
type Arg struct {
	Value string `xml:",chardata"`
}

// Msg represents log messages inside a test or keyword.
type Msg struct {
	Timestamp string `xml:"timestamp,attr"`
	Level     string `xml:"level,attr"`
	Text      string `xml:",chardata"`
}

// Error represents errors in the test execution.
type Error struct {
	Message string `xml:",chardata"`
}

// StatsResult stores computed test statistics.
type StatsResult struct {
	TotalSuites        int
	TotalTests         int
	PassedTests        int
	FailedTests        int
	SkippedTests       int
	TotalKeywords      int
	PassedKeywords     int
	FailedKeywords     int
	SkippedKeywords    int
	TotalCritical      int
	CriticalPassed     int
	CriticalFailed     int
	FailureRate        float64
	SkippedRate        float64
	ExecutionTime      float64
	FailedTestsDetails []FailedTestDetails
}

// FailedTestDetails stores information about failed tests.
type FailedTestDetails struct {
	Name         string
	Suite        string
	Status       string
	ErrorMessage string
}
