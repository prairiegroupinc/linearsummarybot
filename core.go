package main

import (
	"cmp"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/prairiegroupinc/linearsummarybot/yearmonth"
)

func makeIssue(issue LinearIssue) *IssueData {
	// Skip issues with 0 points
	points := 0
	if issue.Estimate != nil {
		points = *issue.Estimate
	}
	if points == 0 {
		return nil
	}

	// Compute month info
	hasCycle := (issue.Cycle != nil)
	hasDeadline := (issue.DueDate != nil && *issue.DueDate != "")
	if !hasCycle && !hasDeadline {
		return nil
	}

	var deadline *time.Time
	if hasDeadline {
		if d, err := parseDateString(*issue.DueDate); err == nil {
			deadline = &d
		}
	}

	var cycleStartTime, cycleEndTime, cycleMidTime *time.Time
	if hasCycle {
		start, err1 := parseDateString(issue.Cycle.StartsAt)
		end, err2 := parseDateString(issue.Cycle.EndsAt)
		if err1 == nil && err2 == nil {
			mid := start.Add(end.Sub(start) / 2)

			cycleStartTime = &start
			cycleMidTime = &mid
			cycleEndTime = &end
		}
	}

	targetDate := getIssueTargetDate(cycleStartTime, cycleMidTime, cycleEndTime, deadline)
	if targetDate.IsZero() {
		return nil
	}
	monthName := targetDate.Format("January 2006")

	// Compute schedule
	schedule := Unscheduled
	if issue.Cycle != nil {
		// Parse cycle end date
		cycleEnd, err := parseDateString(issue.Cycle.EndsAt)
		if err == nil && issue.DueDate != nil {
			dueDate, err := parseDateString(*issue.DueDate)
			if err == nil {
				// Fixed if due date is within 14 days of cycle end
				if dueDate.Before(cycleEnd.Add(14 * 24 * time.Hour)) {
					schedule = Fixed
				} else {
					schedule = Planned
				}
			}
		} else {
			schedule = Planned
		}
	} else if issue.DueDate != nil {
		schedule = Flex
	}

	result := &IssueData{
		Identifier: issue.Identifier,
		Title:      issue.Title,
		Points:     points,
		Schedule:   schedule,
		MonthName:  monthName,
		YearMonth:  yearmonth.FromTime(targetDate),
		URL:        issue.URL,
	}

	for _, label := range issue.Labels.Nodes {
		tag := label.Name
		result.Labels = append(result.Labels, tag)
		if s, ok := strings.CutPrefix(tag, "Client-"); ok {
			result.Clients = append(result.Clients, s)
		}
		if s := config.TagsToBuckets[tag]; s != "" {
			result.Bucket = s
		}
	}

	// Compute initiative name
	result.InitName = "Other"
	if issue.Project != nil && len(issue.Project.Initiatives.Nodes) > 0 {
		result.InitName = issue.Project.Initiatives.Nodes[0].Name
	} else if issue.Project != nil && issue.Project.Name != "" {
		result.InitName = issue.Project.Name
	}
	if result.Bucket != "" {
		result.InitName = result.Bucket
	}

	return result
}

// getIssueTargetDate computes the target date for an issue based on its cycle and deadline.
// The rules are:
// 1. If it has a cycle and deadline:
//   - If the deadline is within the cycle and before mid-cycle, use the deadline
//   - Otherwise use the mid-cycle date
//
// 2. If it only has a deadline, use the deadline
// 3. If it only has a cycle, use the mid-cycle date
func getIssueTargetDate(cycleStart, cycleMid, cycleEnd, deadline *time.Time) time.Time {
	if cycleMid != nil {
		// Has cycle - only use deadline if it's within cycle and before mid-cycle
		if deadline != nil && cycleStart != nil && cycleEnd != nil {
			if !deadline.Before(*cycleStart) && !deadline.After(*cycleEnd) && deadline.Before(*cycleMid) {
				return *deadline
			}
		}
		return *cycleMid
	}
	// No cycle - use deadline
	if deadline != nil {
		return *deadline
	}
	return time.Time{}
}

func computeReport(issues []LinearIssue) (*Report, error) {
	// First convert all issues
	wrappedIssues := make([]*IssueData, 0, len(issues))
	for _, issue := range issues {
		if _, ok := StatesToSkip[issue.State.Name]; ok {
			continue
		}
		if wrapped := makeIssue(issue); wrapped != nil {
			wrappedIssues = append(wrappedIssues, wrapped)
		}
	}

	currentMonth := yearmonth.FromTime(time.Now().UTC())

	// Group by month
	monthData := make(map[yearmonth.YM]*MonthData)

	for _, issue := range wrappedIssues {
		md, ok := monthData[issue.YearMonth]
		if !ok {
			md = &MonthData{
				Name:        issue.MonthName,
				Key:         issue.YearMonth,
				IsPast:      issue.YearMonth < currentMonth,
				Initiatives: make(map[string]*InitiativeData),
			}
			md.Config = config.ByMonth[md.Key]
			if md.Config == nil {
				md.Config = &MonthConfig{}
			}
			md.Capacity = md.Config.Capacity
			if md.Capacity == 0 {
				md.Capacity = config.DefaultCapacity
			}
			for bucket := range md.Config.Budget {
				_ = md.LookupInitiative(bucket)
			}
			monthData[issue.YearMonth] = md
		}

		// Add to initiatives ("Other" for orphans)
		idata := md.LookupInitiative(issue.InitName)

		// Store the issue
		idata.Issues = append(idata.Issues, issue)

		switch issue.Schedule {
		case Fixed:
			idata.Fixed += issue.Points
		case Planned:
			idata.Planned += issue.Points
		case Flex:
			idata.Flex += issue.Points
		}
		idata.Total = idata.Fixed + idata.Planned + idata.Flex
	}

	// Get sorted slice of months
	monthSlice := slices.Collect(maps.Values(monthData))

	// Sort months by key
	slices.SortFunc(monthSlice, func(a, b *MonthData) int {
		return cmp.Compare(a.Key, b.Key)
	})

	// Calculate totals and sort initiatives within each month
	for _, md := range monthSlice {
		// Calculate month totals from initiatives
		for _, idata := range md.Initiatives {
			idata.Budget = md.Config.Budget[idata.Name]

			idata.Used = idata.Fixed + idata.Planned + idata.Flex
			idata.Total = max(idata.Budget, idata.Used)

			md.Fixed += idata.Fixed
			md.Planned += idata.Planned
			md.Flex += idata.Flex
			md.Used += idata.Used
			md.Total += idata.Total
		}

		// Get sorted slice of initiatives
		initSlice := slices.Collect(maps.Values(md.Initiatives))

		// Sort initiatives by total points (descending)
		slices.SortFunc(initSlice, func(a, b *InitiativeData) int {
			return b.Total - a.Total
		})

		// Store sorted initiatives
		md.SortedInitiatives = initSlice

		// Sort issues within each initiative
		for _, idata := range initSlice {
			idata.sortIssues()
		}
	}

	return &Report{Months: monthSlice}, nil
}
