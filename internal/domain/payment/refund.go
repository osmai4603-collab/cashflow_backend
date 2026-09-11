package payment

import (
	platformerrors "cashflow_backend/internal/platform/errors"
	"time"
)

type RefundReason string

const (
	RefundReasonCustomerRequest RefundReason = "customer_request"
	RefundReasonDuplicate       RefundReason = "duplicate"
	RefundReasonFraudulent      RefundReason = "fraudulent"
	RefundReasonProductIssue    RefundReason = "product_issue"
)

type PaymentRefund struct {
	ID                int64        `json:"id"`
	OriginalTxID      int64        `json:"original_transaction_id"`
	RefundTxID        *int64       `json:"refund_transaction_id,omitempty"`
	Amount            float64      `json:"amount"`
	Currency          string       `json:"currency"`
	Reason            RefundReason `json:"reason"`
	ProviderReference string       `json:"provider_reference,omitempty"`
	State             string       `json:"state"`
	CompanyID         int64        `json:"company_id"`
	CreatedAt         time.Time    `json:"created_at"`
}

func (r *PaymentRefund) Validate(original *PaymentTransaction) error {
	if original == nil || r.OriginalTxID != original.ID || r.Amount <= 0 || r.Amount > original.Amount || r.Currency == "" {
		return platformerrors.Validation("invalid refund amount or transaction", nil)
	}
	if r.State == "" {
		r.State = "pending"
	}
	return nil
}
