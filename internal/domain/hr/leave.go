package hr

import (
	"fmt"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// LeaveType constants
const (
	LeaveTypeAnnual    = "annual"    // Regular annual leave / PTO
	LeaveTypeSick      = "sick"      // Medical / sick leave
	LeaveTypeUnpaid    = "unpaid"    // Leave without pay
	LeaveTypeEmergency = "emergency" // Compassionate / emergency leave
	LeaveTypeMaternity = "maternity" // Maternity leave
	LeaveTypePaternity = "paternity" // Paternity leave
)

// LeaveState represents the workflow state of a leave request
type LeaveState string

const (
	LeaveStateDraft     LeaveState = "draft"     // Initial draft
	LeaveStateConfirm   LeaveState = "confirm"   // Submitted, awaiting approval
	LeaveStateValidate  LeaveState = "validate"  // Approved and booked
	LeaveStateRefuse    LeaveState = "refuse"    // Rejected with reason
	LeaveStateCancelled LeaveState = "cancelled" // Cancelled
)

// AllocationState represents the status of a leave allocation
type AllocationState string

const (
	AllocationStateDraft     AllocationState = "draft"
	AllocationStateApproved  AllocationState = "approved"
	AllocationStateCancelled AllocationState = "cancelled"
)

// LeaveAllocation represents allocated vacation days per employee and leave type (hr.leave.allocation in Odoo).
type LeaveAllocation struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	EmployeeID    int64           `json:"employee_id"`
	LeaveType     string          `json:"leave_type"`
	AllocatedDays float64         `json:"allocated_days"`
	Year          int             `json:"year"`
	State         AllocationState `json:"state"`
	Notes         string          `json:"notes,omitempty"`
	CompanyID     *int64          `json:"company_id,omitempty"`
	Audit         audit.Fields    `json:"audit"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	CreatedBy     *int64          `json:"created_by,omitempty"`
	UpdatedBy     *int64          `json:"updated_by,omitempty"`
}

// Validate checks business invariants for LeaveAllocation.
func (a *LeaveAllocation) Validate() error {
	if a.EmployeeID <= 0 {
		return platformerrors.Validation("employee is required", map[string]string{
			"employee_id": "must reference a valid employee",
		})
	}
	a.LeaveType = strings.TrimSpace(a.LeaveType)
	if a.LeaveType == "" {
		return platformerrors.Validation("leave type is required", map[string]string{
			"leave_type": "cannot be empty",
		})
	}
	if a.AllocatedDays < 0 {
		return platformerrors.Validation("invalid allocated days", map[string]string{
			"allocated_days": "must be greater than or equal to 0",
		})
	}
	if a.Year <= 1900 || a.Year > 2200 {
		a.Year = time.Now().Year()
	}
	if a.State == "" {
		a.State = AllocationStateApproved
	}
	if a.Name == "" {
		a.Name = fmt.Sprintf("Allocation %s %d", strings.Title(a.LeaveType), a.Year)
	}
	return nil
}

// LeaveRequest represents an employee's time-off request (hr.leave in Odoo).
type LeaveRequest struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	EmployeeID    int64        `json:"employee_id"`
	LeaveType     string       `json:"leave_type"`
	DateFrom      time.Time    `json:"date_from"`
	DateTo        time.Time    `json:"date_to"`
	Days          float64      `json:"days"`
	State         LeaveState   `json:"state"`
	Description   string       `json:"description,omitempty"`
	ApproverID    *int64       `json:"approver_id,omitempty"`
	RefusalReason string       `json:"refusal_reason,omitempty"`
	CompanyID     *int64       `json:"company_id,omitempty"`
	Audit         audit.Fields `json:"audit"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	CreatedBy     *int64       `json:"created_by,omitempty"`
	UpdatedBy     *int64       `json:"updated_by,omitempty"`
}

