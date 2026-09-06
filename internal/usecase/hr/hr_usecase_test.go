package hrusecase_test

import (
	"context"
	"testing"
	"time"

	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/domain/partner"
	hrusecase "cashflow_backend/internal/usecase/hr"
)

type mockPartnerRepo struct {
	partners map[int64]*partner.Partner
	lastID   int64
}

func newMockPartnerRepo() *mockPartnerRepo {
	return &mockPartnerRepo{partners: make(map[int64]*partner.Partner)}
}

func (m *mockPartnerRepo) GetByID(ctx context.Context, id int64) (*partner.Partner, error) {
	p, ok := m.partners[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockPartnerRepo) Create(ctx context.Context, p *partner.Partner) error {
	m.lastID++
	p.ID = m.lastID
	m.partners[p.ID] = p
	return nil
}

func TestHRUseCase_DepartmentsAndJobs(t *testing.T) {
	repo := hrstorage.NewMemoryRepo()
	pRepo := newMockPartnerRepo()
	uc := hrusecase.New(repo, pRepo, nil)
	ctx := context.Background()

	// 1. Create parent department
	parent, err := uc.CreateDepartment(ctx, hrusecase.CreateDepartmentInput{
		Name: "Operations",
	})
	if err != nil {
		t.Fatalf("failed to create department: %v", err)
	}

	// 2. Create child department
	child, err := uc.CreateDepartment(ctx, hrusecase.CreateDepartmentInput{
		Name:     "Logistics",
		ParentID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("failed to create child department: %v", err)
	}
	if child.CompleteName != "Operations / Logistics" {
		t.Errorf("got complete name %s, want Operations / Logistics", child.CompleteName)
	}

	// 3. Tree check
	tree, err := uc.GetDepartmentTree(ctx, &parent.ID)
	if err != nil {
		t.Fatalf("failed to get department tree: %v", err)
	}
	if len(tree) != 1 || tree[0].ID != child.ID {
		t.Errorf("unexpected tree structure: %+v", tree)
	}

	// 4. Create Job
	job, err := uc.CreateJob(ctx, hrusecase.CreateJobInput{
		Name:              "Logistics Coordinator",
		DepartmentID:      &child.ID,
		ExpectedEmployees: 2,
	})
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	if job.ID == 0 {
		t.Errorf("expected non-zero job ID")
	}
}

func TestHRUseCase_EmployeeAndLeaveLifecycle(t *testing.T) {
	repo := hrstorage.NewMemoryRepo()
	pRepo := newMockPartnerRepo()
	uc := hrusecase.New(repo, pRepo, nil)
	ctx := context.Background()

	// 1. Create Employee with AutoCreatePartner
	emp, err := uc.CreateEmployee(ctx, hrusecase.CreateEmployeeInput{
		Name:              "Zaid Mansour",
		WorkEmail:         "zaid@example.com",
		AutoCreatePartner: true,
	})
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}
	if emp.PartnerID == nil {
		t.Errorf("expected auto-created partner ID")
	}

	// 2. Allocate Annual Leave (10 days for 2026)
	_, err = uc.CreateAllocation(ctx, hrusecase.CreateAllocationInput{
		EmployeeID:    emp.ID,
		LeaveType:     hr.LeaveTypeAnnual,
		AllocatedDays: 10,
		Year:          2026,
	})
	if err != nil {
		t.Fatalf("failed to create allocation: %v", err)
	}

	// 3. Check initial balance
	bal, err := uc.GetEmployeeLeaveBalance(ctx, emp.ID, 2026)
	if err != nil {
		t.Fatalf("failed to get leave balance: %v", err)
	}

	var annualBal *hr.LeaveBalance
	for i := range bal.Balances {
		if bal.Balances[i].LeaveType == hr.LeaveTypeAnnual {
			annualBal = &bal.Balances[i]
			break
		}
	}
	if annualBal == nil || annualBal.RemainingDays != 10 {
		t.Fatalf("unexpected annual balance: %+v", annualBal)
	}

	// 4. Try to request more days than allocated (12 days) -> Should fail on Approve
	dFrom1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	dTo1 := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	excessReq, err := uc.CreateLeaveRequest(ctx, hrusecase.CreateLeaveRequestInput{
		EmployeeID:  emp.ID,
		LeaveType:   hr.LeaveTypeAnnual,
		DateFrom:    dFrom1,
		DateTo:      dTo1,
		AutoConfirm: true,
	})
	if err != nil {
		t.Fatalf("failed to create excess leave request: %v", err)
	}

	_, err = uc.ApproveLeaveRequest(ctx, excessReq.ID, 1)
	if err == nil {
		t.Fatalf("expected error approving leave exceeding available balance")
	}

	// Cancel the excess request
	_, err = uc.CancelLeaveRequest(ctx, excessReq.ID)
	if err != nil {
		t.Fatalf("failed to cancel excess request: %v", err)
	}

	// 5. Valid Leave Request (3 days)
	dFrom2 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	dTo2 := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	validReq, err := uc.CreateLeaveRequest(ctx, hrusecase.CreateLeaveRequestInput{
		EmployeeID:  emp.ID,
		LeaveType:   hr.LeaveTypeAnnual,
		DateFrom:    dFrom2,
		DateTo:      dTo2,
		AutoConfirm: true,
	})
	if err != nil {
		t.Fatalf("failed to create valid leave request: %v", err)
	}

	// 6. Overlapping request check
	overlapReq, err := uc.CreateLeaveRequest(ctx, hrusecase.CreateLeaveRequestInput{
		EmployeeID: emp.ID,
		LeaveType:  hr.LeaveTypeSick,
		DateFrom:   dFrom2,
		DateTo:     dTo2,
	})
	if err == nil {
		t.Fatalf("expected conflict on overlapping leave dates, got req ID %d", overlapReq.ID)
	}

	// 7. Approve valid request
	approved, err := uc.ApproveLeaveRequest(ctx, validReq.ID, 1)
	if err != nil {
		t.Fatalf("failed to approve valid leave: %v", err)
	}
	if approved.State != hr.LeaveStateValidate {
		t.Errorf("state = %s, want validate", approved.State)
	}

	// 8. Check balance after approval
	balAfter, err := uc.GetEmployeeLeaveBalance(ctx, emp.ID, 2026)
	if err != nil {
		t.Fatalf("failed to get updated balance: %v", err)
	}
	for _, b := range balAfter.Balances {
		if b.LeaveType == hr.LeaveTypeAnnual {
			if b.UsedDays != 3.0 || b.RemainingDays != 7.0 {
				t.Errorf("after approve: used=%.1f, remaining=%.1f; want used=3, remaining=7", b.UsedDays, b.RemainingDays)
			}
		}
	}
}
