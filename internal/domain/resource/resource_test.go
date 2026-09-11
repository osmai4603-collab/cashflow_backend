package resource

import (
	"testing"
	"time"
)

func TestWorkEntryCalculatesDuration(t *testing.T) {
	entry := &WorkEntry{Name: "Attendance", EmployeeID: 1, CompanyID: 1, DateStart: time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC), DateStop: time.Date(2026, 9, 11, 17, 0, 0, 0, time.UTC)}
	if err := entry.Validate(); err != nil {
		t.Fatalf("validate work entry: %v", err)
	}
	if entry.DurationHours != 8 {
		t.Fatalf("unexpected duration: %v", entry.DurationHours)
	}
}
