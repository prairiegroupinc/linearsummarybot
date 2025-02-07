package main

import (
	"testing"
	"time"
)

func TestGetIssueTargetDate(t *testing.T) {
	// Helper to create time.Time values
	date := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name       string
		cycleStart *time.Time
		cycleMid   *time.Time
		cycleEnd   *time.Time
		deadline   *time.Time
		want       time.Time
		wantError  bool
	}{
		// No cycle cases
		{
			name:     "no cycle, has deadline - use deadline",
			deadline: &[]time.Time{date(2025, 1, 15)}[0],
			want:     date(2025, 1, 15),
		},
		{
			name: "no cycle, no deadline - empty time",
			want: time.Time{},
		},

		// Only cycle cases
		{
			name:       "cycle only - use mid-cycle",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			want:       date(2025, 2, 13),
		},

		// Deadline within cycle cases
		{
			name:       "deadline within cycle and before mid-cycle - use deadline",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			deadline:   &[]time.Time{date(2025, 2, 12)}[0],
			want:       date(2025, 2, 12),
		},
		{
			name:       "deadline within cycle but after mid-cycle - use mid-cycle",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			deadline:   &[]time.Time{date(2025, 2, 14)}[0],
			want:       date(2025, 2, 13),
		},

		// Deadline outside cycle cases
		{
			name:       "deadline before cycle start - use mid-cycle",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			deadline:   &[]time.Time{date(2025, 1, 15)}[0],
			want:       date(2025, 2, 13),
		},
		{
			name:       "deadline after cycle end - use mid-cycle",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			deadline:   &[]time.Time{date(2025, 3, 15)}[0],
			want:       date(2025, 2, 13),
		},

		// Edge cases
		{
			name:       "deadline exactly at cycle start - use deadline",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			deadline:   &[]time.Time{date(2025, 2, 10)}[0],
			want:       date(2025, 2, 10),
		},
		{
			name:       "deadline exactly at cycle end - use mid-cycle",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			deadline:   &[]time.Time{date(2025, 2, 16)}[0],
			want:       date(2025, 2, 13),
		},
		{
			name:       "deadline exactly at mid-cycle - use mid-cycle",
			cycleStart: &[]time.Time{date(2025, 2, 10)}[0],
			cycleMid:   &[]time.Time{date(2025, 2, 13)}[0],
			cycleEnd:   &[]time.Time{date(2025, 2, 16)}[0],
			deadline:   &[]time.Time{date(2025, 2, 13)}[0],
			want:       date(2025, 2, 13),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getIssueTargetDate(tt.cycleStart, tt.cycleMid, tt.cycleEnd, tt.deadline)
			if !got.Equal(tt.want) {
				t.Errorf("getIssueTargetDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
