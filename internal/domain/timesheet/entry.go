package timesheet

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type TimesheetState string

const (
	TimesheetDraft     TimesheetState = "draft"
	TimesheetSubmitted TimesheetState = "submitted"
	TimesheetApproved  TimesheetState = "approved"
	TimesheetRejected  TimesheetState = "rejected"
)

type Entry struct {
	ID                int64          `json:"id"`
	ProjectID         int64          `json:"project_id"`
	TaskID            *int64         `json:"task_id,omitempty"`
	EmployeeID        int64          `json:"employee_id"`
	UserID            int64          `json:"user_id"`
	Date              time.Time      `json:"date"`
	UnitAmount        float64        `json:"unit_amount"`
	Name              string         `json:"name"`
	HourlyCost        float64        `json:"hourly_cost"`
	AmountTotalCost   float64        `json:"amount_total_cost"`
	AnalyticAccountID *int64         `json:"analytic_account_id,omitempty"`
	State             TimesheetState `json:"state"`
	Billable          bool           `json:"billable"`
	InvoicedTimesheet bool           `json:"invoiced_timesheet"`
	CompanyID         int64          `json:"company_id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func (entry *Entry) Validate() error {
	if entry.ProjectID <= 0 || entry.EmployeeID <= 0 || entry.UserID <= 0 || entry.CompanyID <= 0 || entry.Date.IsZero() || entry.UnitAmount <= 0 || entry.UnitAmount > 24 || entry.HourlyCost < 0 || entry.Name == "" {
		return platformerrors.Validation("timesheet entry has invalid required fields", nil)
	}
	if entry.State == "" {
		entry.State = TimesheetDraft
	}
	if entry.State != TimesheetDraft && entry.State != TimesheetSubmitted && entry.State != TimesheetApproved && entry.State != TimesheetRejected {
		return platformerrors.Validation("invalid timesheet state", nil)
	}
	entry.AmountTotalCost = roundMoney(entry.UnitAmount * entry.HourlyCost)
	return nil
}

func (entry *Entry) Submit() error {
	if entry.State != TimesheetDraft && entry.State != TimesheetRejected {
		return platformerrors.Conflict("only draft or rejected timesheets can be submitted")
	}
	entry.State = TimesheetSubmitted
	return nil
}

func (entry *Entry) Approve() error {
	if entry.State != TimesheetSubmitted {
		return platformerrors.Conflict("only submitted timesheets can be approved")
	}
	entry.State = TimesheetApproved
	return nil
}

func (entry *Entry) Reject() error {
	if entry.State != TimesheetSubmitted {
		return platformerrors.Conflict("only submitted timesheets can be rejected")
	}
	entry.State = TimesheetRejected
	return nil
}

func roundMoney(value float64) float64 {
	return float64(int64(value*10000+0.5)) / 10000
}
