package main

// Report represents a complete summary of all issues organized by month
type Report struct {
	Months []*MonthData
}

type IssueData struct {
	Identifier string
	Title      string
	Points     int
	Schedule   Schedule
	MonthName  string
	MonthKey   int
	InitName   string // empty if orphaned
}

type MonthData struct {
	Name        string
	Key         int // YYYYMM format
	Initiatives map[string]*InitiativeData

	// Cached calculations
	Fixed   int
	Planned int
	Flex    int
	Total   int

	// Cached sorting
	SortedInitiatives []*InitiativeData
}

type InitiativeData struct {
	Name    string
	Fixed   int
	Planned int
	Flex    int
	Total   int
	Issues  []*IssueData
}

type Schedule int

const (
	Unscheduled Schedule = iota
	Fixed
	Planned
	Flex
)
