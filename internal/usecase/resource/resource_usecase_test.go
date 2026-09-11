package resourceusecase

import (
	"context"
	"testing"
	"time"

	resourcestorage "cashflow_backend/internal/adapters/storage/resource"
)

func TestGenerateAttendanceWorkEntry(t *testing.T) {
	useCase := New(resourcestorage.NewMemoryRepo())
	start := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	entry, err := useCase.GenerateAttendanceEntry(context.Background(), 1, 1, start, start.Add(8*time.Hour))
	if err != nil || entry.DurationHours != 8 || entry.State != "validated" {
		t.Fatalf("unexpected work entry: %+v error=%v", entry, err)
	}
}
