package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

func buildReport() (string, error) {
	issues, err := fetchLinearIssues()
	if err != nil {
		return "", err
	}

	// We'll store data by month
	monthData := make(map[string]*MonthData)

	for _, issue := range issues {
		// Decide which month it belongs to
		monthName, monthKey := getIssueMonth(issue)
		if monthName == "" {
			continue
		}

		// Decide if this is fixed, planned, or flex
		points := 0
		if issue.Estimate != nil {
			points = *issue.Estimate
		}
		if points == 0 {
			continue
		}

		md, ok := monthData[monthName]
		if !ok {
			md = &MonthData{
				Name:        monthName,
				Key:         monthKey,
				Initiatives: make(map[string]*InitiativeData),
				Orphans:     make([]LinearIssue, 0),
			}
			monthData[monthName] = md
		}

		// Identify the initiative name by checking the project's first initiative
		initName := "Other"
		if issue.Project != nil && len(issue.Project.Initiatives.Nodes) > 0 {
			initName = issue.Project.Initiatives.Nodes[0].Name
		} else if issue.Project != nil && issue.Project.Name != "" {
			initName = issue.Project.Name
		} else {
			md.Orphans = append(md.Orphans, issue)
		}

		idata, ok := md.Initiatives[initName]
		if !ok {
			idata = &InitiativeData{Name: initName}
			md.Initiatives[initName] = idata
		}
		if issue.Cycle != nil {
			// Parse cycle end date
			cycleEnd, err := parseDateString(issue.Cycle.EndsAt)
			if err != nil {
				return "", fmt.Errorf("failed to parse cycle end date: %v", err)
			}

			// Check if there's a deadline within 14 days of cycle end
			isFixed := false
			if issue.DueDate != nil {
				dueDate, err := parseDateString(*issue.DueDate)
				if err != nil {
					return "", fmt.Errorf("failed to parse due date: %v", err)
				}
				// Fixed if due date is within 14 days of cycle end
				isFixed = dueDate.Before(cycleEnd.Add(14 * 24 * time.Hour))
			}

			if isFixed {
				idata.Fixed += points
			} else {
				idata.Planned += points
			}
		} else {
			idata.Flex += points
		}
		idata.Total = idata.Fixed + idata.Planned + idata.Flex
	}

	// Get sorted slice of months
	monthSlice := slices.Collect(maps.Values(monthData))

	// Sort months by key
	slices.SortFunc(monthSlice, func(a, b *MonthData) int {
		return a.Key - b.Key
	})

	var sb strings.Builder

	const sep = "---------------------------------------------------------------------\n"

	// Print table header
	fmt.Fprintf(&sb, "%-45s %5s %5s %5s %5s\n", "", "Total", "Fixed", "Sched", "Flex")
	sb.WriteString(sep)

	for _, md := range monthSlice {

		// Calculate month totals from initiatives
		var fixed, planned, flex int
		for _, idata := range md.Initiatives {
			fixed += idata.Fixed
			planned += idata.Planned
			flex += idata.Flex
		}
		total := fixed + planned + flex
		fmt.Fprintf(&sb, "%-45s %5d %5d %5d %5d\n", strings.ToUpper(md.Name), total, fixed, planned, flex)

		// Get sorted slice of initiatives
		initSlice := slices.Collect(maps.Values(md.Initiatives))

		// Sort initiatives by total points (descending)
		slices.SortFunc(initSlice, func(a, b *InitiativeData) int {
			return b.Total - a.Total
		})

		// Print each initiative row
		for _, idata := range initSlice {
			fmt.Fprintf(&sb, "%-45s %5d %5d %5d %5d\n", idata.Name, idata.Total, idata.Fixed, idata.Planned, idata.Flex)
		}

		// Month separator
		sb.WriteString(sep)
	}

	// Print orphaned issues if any exist
	hasOrphans := false
	for _, md := range monthSlice {
		if len(md.Orphans) > 0 {
			hasOrphans = true
			break
		}
	}

	if hasOrphans {
		sb.WriteString("\n\nIssues without a project:\n")
		for _, md := range monthSlice {
			orphans := md.Orphans
			if len(orphans) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "\n%s:\n", strings.ToUpper(md.Name))
			// Sort orphaned issues
			// Sort by points (desc) and identifier
			slices.SortFunc(orphans, func(a, b LinearIssue) int {
				aPoints := 0
				if a.Estimate != nil {
					aPoints = *a.Estimate
				}
				bPoints := 0
				if b.Estimate != nil {
					bPoints = *b.Estimate
				}
				if aPoints != bPoints {
					return bPoints - aPoints // descending
				}
				return strings.Compare(a.Identifier, b.Identifier)
			})
			// Print sorted issues
			for _, issue := range orphans {
				points := *issue.Estimate // safe because we filtered nil/0 above
				fmt.Fprintf(&sb, "  [%2d] %s - %s\n", points, issue.Identifier, issue.Title)
			}
		}
		sb.WriteString(sep)
	}

	return sb.String(), nil
}
