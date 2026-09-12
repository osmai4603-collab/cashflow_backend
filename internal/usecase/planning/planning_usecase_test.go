package planningusecase

import (
	"context"
	"testing"
	"time"

	planningstorage "cashflow_backend/internal/adapters/storage/planning"
	"cashflow_backend/internal/domain/planning"
)

func TestPlanningUseCase_CreateRoleAndShift(t *testing.T) {
	repo := planningstorage.NewMemoryRepo()
	uc := New(repo)
	ctx := context.Background()

	role, err := uc.CreateRole(ctx, &planning.PlanningRole{
		Name:      "Supervisor",
		Color:     "#ff0000",
		CompanyID: 7,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if role.ID == 0 {
		t.Fatal("role ID was not assigned")
	}

	base := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	shift, err := uc.CreateShift(ctx, &planning.PlanningShift{
		EmployeeID:     int64Ptr(11),
		RoleID:         role.ID,
		StartAt:        base,
		EndAt:          base.Add(8 * time.Hour),
		AllocatedHours: 8,
		CompanyID:      7,
	})
	if err != nil {
		t.Fatalf("create shift: %v", err)
	}
	if shift.ID == 0 {
		t.Fatal("shift ID was not assigned")
	}
	if shift.AllocatedHours != 8 {
		t.Fatalf("expected 8 allocated hours, got %v", shift.AllocatedHours)
	}

	list, err := uc.ListShifts(ctx, 7)
	if err != nil {
		t.Fatalf("list shifts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 shift, got %d", len(list))
	}
}

func int64Ptr(v int64) *int64 { return &v }
