package hrusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	companystorage "cashflow_backend/internal/adapters/storage/company"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/usecase/hr"
)

func setupAttendanceEnv(t *testing.T) (*hrusecase.AttendanceUseCase, *hrstorage.MemoryRepo, *companystorage.MemoryRepo) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hrRepo := hrstorage.NewMemoryRepo()
	companyRepo := companystorage.NewMemoryRepo()
	uc := hrusecase.NewAttendanceUseCase(hrRepo, companyRepo, logger)
	return uc, hrRepo, companyRepo
}

func TestAttendanceUseCase_CheckIn_CheckOut_Overtime(t *testing.T) {
	ctx := context.Background()
	uc, hrRepo, companyRepo := setupAttendanceEnv(t)

	// 1. Setup Company with Overtime Threshold (15 minutes)
	compID := int64(1)
	comp := &company.Company{
		ID:                       compID,
		Name:                     "Test Corp",
		OvertimeCompanyThreshold: 15, // 15 minutes
	}
	companyRepo.Create(ctx, comp)

	// 2. Setup Employee
	empID := int64(100)
	emp := &hr.Employee{
		ID:        empID,
		Name:      "Ahmed",
		CompanyID: &compID,
		Active:    true,
	}
	hrRepo.CreateEmployee(ctx, emp)

	// 3. Check-In
	t.Run("CheckIn Success", func(t *testing.T) {
		att, err := uc.CheckIn(ctx, empID, hr.Attendance{})
		if err != nil {
			t.Fatalf("CheckIn failed: %v", err)
		}
		if att.EmployeeID != empID || att.CheckOut != nil {
			t.Errorf("unexpected attendance state: %+v", att)
		}
	})

	t.Run("CheckIn Conflict", func(t *testing.T) {
		_, err := uc.CheckIn(ctx, empID, hr.Attendance{})
		if err == nil {
			t.Fatal("expected conflict error for double check-in, got nil")
		}
	})

	// 4. Check-Out with Overtime
	t.Run("CheckOut with Overtime (9 hours worked)", func(t *testing.T) {
		// Mock last attendance to be 9 hours ago
		lastAtt, _ := hrRepo.GetLastAttendance(ctx, empID)
		startTime := time.Now().Add(-9 * time.Hour).UTC()
		lastAtt.CheckIn = startTime
		hrRepo.UpdateAttendance(ctx, lastAtt)

		att, err := uc.CheckOut(ctx, empID, hr.Attendance{OutMode: hr.AttendanceModeManual})
		if err != nil {
			t.Fatalf("CheckOut failed: %v", err)
		}

		// 9 hours worked - 8 hours expected = 1 hour overtime
		if att.WorkedHours < 8.99 || att.WorkedHours > 9.01 {
			t.Errorf("expected ~9 worked hours, got %v", att.WorkedHours)
		}
		if att.OvertimeHours < 0.99 || att.OvertimeHours > 1.01 {
			t.Errorf("expected ~1 hour overtime, got %v", att.OvertimeHours)
		}
		if att.OvertimeStatus != hr.OvertimeStatusToApprove {
			t.Errorf("expected status to_approve, got %s", att.OvertimeStatus)
		}
	})
}

func TestAttendanceUseCase_OvertimeThreshold(t *testing.T) {
	ctx := context.Background()
	uc, hrRepo, companyRepo := setupAttendanceEnv(t)

	compID := int64(1)
	companyRepo.Create(ctx, &company.Company{
		ID:                       compID,
		OvertimeCompanyThreshold: 30, // 30 mins threshold
	})

	empID := int64(101)
	hrRepo.CreateEmployee(ctx, &hr.Employee{ID: empID, CompanyID: &compID, Active: true})

	// Case: 10 minutes overtime (below 30 mins threshold)
	uc.CheckIn(ctx, empID, hr.Attendance{})
	lastAtt, _ := hrRepo.GetLastAttendance(ctx, empID)
	lastAtt.CheckIn = time.Now().Add(-8*time.Hour - 10*time.Minute).UTC()
	hrRepo.UpdateAttendance(ctx, lastAtt)

	att, _ := uc.CheckOut(ctx, empID, hr.Attendance{})
	if att.OvertimeHours != 0 {
		t.Errorf("expected 0 overtime hours due to threshold, got %v", att.OvertimeHours)
	}
}
