package hr_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/domain/hr"
)

func TestDepartment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dept    hr.Department
		wantErr bool
	}{
		{
			name: "valid department",
			dept: hr.Department{
				Name: "Engineering",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			dept: hr.Department{
				Name: "   ",
			},
			wantErr: true,
		},
		{
			name: "self parent",
			dept: hr.Department{
				ID:       5,
				Name:     "Finance",
				ParentID: func() *int64 { id := int64(5); return &id }(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dept.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJob_Validate(t *testing.T) {
	tests := []struct {
		name    string
		job     hr.Job
		wantErr bool
	}{
		{
			name: "valid job",
			job: hr.Job{
				Name:              "Backend Developer",
				ExpectedEmployees: 3,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			job: hr.Job{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "negative expected employees",
			job: hr.Job{
				Name:              "Tester",
				ExpectedEmployees: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmployee_Validate(t *testing.T) {
	tests := []struct {
		name    string
		emp     hr.Employee
		wantErr bool
	}{
		{
			name: "valid employee",
			emp: hr.Employee{
				Name:      "Ahmed Ali",
				WorkEmail: "ahmed@example.com",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			emp: hr.Employee{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			emp: hr.Employee{
				Name:      "Ahmed Ali",
				WorkEmail: "invalid-email",
			},
			wantErr: true,
		},
		{
			name: "self manager",
			emp: hr.Employee{
				ID:        10,
				Name:      "CEO",
				ManagerID: func() *int64 { id := int64(10); return &id }(),
			},
			wantErr: true,
		},
		{
			name: "self expense manager",
			emp: hr.Employee{
				ID:               10,
				Name:             "CEO",
				ExpenseManagerID: func() *int64 { id := int64(10); return &id }(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.emp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaveRequest_CalculateDays(t *testing.T) {
	d1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	days := hr.CalculateDays(d1, d2)
	if days != 5.0 {
		t.Errorf("CalculateDays() = %v, want 5.0", days)
	}

	sameDay := hr.CalculateDays(d1, d1)
	if sameDay != 1.0 {
		t.Errorf("CalculateDays() same day = %v, want 1.0", sameDay)
	}

	inverted := hr.CalculateDays(d2, d1)
	if inverted != 0.0 {
		t.Errorf("CalculateDays() inverted = %v, want 0.0", inverted)
	}
}

func TestLeaveRequest_StateTransitions(t *testing.T) {
	from := time.Now()
	to := from.AddDate(0, 0, 2)
	req := hr.LeaveRequest{
		EmployeeID: 1,
		LeaveType:  hr.LeaveTypeAnnual,
		DateFrom:   from,
		DateTo:     to,
		Days:       3,
		State:      hr.LeaveStateDraft,
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Confirm
	if err := req.Confirm(); err != nil {
		t.Fatalf("failed to confirm: %v", err)
	}
	if req.State != hr.LeaveStateConfirm {
		t.Errorf("state after confirm = %s, want %s", req.State, hr.LeaveStateConfirm)
	}

	// Cannot confirm again
	if err := req.Confirm(); err == nil {
		t.Errorf("expected error confirming non-draft request")
	}

	// Approve
	if err := req.Approve(99); err != nil {
		t.Fatalf("failed to approve: %v", err)
	}
	if req.State != hr.LeaveStateValidate || req.ApproverID == nil || *req.ApproverID != 99 {
		t.Errorf("invalid approved state or approver id: state=%s, approver=%v", req.State, req.ApproverID)
	}

	// Cancel
	if err := req.Cancel(); err != nil {
		t.Fatalf("failed to cancel: %v", err)
	}
	if req.State != hr.LeaveStateCancelled {
		t.Errorf("state after cancel = %s, want %s", req.State, hr.LeaveStateCancelled)
	}
}
