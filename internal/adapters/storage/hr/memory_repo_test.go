package hrstorage_test

import (
	"context"
	"testing"
	"time"

	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_Departments(t *testing.T) {
	repo := hrstorage.NewMemoryRepo()
	ctx := context.Background()

	dept := &hr.Department{
		Name:   "Product Development",
		Active: true,
	}

	if err := repo.CreateDepartment(ctx, dept); err != nil {
		t.Fatalf("failed to create department: %v", err)
	}
	if dept.ID == 0 {
		t.Errorf("expected non-zero ID")
	}

	fetched, err := repo.GetDepartmentByID(ctx, dept.ID)
	if err != nil {
		t.Fatalf("failed to get department: %v", err)
	}
	if fetched.Name != dept.Name {
		t.Errorf("got name %s, want %s", fetched.Name, dept.Name)
	}

	// Update
	dept.Name = "Product Engineering"
	if err := repo.UpdateDepartment(ctx, dept); err != nil {
		t.Fatalf("failed to update department: %v", err)
	}

	// List
	res, err := repo.ListDepartments(ctx, nil, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("failed to list departments: %v", err)
	}
	if res.TotalItems < 1 {
		t.Errorf("expected at least 1 department, got %d", res.TotalItems)
	}
}

func TestMemoryRepo_EmployeesAndLeaves(t *testing.T) {
	repo := hrstorage.NewMemoryRepo()
	ctx := context.Background()

	// 1. Create employee
	emp := &hr.Employee{
		Name:      "Fatima Zahra",
		WorkEmail: "fatima@example.com",
		Active:    true,
	}
	if err := repo.CreateEmployee(ctx, emp); err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}

	// 2. Allocate leave
	alloc := &hr.LeaveAllocation{
		EmployeeID:    emp.ID,
		LeaveType:     hr.LeaveTypeAnnual,
		AllocatedDays: 21.0,
		Year:          2026,
		State:         hr.AllocationStateApproved,
	}
	if err := repo.CreateAllocation(ctx, alloc); err != nil {
		t.Fatalf("failed to create allocation: %v", err)
	}

	allocDays, err := repo.GetTotalAllocatedDays(ctx, emp.ID, hr.LeaveTypeAnnual, 2026)
	if err != nil || allocDays != 21.0 {
		t.Fatalf("got allocated days %v, err %v", allocDays, err)
	}

	// 3. Leave Request
	dFrom := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	dTo := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)
	leaveReq := &hr.LeaveRequest{
		EmployeeID: emp.ID,
		LeaveType:  hr.LeaveTypeAnnual,
		DateFrom:   dFrom,
		DateTo:     dTo,
		Days:       5.0,
		State:      hr.LeaveStateConfirm,
	}
	if err := repo.CreateLeaveRequest(ctx, leaveReq); err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	// Pending days
	pending, err := repo.GetPendingLeaveDays(ctx, emp.ID, hr.LeaveTypeAnnual, 2026)
	if err != nil || pending != 5.0 {
		t.Errorf("got pending days %v, want 5.0", pending)
	}

	// Overlap check
	overlap, err := repo.HasOverlappingLeave(ctx, emp.ID, dFrom.AddDate(0, 0, 2), dTo.AddDate(0, 0, 5), 0)
	if err != nil || !overlap {
		t.Errorf("expected overlap, got %v, err %v", overlap, err)
	}

	// Approve leave
	leaveReq.State = hr.LeaveStateValidate
	if err := repo.UpdateLeaveRequest(ctx, leaveReq); err != nil {
		t.Fatalf("failed to update leave request: %v", err)
	}

	approved, err := repo.GetApprovedLeaveDays(ctx, emp.ID, hr.LeaveTypeAnnual, 2026)
	if err != nil || approved != 5.0 {
		t.Errorf("got approved days %v, want 5.0", approved)
	}
}
