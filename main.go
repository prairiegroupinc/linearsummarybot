package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andreyvit/mvp/httpcall"
)

// LinearIssue is a minimal struct for parsing GraphQL results
type LinearIssue struct {
	Id       string  `json:"id"`
	Title    string  `json:"title"`
	Estimate *int    `json:"estimate"`
	DueDate  *string `json:"dueDate"`
	Cycle    *struct {
		StartsAt string `json:"startsAt"`
		EndsAt   string `json:"endsAt"`
	} `json:"cycle"`
}

func main() {
	log.SetFlags(0)

	onceFlag := flag.Bool("once", false, "Run once on launch")
	httpAddr := flag.String("http", "", "Listen address for HTTP server, e.g. :8080")
	flag.Parse()

	// If -once, run and print.
	if *onceFlag {
		rep, err := buildReport()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		fmt.Println(rep)
		return
	}

	// If -http, serve the same result on GET /
	if *httpAddr != "" {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			rep, err := buildReport()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, rep)
		})
		log.Printf("Listening on %s", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, nil))
		return
	}

	// If neither flag is given, just print usage
	flag.Usage()
}

// buildReport is the main aggregator: loads issues, classifies them, sums points, formats result.
func buildReport() (string, error) {
	issues, err := fetchLinearIssues()
	if err != nil {
		return "", err
	}

	// We'll accumulate total points by "MonthName: sum"
	monthTotals := make(map[string]int)

	for _, issue := range issues {
		points := 0
		if issue.Estimate != nil {
			points = *issue.Estimate
		}

		// Figure out which month this belongs to, if any
		month := getIssueMonth(issue)
		if month == "" {
			// ignore
			continue
		}
		monthTotals[month] += points
	}

	// We want a stable month order: Jan, Feb, etc. We'll produce the lines for only the months found.
	// We'll parse them as times with year=some fixed year, gather unique months, then sort them.
	type mo struct {
		t  time.Time
		pm string
	}
	var allMonths []mo
	for m := range monthTotals {
		// parse as January, 2006 for sorting
		t, _ := time.Parse("January", m)
		allMonths = append(allMonths, mo{t: t, pm: m})
	}
	// sort by t.Month()
	for i := 0; i < len(allMonths); i++ {
		for j := i + 1; j < len(allMonths); j++ {
			if allMonths[j].t.Month() < allMonths[i].t.Month() {
				allMonths[i], allMonths[j] = allMonths[j], allMonths[i]
			}
		}
	}

	var sb strings.Builder
	for _, m := range allMonths {
		fmt.Fprintf(&sb, "%-10s %4d pts\n", m.pm+":", monthTotals[m.pm])
	}

	return sb.String(), nil
}

// fetchLinearIssues calls the Linear GraphQL API for all non-completed issues
func fetchLinearIssues() ([]LinearIssue, error) {
	linearToken := os.Getenv("LINEAR_API_KEY")
	if linearToken == "" {
		return nil, fmt.Errorf("please set LINEAR_API_KEY environment variable")
	}

	// We fetch issues that are not completed. We'll do that via GraphQL.
	// For simplicity, we only fetch first 250 open issues; if you have more, add pagination.
	query := `
	query {
	  issues(
	    first: 250
	    filter: {
	      completedAt: { null: true }
	    }
	  ) {
	    nodes {
	      id
	      title
	      estimate
	      dueDate
	      cycle {
	        startsAt
	        endsAt
	      }
	    }
	  }
	}`

	var out struct {
		Data struct {
			Issues struct {
				Nodes []LinearIssue `json:"nodes"`
			} `json:"issues"`
		} `json:"data"`
	}

	req := &httpcall.Request{
		Method:      "POST",
		BaseURL:     "https://api.linear.app",
		Path:        "/graphql",
		Headers:     map[string][]string{"Authorization": {"Bearer " + linearToken}},
		Input:       map[string]any{"query": query},
		OutputPtr:   &out,
		MaxAttempts: 3,
	}
	err := req.Do()
	if err != nil {
		return nil, err
	}
	return out.Data.Issues.Nodes, nil
}

// getIssueMonth implements the logic described:
//  1. if in a cycle, use the month of the cycle's midpoint
//     except if there's a deadline inside the cycle that is earlier than midpoint => use deadline's month
//  2. if not in a cycle but has a deadline => use that deadline's month
//  3. otherwise ignore => returns empty string
func getIssueMonth(issue LinearIssue) string {
	hasCycle := issue.Cycle != nil
	hasDeadline := (issue.DueDate != nil && *issue.DueDate != "")
	if !hasCycle && !hasDeadline {
		return ""
	}

	var cycleStart, cycleEnd, cycleMid time.Time
	if hasCycle {
		start, err1 := time.Parse(time.RFC3339, issue.Cycle.StartsAt)
		end, err2 := time.Parse(time.RFC3339, issue.Cycle.EndsAt)
		if err1 != nil || err2 != nil {
			// if we can't parse cycle times, treat as if no cycle
			hasCycle = false
		} else {
			cycleStart, cycleEnd = start.UTC(), end.UTC()
			cycleMid = cycleStart.Add(cycleEnd.Sub(cycleStart) / 2)
		}
	}

	var deadlineTime time.Time
	if hasDeadline {
		dt, err := time.Parse("2006-01-02", *issue.DueDate)
		if err == nil {
			deadlineTime = dt.UTC()
		} else {
			// if we can't parse due date, treat as if no deadline
			hasDeadline = false
		}
	}

	switch {
	case hasCycle && hasDeadline:
		// If the deadline is within cycle range and earlier than midpoint, use deadline's month.
		// If not, use midpoint's month.
		if !deadlineTime.Before(cycleStart) && !deadlineTime.After(cycleEnd) && deadlineTime.Before(cycleMid) {
			return deadlineTime.Format("January")
		}
		// otherwise use midpoint month
		return cycleMid.Format("January")

	case hasCycle && !hasDeadline:
		return cycleMid.Format("January")

	case !hasCycle && hasDeadline:
		return deadlineTime.Format("January")

	default:
		return "" // should not happen given checks above
	}
}
