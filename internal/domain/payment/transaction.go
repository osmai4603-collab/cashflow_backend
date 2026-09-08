package payment

import (
	"time"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type TransactionState string

const (
	TransactionStateDraft      TransactionState = "draft"
	TransactionStatePending    TransactionState = "pending"
	TransactionStateAuthorized TransactionState = "authorized"
	TransactionStateConfirmed  TransactionState = "confirmed"
	TransactionStateError      TransactionState = "error"
	TransactionStateCancelled  TransactionState = "cancel"
)

type PaymentProvider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"` // e.g., "stripe", "paypal", "tap"
	Active    bool   `json:"active"`
	CompanyID int64  `json:"company_id"`
}

type PaymentTransaction struct {
	ID                int64            `json:"id"`
	Reference         string           `json:"reference"`
	Amount            float64          `json:"amount"`
	Currency          string           `json:"currency"`
	ProviderID        int64            `json:"provider_id"`
	PartnerID         int64            `json:"partner_id"`
	State             TransactionState `json:"state"`
	ProviderReference string           `json:"provider_reference,omitempty"`
	PaymentID         *int64           `json:"payment_id,omitempty"` // Internal account.payment link
	Metadata          map[string]any   `json:"metadata,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	CompanyID         int64            `json:"company_id"`
}

func (t *PaymentTransaction) Validate() error {
	if t.Reference == "" {
		return platformerrors.Validation("reference is required", nil)
	}
	if t.Amount <= 0 {
		return platformerrors.Validation("amount must be positive", nil)
	}
	if t.ProviderID <= 0 {
		return platformerrors.Validation("provider_id is required", nil)
	}
	if t.PartnerID <= 0 {
		return platformerrors.Validation("partner_id is required", nil)
	}
	if t.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	return nil
}
