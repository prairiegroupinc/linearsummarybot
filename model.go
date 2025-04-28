package main

import (
	"sort"

	"github.com/prairiegroupinc/linearsummarybot/yearmonth"
)

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
	YearMonth  yearmonth.YM
	InitName   string // empty if orphaned
	URL        string
	Bucket     string
	Labels     []string
	Clients    []string
}

type MonthData struct {
	Name        string
	Key         yearmonth.YM
	Initiatives map[string]*InitiativeData
	Config      *MonthConfig
	IsPast      bool

	Capacity int

	// Cached calculations
	Fixed   int
	Planned int
	Flex    int
	Used    int
	Total   int

	// Cached sorting
	SortedInitiatives []*InitiativeData
}

func (md *MonthData) RemainingBudget() int {
	return md.Capacity - md.Total
}

func (md *MonthData) IsOverCapacity() bool {
	return md.RemainingBudget() < 0
}

func (md *MonthData) LookupInitiative(name string) *InitiativeData {
	idata, ok := md.Initiatives[name]
	if !ok {
		idata = &InitiativeData{
			Name:   name,
			Issues: make([]*IssueData, 0),
		}
		md.Initiatives[name] = idata
	}
	return idata
}

type InitiativeData struct {
	Name    string
	Fixed   int
	Planned int
	Flex    int
	Total   int
	Used    int
	Budget  int
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
