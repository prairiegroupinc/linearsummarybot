package main

import (
	"maps"
	"slices"
	"time"
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
	monthKey := targetDate.Year()*100 + int(targetDate.Month())

	// Compute initiative name
	initName := "Other"
	if issue.Project != nil && len(issue.Project.Initiatives.Nodes) > 0 {
		initName = issue.Project.Initiatives.Nodes[0].Name
	} else if issue.Project != nil && issue.Project.Name != "" {
		initName = issue.Project.Name
	}

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
		MonthKey:   monthKey,
		InitName:   initName,
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
		if wrapped := makeIssue(issue); wrapped != nil {
			wrappedIssues = append(wrappedIssues, wrapped)
		}
	}

	// Group by month
	monthData := make(map[string]*MonthData)

	for _, issue := range wrappedIssues {
		md, ok := monthData[issue.MonthName]
		if !ok {
			md = &MonthData{
				Name:        issue.MonthName,
				Key:         issue.MonthKey,
				Initiatives: make(map[string]*InitiativeData),
			}
			monthData[issue.MonthName] = md
		}

		// Add to initiatives ("Other" for orphans)
		idata, ok := md.Initiatives[issue.InitName]
		if !ok {
			idata = &InitiativeData{
				Name:   issue.InitName,
				Issues: make([]*IssueData, 0),
			}
			md.Initiatives[issue.InitName] = idata
		}

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
		return a.Key - b.Key
	})

	// Calculate totals and sort initiatives within each month
	for _, md := range monthSlice {
		// Calculate month totals from initiatives
		for _, idata := range md.Initiatives {
			md.Fixed += idata.Fixed
			md.Planned += idata.Planned
			md.Flex += idata.Flex
		}
		md.Total = md.Fixed + md.Planned + md.Flex

		// Get sorted slice of initiatives
		initSlice := slices.Collect(maps.Values(md.Initiatives))

		// Sort initiatives by total points (descending)
		slices.SortFunc(initSlice, func(a, b *InitiativeData) int {
			return b.Total - a.Total
		})

		// Store sorted initiatives
		md.SortedInitiatives = initSlice
	}

	return &Report{Months: monthSlice}, nil
}
