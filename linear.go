package main

import (
	"fmt"
	"os"

	"github.com/andreyvit/mvp/httpcall"
)

type LinearIssue struct {
	Id         string  `json:"id"`
	Identifier string  `json:"identifier"`
	Title      string  `json:"title"`
	Estimate   *int    `json:"estimate"`
	DueDate    *string `json:"dueDate"`
	URL        string  `json:"url"`
	State      struct {
		Name string `json:"name"`
	} `json:"state"`
	Cycle *struct {
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
	      url
		  state {
		    name
		  }
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
