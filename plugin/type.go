package plugin

// RobotOutput represents the structure of Robot Framework's output.xml
type RobotOutput struct {
	Suite  Suite   `xml:"suite"`
	Errors []Error `xml:"errors>msg"`
}

// Suite represents a test suite, which contains tests and sub-suites.
type Suite struct {
	Name     string    `xml:"name,attr"`
	Tests    []Test    `xml:"test"`
	Keywords []Keyword `xml:"kw"`
	Status   Status    `xml:"status"`
	Suites   []Suite   `xml:"suite"`
}

// Test represents a test case inside a suite.
type Test struct {
	Name     string    `xml:"name,attr"`
	Status   Status    `xml:"status"`
	Keywords []Keyword `xml:"kw"`
}

// Keyword represents a keyword inside a test case or suite.
type Keyword struct {
	Name   string `xml:"name,attr"`
	Status Status `xml:"status"`
}

type Status struct {
	Status    string `xml:"status,attr"`
	Critical  string `xml:"critical,attr"`
	StartTime string `xml:"starttime,attr"`
	EndTime   string `xml:"endtime,attr"`
	Messages  []Msg  `xml:"msg"`
}

type Msg struct {
	Text  string `xml:",chardata"`
	Level string `xml:"level,attr"`
}

// Error represents errors in the test execution.
type Error struct {
	Message string `xml:",chardata"`
}

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

type FailedTestDetails struct {
	Name         string
	Suite        string
	Status       string
	ErrorMessage string
}
