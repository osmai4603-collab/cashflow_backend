package pos

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type PosPayment struct {
	ID              int64     `json:"id"`
	OrderID         int64     `json:"order_id"`
	SessionID       int64     `json:"session_id"`
	PaymentMethodID int64     `json:"payment_method_id"`
	Amount          float64   `json:"amount"`
	PaymentDate     time.Time `json:"payment_date"`
	TransactionID   string    `json:"transaction_id,omitempty"`
}

func (payment *PosPayment) Validate() error {
	payment.TransactionID = strings.TrimSpace(payment.TransactionID)
	if payment.PaymentMethodID <= 0 {
		return platformerrors.Validation("payment method is required", nil)
	}
	if payment.Amount <= 0 {
		return platformerrors.Validation("payment amount must be positive", nil)
	}
	if payment.PaymentDate.IsZero() {
		payment.PaymentDate = time.Now().UTC()
	}
	return nil
}
