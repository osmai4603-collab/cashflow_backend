package timesheet

import (
	"testing"
	"time"
)

func TestEntryApprovalLifecycleAndCost(t *testing.T) {
	entry := &Entry{ProjectID: 1, EmployeeID: 2, UserID: 3, Date: time.Now(), UnitAmount: 2.5, Name: "Implementation", HourlyCost: 40, CompanyID: 1}
	if err := entry.Validate(); err != nil {
		t.Fatalf("validate entry: %v", err)
	}
	if entry.AmountTotalCost != 100 {
		t.Fatalf("unexpected cost: %v", entry.AmountTotalCost)
	}
	if err := entry.Submit(); err != nil {
		t.Fatalf("submit entry: %v", err)
	}
	if err := entry.Approve(); err != nil {
		t.Fatalf("approve entry: %v", err)
	}
}

func TestTaskTimerStopReturnsHours(t *testing.T) {
	start := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	timer := &TaskTimer{TaskID: 1, EmployeeID: 2}
	if err := timer.Start(start); err != nil {
		t.Fatalf("start timer: %v", err)
	}
	hours, err := timer.Stop(start.Add(90 * time.Minute))
	if err != nil || hours != 1.5 || timer.IsRunning {
		t.Fatalf("unexpected timer result: hours=%v running=%v error=%v", hours, timer.IsRunning, err)
	}
}
