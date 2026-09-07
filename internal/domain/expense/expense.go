package expense

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// ExpenseState defines the status of an expense.
type ExpenseState string

const (
	StateDraft     ExpenseState = "draft"
	StateSubmitted ExpenseState = "submitted"
	StateApproved  ExpenseState = "approved"
	StatePosted    ExpenseState = "posted"
	StatePaid      ExpenseState = "paid"
	StateRefused   ExpenseState = "refused"
)

// PaymentMode defines who paid for the expense.
type PaymentMode string

const (
	PaymentOwnAccount     PaymentMode = "own_account"     // Employee (to reimburse)
	PaymentCompanyAccount PaymentMode = "company_account" // Company (direct payment)
)

// Expense represents an employee expense (hr.expense in Odoo).
type Expense struct {
	ID                int64        `json:"id"`
	Name              string       `json:"name"` // Description
	Date              time.Time    `json:"date"`
	EmployeeID        int64        `json:"employee_id"`
	ManagerID         *int64       `json:"manager_id,omitempty"`
	DepartmentID      *int64       `json:"department_id,omitempty"`
	ProductID         *int64       `json:"product_id,omitempty"` // Expense Category
	UnitAmount        float64      `json:"unit_amount"`
	Quantity          float64      `json:"quantity"`
	TotalAmount       float64      `json:"total_amount"`
	UntaxedAmount     float64      `json:"untaxed_amount"`
	TaxAmount         float64      `json:"tax_amount"`
	CurrencyID        int64        `json:"currency_id"`
	PaymentMode       PaymentMode  `json:"payment_mode"`
	AccountID         *int64       `json:"account_id,omitempty"`
	AnalyticAccountID *int64       `json:"analytic_account_id,omitempty"`
	AccountMoveID     *int64       `json:"account_move_id,omitempty"` // Linked journal entry
	VendorID          *int64       `json:"vendor_id,omitempty"`
	Description       string       `json:"description,omitempty"` // Internal notes
	State             ExpenseState `json:"state"`
	ApprovalDate      *time.Time   `json:"approval_date,omitempty"`
	RefuseReason      string       `json:"refuse_reason,omitempty"`
	AttachmentIDs     []int64      `json:"attachment_ids,omitempty"`
	SplitOriginID     *int64       `json:"split_origin_id,omitempty"` // If this expense was split from another
	CompanyID         int64        `json:"company_id"`
	Audit             audit.Fields `json:"audit"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

// Validate checks business invariants for the Expense entity.
func (e *Expense) Validate() error {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		return platformerrors.Validation("expense name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if e.EmployeeID <= 0 {
		return platformerrors.Validation("employee is required", map[string]string{
			"employee_id": "must be a valid employee ID",
		})
	}

	if e.Quantity <= 0 {
		e.Quantity = 1.0
	}

	if e.TotalAmount < 0 {
		return platformerrors.Validation("total amount cannot be negative", map[string]string{
			"total_amount": "must be greater than or equal to 0",
		})
	}

	if e.PaymentMode == "" {
		e.PaymentMode = PaymentOwnAccount
	}

	if e.State == "" {
		e.State = StateDraft
	}

	return nil
}

// ExpenseSplitRequest holds the data needed to split an expense.
type ExpenseSplitRequest struct {
	ExpenseID int64                `json:"expense_id"`
	Splits    []ExpenseSplitLine   `json:"splits"`
}

type ExpenseSplitLine struct {
	Name              string  `json:"name"`
	ProductID         *int64  `json:"product_id,omitempty"`
	TotalAmount       float64 `json:"total_amount"`
	AccountID         *int64  `json:"account_id,omitempty"`
	AnalyticAccountID *int64  `json:"analytic_account_id,omitempty"`
}
