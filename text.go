package main

import (
	"fmt"
	"strings"
)

func formatTextReport(report *Report) string {
	var sb strings.Builder

	// Print header
	fmt.Fprintf(&sb, "%-45s %5s %5s %5s %5s\n", "", "Total", "Fixed", "Sched", "Flex")
	sb.WriteString("---------------------------------------------------------------------\n")

	// Print each month
	for _, md := range report.Months {
		// Print month row
		fmt.Fprintf(&sb, "%-45s %5d %5d %5d %5d\n", strings.ToUpper(md.Name), md.Total, md.Fixed, md.Planned, md.Flex)

		// Print each initiative
		for _, idata := range md.SortedInitiatives {
			fmt.Fprintf(&sb, "%-45s %5d %5d %5d %5d\n", idata.Name, idata.Total, idata.Fixed, idata.Planned, idata.Flex)
		}

		sb.WriteString("---------------------------------------------------------------------\n")
	}

	// Print orphaned issues if any exist
	hasOrphans := false
	for _, md := range report.Months {
		if other, ok := md.Initiatives["Other"]; ok && len(other.Issues) > 0 {
			hasOrphans = true
			break
		}
	}

	if hasOrphans {
		sb.WriteString("\n\nIssues without a project:\n")
		for _, md := range report.Months {
			other, ok := md.Initiatives["Other"]
			if !ok || len(other.Issues) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "\n%s:\n", strings.ToUpper(md.Name))
			// Print sorted issues
			for _, issue := range other.Issues {
				fmt.Fprintf(&sb, "  [%2d] %s: %s\n", issue.Points, issue.Identifier, issue.Title)
			}
		}
		sb.WriteString("---------------------------------------------------------------------\n")
	}

	return sb.String()
}

func buildReport() (*Report, error) {
	issues, err := fetchLinearIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %v", err)
	}

	report, err := computeReport(issues)
	if err != nil {
		return nil, fmt.Errorf("failed to compute report: %v", err)
	}

	return report, nil
}
