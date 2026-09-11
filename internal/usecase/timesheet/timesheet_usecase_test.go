package timesheetusecase

import (
	"context"
	"testing"
	"time"

	timesheetstorage "cashflow_backend/internal/adapters/storage/timesheet"
)

func TestTimerStopCreatesTimesheetEntry(t *testing.T) {
	repo := timesheetstorage.NewMemoryRepo()
	useCase := New(repo)
	start := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	if _, err := useCase.StartTimer(context.Background(), 9, 2, start); err != nil {
		t.Fatalf("start timer: %v", err)
	}
	entry, err := useCase.StopTimer(context.Background(), 2, start.Add(2*time.Hour), 3, 4, 1, 50)
	if err != nil {
		t.Fatalf("stop timer: %v", err)
	}
	if entry.UnitAmount != 2 || entry.AmountTotalCost != 100 {
		t.Fatalf("unexpected timer entry: %+v", entry)
	}
}
