package main

import (
	"time"
)

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
