package payment

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// PaymentReconciliation records the financial match between a payment and an invoice.
type PaymentReconciliation struct {
	ID           int64     `json:"id"`
	PaymentID    int64     `json:"payment_id"`
	InvoiceID    int64     `json:"invoice_id"`
	Amount       float64   `json:"amount"`
	ReconciledAt time.Time `json:"reconciled_at"`
}

// Validate checks reconciliation record constraints.
func (r *PaymentReconciliation) Validate() error {
	if r.PaymentID <= 0 {
		return platformerrors.Validation("payment_id is required", map[string]string{
			"payment_id": "must reference a valid payment",
		})
	}
	if r.InvoiceID <= 0 {
		return platformerrors.Validation("invoice_id is required", map[string]string{
			"invoice_id": "must reference a valid invoice",
		})
	}
	if r.Amount <= 0 {
		return platformerrors.Validation("amount is required", map[string]string{
			"amount": "reconciliation amount must be greater than zero",
		})
	}
	r.Amount = roundTo4(r.Amount)
	if r.ReconciledAt.IsZero() {
		r.ReconciledAt = time.Now().UTC()
	}
	return nil
}
