package main

// Report represents a complete summary of all issues organized by month
type Report struct {
	Months []*MonthData
}

type MonthData struct {
	Name        string
	Key         int // YYYYMM format
	Initiatives map[string]*InitiativeData
	Orphans     []LinearIssue
}

type InitiativeData struct {
	Name    string
	Fixed   int
	Planned int
	Flex    int
	Total   int
}
