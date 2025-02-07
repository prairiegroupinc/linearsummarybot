package main

import "sort"

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
	URL        string
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

// sortIssues sorts Issues by points (descending) and identifier (ascending)
func (i *InitiativeData) sortIssues() {
	sort.Slice(i.Issues, func(a, b int) bool {
		if i.Issues[a].Points != i.Issues[b].Points {
			return i.Issues[a].Points > i.Issues[b].Points // descending
		}
		return i.Issues[a].Identifier < i.Issues[b].Identifier // ascending
	})
}

type Schedule int

const (
	Unscheduled Schedule = iota
	Fixed
	Planned
	Flex
)
