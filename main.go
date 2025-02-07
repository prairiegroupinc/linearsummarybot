package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	// The only allowed non-stdlib import, as provided.
	"github.com/andreyvit/mvp/httpcall"
)

type LinearIssue struct {
	Id         string  `json:"id"`
	Identifier string  `json:"identifier"`
	Title      string  `json:"title"`
	Estimate   *int    `json:"estimate"`
	DueDate    *string `json:"dueDate"`
	Cycle      *struct {
		StartsAt string `json:"startsAt"`
		EndsAt   string `json:"endsAt"`
	} `json:"cycle"`
	Project *struct {
		Name        string `json:"name"`
		Initiatives struct {
			Nodes []struct {
				Name string `json:"name"`
			} `json:"nodes"`
		} `json:"initiatives"`
	} `json:"project"`
}

// We'll store data like:
// monthData[monthString] = &MonthData{Fixed: X, Flex: Y, Initiatives: map[initiativeName]*InitiativeData{...}}
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

func main() {
	log.SetFlags(0)

	onceFlag := flag.Bool("once", false, "Run once on launch")
	httpAddr := flag.String("http", "", "Listen address for HTTP server, e.g. :8080")
	flag.Parse()

	if *onceFlag {
		rep, err := buildReport()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		fmt.Println(rep)
		return
	}

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

	flag.Usage()
}

// buildReport loads the issues, categorizes by month and initiative, then prints.
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

// fetchPageOfLinearIssues calls Linear GraphQL to fetch a single page of issues.
// fetchLinearIssues fetches all non-completed issues from Linear, handling pagination automatically.
func fetchLinearIssues() ([]LinearIssue, error) {
	var allIssues []LinearIssue
	var after *string

	for {
		issues, endCursor, hasNextPage, err := fetchPageOfLinearIssues(after)
		if err != nil {
			return nil, fmt.Errorf("fetching page of issues: %w", err)
		}

		allIssues = append(allIssues, issues...)

		if !hasNextPage {
			break
		}
		after = &endCursor
	}

	return allIssues, nil
}

// fetchPageOfLinearIssues calls Linear GraphQL to fetch a single page of issues.
func fetchPageOfLinearIssues(after *string) ([]LinearIssue, string, bool, error) {
	linearToken := os.Getenv("LINEAR_API_KEY")
	if linearToken == "" {
		return nil, "", false, fmt.Errorf("please set LINEAR_API_KEY environment variable")
	}

	// We fetch non-completed issues (first 250), including project name and its first initiative
	query := `
	query($after: String) {
	  issues(
	    first: 250
	    after: $after
	    filter: {
	      state: { type: { nin: ["completed", "canceled"] } }
	    }
	  ) {
	    nodes {
	      id
	      identifier
	      title
	      estimate
	      dueDate
	      cycle {
	        startsAt
	        endsAt
	      }
	      project {
	        name
	        initiatives(first: 1) {
	          nodes {
	            name
	          }
	        }
	      }
	    }
	  }
	}`

	var out struct {
		Data struct {
			Issues struct {
				Nodes    []LinearIssue `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"issues"`
		} `json:"data"`
	}

	req := &httpcall.Request{
		Method:  "POST",
		BaseURL: "https://api.linear.app",
		Path:    "/graphql",
		Headers: map[string][]string{"Authorization": {"Bearer " + linearToken}},
		Input: map[string]any{
			"query":     query,
			"variables": map[string]any{"after": after},
		},
		OutputPtr:   &out,
		MaxAttempts: 3,
	}
	err := req.Do()
	if err != nil {
		return nil, "", false, err
	}
	return out.Data.Issues.Nodes, out.Data.Issues.PageInfo.EndCursor, out.Data.Issues.PageInfo.HasNextPage, nil
}

// getIssueMonth uses the logic: cycle midpoint unless there's an earlier in-cycle deadline.
// Returns month name and key (YYYYMM format)
func parseDateString(dateStr string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t, nil
	}
	// Try YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("failed to parse date: %s", dateStr)
}

func getIssueMonth(issue LinearIssue) (string, int) {
	hasCycle := (issue.Cycle != nil)
	hasDeadline := (issue.DueDate != nil && *issue.DueDate != "")
	if !hasCycle && !hasDeadline {
		return "", 0
	}

	var cycleStart, cycleEnd, cycleMid time.Time
	if hasCycle {
		start, err1 := time.Parse(time.RFC3339, issue.Cycle.StartsAt)
		end, err2 := time.Parse(time.RFC3339, issue.Cycle.EndsAt)
		if err1 == nil && err2 == nil {
			cycleStart, cycleEnd = start.UTC(), end.UTC()
			cycleMid = cycleStart.Add(cycleEnd.Sub(cycleStart) / 2)
		} else {
			hasCycle = false
		}
	}

	var deadlineTime time.Time
	if hasDeadline {
		dt, err := time.Parse("2006-01-02", *issue.DueDate)
		if err == nil {
			deadlineTime = dt.UTC()
		} else {
			hasDeadline = false
		}
	}

	var t time.Time
	switch {
	case hasCycle && hasDeadline:
		if !deadlineTime.Before(cycleStart) && !deadlineTime.After(cycleEnd) && deadlineTime.Before(cycleMid) {
			t = deadlineTime
		} else {
			t = cycleMid
		}
	case hasCycle && !hasDeadline:
		t = cycleMid
	case !hasCycle && hasDeadline:
		t = deadlineTime
	default:
		return "", 0
	}
	return t.Format("January 2006"), t.Year()*100 + int(t.Month())
}
