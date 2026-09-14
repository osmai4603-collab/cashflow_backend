package cleanarch

import (
	"errors"
	"time"
)

// =============================================================================
// LAYER 1: DOMAIN LAYER (Enterprise Business Rules & Entities)
// =============================================================================
// Rules:
// - Standard library only.
// - Zero imports of frameworks, ORMs, HTTP, or database packages.
// - Methods enforce domain invariants.
// =============================================================================

var (
	ErrInvalidAmount       = errors.New("domain: amount must be strictly positive")
	ErrInvoiceAlreadyPaid  = errors.New("domain: invoice is already marked as paid")
	ErrEmptyCustomerID     = errors.New("domain: customer id cannot be empty")
)

type InvoiceStatus string

const (
	InvoiceStatusDraft InvoiceStatus = "DRAFT"
	InvoiceStatusPaid  InvoiceStatus = "PAID"
	InvoiceStatusVoid  InvoiceStatus = "VOID"
)

// Invoice represents the pure business entity.
type Invoice struct {
	id         string
	customerID string
	amount     float64
	status     InvoiceStatus
	createdAt  time.Time
	paidAt     *time.Time
}

// NewInvoice constructs a valid domain entity, validating invariants.
func NewInvoice(id, customerID string, amount float64) (*Invoice, error) {
	if customerID == "" {
		return nil, ErrEmptyCustomerID
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return &Invoice{
		id:         id,
		customerID: customerID,
		amount:     amount,
		status:     InvoiceStatusDraft,
		createdAt:  time.Now(),
	}, nil
}

// Getters expose domain state safely.
func (i *Invoice) ID() string             { return i.id }
func (i *Invoice) CustomerID() string     { return i.customerID }
func (i *Invoice) Amount() float64         { return i.amount }
func (i *Invoice) Status() InvoiceStatus  { return i.status }
func (i *Invoice) CreatedAt() time.Time   { return i.createdAt }
func (i *Invoice) PaidAt() *time.Time     { return i.paidAt }

// MarkAsPaid enforces business rules regarding payments.
func (i *Invoice) MarkAsPaid(at time.Time) error {
	if i.status == InvoiceStatusPaid {
		return ErrInvoiceAlreadyPaid
	}
	i.status = InvoiceStatusPaid
	i.paidAt = &at
	return nil
}