// CalculateDays calculates the inclusive number of calendar days between dateFrom and dateTo.
func CalculateDays(from, to time.Time) float64 {
	// Truncate to start of day in UTC
	dFrom := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	dTo := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	if dTo.Before(dFrom) {
		return 0
	}
	hours := dTo.Sub(dFrom).Hours()
	return math.Round(hours/24.0) + 1.0
}

// Validate checks invariants for LeaveRequest.
func (r *LeaveRequest) Validate() error {
	if r.EmployeeID <= 0 {
		return platformerrors.Validation("employee is required", map[string]string{
			"employee_id": "must reference a valid employee",
		})
	}
	r.LeaveType = strings.TrimSpace(r.LeaveType)
	if r.LeaveType == "" {
		return platformerrors.Validation("leave type is required", map[string]string{
			"leave_type": "cannot be empty",
		})
	}
	if r.DateTo.Before(r.DateFrom) {
		return platformerrors.Validation("invalid date range", map[string]string{
			"date_to": "end date cannot be before start date",
		})
	}
	if r.Days <= 0 {
		calc := CalculateDays(r.DateFrom, r.DateTo)
		if calc <= 0 {
			calc = 1.0
		}
		r.Days = calc
	}
	if r.State == "" {
		r.State = LeaveStateDraft
	}
	if r.Name == "" {
		r.Name = fmt.Sprintf("Leave: %s (%.1f days)", r.LeaveType, r.Days)
	}
	return nil
}

// State transition methods

// Confirm transitions a draft request to confirm (submitted).
func (r *LeaveRequest) Confirm() error {
	if r.State != LeaveStateDraft {
		return platformerrors.Validation(
			fmt.Sprintf("cannot submit leave request from state '%s'; must be in 'draft'", r.State),
			nil,
		)
	}
	r.State = LeaveStateConfirm
	return nil
}

// Approve transitions a submitted request to validate (approved).
func (r *LeaveRequest) Approve(approverID int64) error {
	if r.State != LeaveStateConfirm && r.State != LeaveStateDraft {
		return platformerrors.Validation(
			fmt.Sprintf("cannot approve leave request from state '%s'; must be in 'confirm' or 'draft'", r.State),
			nil,
		)
	}
	r.State = LeaveStateValidate
	if approverID > 0 {
		r.ApproverID = &approverID
	}
	return nil
}

// Refuse transitions a request to refuse (rejected).
func (r *LeaveRequest) Refuse(approverID int64, reason string) error {
	if r.State != LeaveStateConfirm && r.State != LeaveStateDraft {
		return platformerrors.Validation(
			fmt.Sprintf("cannot refuse leave request from state '%s'; must be in 'confirm' or 'draft'", r.State),
			nil,
		)
	}
	r.State = LeaveStateRefuse
	r.RefusalReason = strings.TrimSpace(reason)
	if approverID > 0 {
		r.ApproverID = &approverID
	}
	return nil
}

// Cancel transitions a request to cancelled and frees up balance.
func (r *LeaveRequest) Cancel() error {
	if r.State == LeaveStateCancelled {
		return platformerrors.Validation("leave request is already cancelled", nil)
	}
	r.State = LeaveStateCancelled
	return nil
}

// LeaveBalance represents the leave accounting status for an employee.
type LeaveBalance struct {
	EmployeeID    int64   `json:"employee_id"`
	LeaveType     string  `json:"leave_type"`
	AllocatedDays float64 `json:"allocated_days"`
	UsedDays      float64 `json:"used_days"`
	PendingDays   float64 `json:"pending_days"`
	RemainingDays float64 `json:"remaining_days"`
}

// EmployeeLeaveSummary gathers all balances for an employee.
type EmployeeLeaveSummary struct {
	EmployeeID   int64          `json:"employee_id"`
	EmployeeName string         `json:"employee_name"`
	Year         int            `json:"year"`
	Balances     []LeaveBalance `json:"balances"`
}
