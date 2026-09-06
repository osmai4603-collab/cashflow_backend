package payment

import (
	"fmt"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// PaymentType determines the flow direction of the financial transaction.
type PaymentType string

const (
	PaymentTypeInbound  PaymentType = "inbound"  // Customer payment (receivable / collection)
	PaymentTypeOutbound PaymentType = "outbound" // Vendor payment (payable / disbursement)
)

// PartnerType identifies whether the recipient/payer is a customer or supplier.
type PartnerType string

const (
	PartnerTypeCustomer PartnerType = "customer"
	PartnerTypeSupplier PartnerType = "supplier"
)

// PaymentMethod defines the medium of money transfer.
type PaymentMethod string

const (
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCheck        PaymentMethod = "check"
)

// PaymentState represents the lifecycle status of a payment.
type PaymentState string

const (
	PaymentStateDraft      PaymentState = "draft"
	PaymentStatePosted     PaymentState = "posted"
	PaymentStateReconciled PaymentState = "reconciled"
	PaymentStateCancelled  PaymentState = "cancelled"
)

// Payment represents a financial payment transaction (account.payment in Odoo).
type Payment struct {
	ID               int64         `json:"id"`
	Name             string        `json:"name"` // Sequence number e.g. "PAY/2026/00001" or "/"
	PaymentType      PaymentType   `json:"payment_type"`
	PartnerType      PartnerType   `json:"partner_type"`
	PartnerID        int64         `json:"partner_id"`
	Amount           float64       `json:"amount"`
	Currency         string        `json:"currency"`
	PaymentMethod    PaymentMethod `json:"payment_method"`
	JournalID        int64         `json:"journal_id"`
	Date             time.Time     `json:"date"`
	State            PaymentState  `json:"state"`
	Ref              string        `json:"ref,omitempty"`
	MoveID           *int64        `json:"move_id,omitempty"`
	InvoiceIDs       []int64       `json:"invoice_ids,omitempty"`
	ReconciledAmount float64       `json:"reconciled_amount"`
	ResidualAmount   float64       `json:"residual_amount"` // Unallocated payment amount
	CompanyID        *int64        `json:"company_id,omitempty"`
	Active           bool          `json:"active"`
	Audit            audit.Fields  `json:"audit"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	CreatedBy        *int64        `json:"created_by,omitempty"`
	UpdatedBy        *int64        `json:"updated_by,omitempty"`
}

// Validate checks business constraints on the Payment entity.
func (p *Payment) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = "/"
	}

	if p.PartnerID <= 0 {
		return platformerrors.Validation("partner is required", map[string]string{
			"partner_id": "must reference a valid partner",
		})
	}

	if p.JournalID <= 0 {
		return platformerrors.Validation("journal is required", map[string]string{
			"journal_id": "must reference a valid cash or bank journal",
		})
	}

	if p.Amount <= 0 {
		return platformerrors.Validation("invalid payment amount", map[string]string{
			"amount": "payment amount must be greater than zero",
		})
	}

	p.Amount = roundTo4(p.Amount)

	switch p.PaymentType {
	case PaymentTypeInbound, PaymentTypeOutbound:
	default:
		return platformerrors.Validation("invalid payment type", map[string]string{
			"payment_type": fmt.Sprintf("unsupported payment type '%s', expected 'inbound' or 'outbound'", p.PaymentType),
		})
	}

	if p.PartnerType == "" {
		if p.PaymentType == PaymentTypeInbound {
			p.PartnerType = PartnerTypeCustomer
		} else {
			p.PartnerType = PartnerTypeSupplier
		}
	}

	switch p.PartnerType {
	case PartnerTypeCustomer, PartnerTypeSupplier:
	default:
		return platformerrors.Validation("invalid partner type", map[string]string{
			"partner_type": fmt.Sprintf("unsupported partner type '%s', expected 'customer' or 'supplier'", p.PartnerType),
		})
	}

	if p.PaymentMethod == "" {
		p.PaymentMethod = PaymentMethodCash
	}
	switch p.PaymentMethod {
	case PaymentMethodCash, PaymentMethodBankTransfer, PaymentMethodCheck:
	default:
		return platformerrors.Validation("invalid payment method", map[string]string{
			"payment_method": fmt.Sprintf("unsupported payment method '%s'", p.PaymentMethod),
		})
	}

	if p.Date.IsZero() {
		p.Date = time.Now().UTC()
	}

	if p.Currency == "" {
		p.Currency = "USD"
	}

	if p.State == "" {
		p.State = PaymentStateDraft
	}

	// Initialize residual amount if fresh draft
	if p.ReconciledAmount == 0 && p.ResidualAmount == 0 && p.State == PaymentStateDraft {
		p.ResidualAmount = p.Amount
	}

	return nil
}

// Post transitions the payment to posted state and sets sequence number.
func (p *Payment) Post(seq string) error {
	if p.State == PaymentStatePosted || p.State == PaymentStateReconciled {
		return platformerrors.Conflict("payment is already posted")
	}
	if p.State == PaymentStateCancelled {
		return platformerrors.Conflict("cannot post a cancelled payment")
	}

	p.State = PaymentStatePosted
	if seq != "" {
		p.Name = seq
	}
	if p.ResidualAmount == 0 && p.ReconciledAmount == 0 {
		p.ResidualAmount = p.Amount
	}
	return nil
}

// Cancel transitions the payment to cancelled state.
func (p *Payment) Cancel() error {
	if p.State == PaymentStateCancelled {
		return nil
	}
	p.State = PaymentStateCancelled
	p.ResidualAmount = 0
	return nil
}

// AllocateReconciliation registers an applied portion of the payment to an invoice.
func (p *Payment) AllocateReconciliation(appliedAmount float64) error {
	if p.State != PaymentStatePosted && p.State != PaymentStateReconciled {
		return platformerrors.Conflict("only posted payments can be reconciled")
	}
	appliedAmount = roundTo4(appliedAmount)
	if appliedAmount <= 0 {
		return platformerrors.Validation("reconciliation amount must be greater than zero", nil)
	}
	if appliedAmount > p.ResidualAmount+0.0001 {
		return platformerrors.Validation("reconciliation amount exceeds residual payment amount", map[string]string{
			"amount": fmt.Sprintf("attempted %.4f but residual is %.4f", appliedAmount, p.ResidualAmount),
		})
	}

	p.ReconciledAmount = roundTo4(p.ReconciledAmount + appliedAmount)
	p.ResidualAmount = roundTo4(p.ResidualAmount - appliedAmount)
	if p.ResidualAmount < 0.0001 {
		p.ResidualAmount = 0
		p.State = PaymentStateReconciled
	} else {
		p.State = PaymentStatePosted
	}
	return nil
}

// Unreconcile restores all allocated reconciliation amounts back to residual.
func (p *Payment) Unreconcile() error {
	if p.State == PaymentStateCancelled {
		return platformerrors.Conflict("cannot unreconcile a cancelled payment")
	}
	p.ResidualAmount = p.Amount
	p.ReconciledAmount = 0
	p.State = PaymentStatePosted
	return nil
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
