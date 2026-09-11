package payment

import (
	"fmt"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type TransactionState string

const (
	TransactionStateDraft      TransactionState = "draft"
	TransactionStatePending    TransactionState = "pending"
	TransactionStateAuthorized TransactionState = "authorized"
	TransactionStateConfirmed  TransactionState = "confirmed"
	TransactionStateDone       TransactionState = "done"
	TransactionStateError      TransactionState = "error"
	TransactionStateCancelled  TransactionState = "cancel"
)

type PaymentProvider struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"` // e.g., "stripe", "paypal", "tap"
	State     string    `json:"state"` // "enabled", "disabled", "test"
	Active    bool      `json:"active"`
	CompanyID int64     `json:"company_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	SaleOrderID       *int64           `json:"sale_order_id,omitempty"`
	InvoiceID         *int64           `json:"invoice_id,omitempty"`
	PaymentID         *int64           `json:"payment_id,omitempty"` // Internal account.payment link
	IdempotencyKey    string           `json:"idempotency_key,omitempty"`
	ReturnURL         string           `json:"return_url,omitempty"`
	WebhookReceived   bool             `json:"webhook_received"`
	LastError         string           `json:"last_error,omitempty"`
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
	if t.State == "" {
		t.State = TransactionStateDraft
	}
	return nil
}

// Transition performs a validated state machine transition for the payment transaction.
func (t *PaymentTransaction) Transition(to TransactionState) error {
	if t.State == to {
		return nil
	}

	valid := false
	switch t.State {
	case TransactionStateDraft:
		valid = to == TransactionStatePending || to == TransactionStateAuthorized || to == TransactionStateConfirmed || to == TransactionStateDone || to == TransactionStateCancelled || to == TransactionStateError
	case TransactionStatePending:
		valid = to == TransactionStateAuthorized || to == TransactionStateConfirmed || to == TransactionStateDone || to == TransactionStateCancelled || to == TransactionStateError
	case TransactionStateAuthorized:
		valid = to == TransactionStateConfirmed || to == TransactionStateDone || to == TransactionStateCancelled || to == TransactionStateError
	case TransactionStateConfirmed, TransactionStateDone:
		// Terminal successful states
		valid = false
	case TransactionStateCancelled:
		// Terminal cancelled state
		valid = false
	case TransactionStateError:
		// Error can retry to pending or be cancelled
		valid = to == TransactionStatePending || to == TransactionStateDraft || to == TransactionStateCancelled
	default:
		valid = false
	}

	if !valid {
		return platformerrors.Conflict(
			fmt.Sprintf("cannot transition payment transaction from %s to %s", t.State, to),
		)
	}

	t.State = to
	t.UpdatedAt = time.Now().UTC()
	return nil
}

