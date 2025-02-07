package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func newMockReport() *Report {
	// Create a sample report with a mix of initiatives and orphaned issues
	agMVP := &InitiativeData{
		Name:    "AG MVP",
		Fixed:   80,
		Planned: 1,
		Total:   81,
		Issues: []*IssueData{
			{
				Identifier: "DEV-123",
				Title:      "Implement feature X",
				Points:     5,
				Schedule:   Fixed,
			},
		},
	}

	other := &InitiativeData{
		Name:    "Other",
		Fixed:   1,
		Total:   1,
		Issues: []*IssueData{
			{
				Identifier: "DEV-225",
				Title:      "Refresh page after adding vendible to cart",
				Points:     1,
				Schedule:   Fixed,
			},
		},
	}

	month := &MonthData{
		Name: "February 2025",
		Key:  202502,
		Initiatives: map[string]*InitiativeData{
			"AG MVP": agMVP,
			"Other":  other,
		},
		SortedInitiatives: []*InitiativeData{agMVP, other},
		Fixed:            81,
		Planned:          1,
		Total:            82,
	}

	return &Report{
		Months: []*MonthData{month},
	}
}

func TestServeSpecificHTMLReport(t *testing.T) {
	// Create a mock report
	report := newMockReport()

	// Create a response recorder
	w := httptest.NewRecorder()

	// Call the handler
	err := serveSpecificHTMLReport(w, report)
	if err != nil {
		t.Fatalf("Failed to serve HTML report: %v", err)
	}

	// Check response
	response := w.Body.String()

	// Basic checks for expected content
	expectedStrings := []string{
		"February 2025",
		"AG MVP",
		"DEV-123",
		"Implement feature X",
		"DEV-225",
		"Refresh page after adding vendible to cart",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(response, s) {
			t.Errorf("Response missing expected string: %q", s)
		}
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Wrong content type, got %q, want text/html; charset=utf-8", contentType)
	}
}
