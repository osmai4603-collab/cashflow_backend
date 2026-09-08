package accounting

import (
	"fmt"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// PaymentTermValueType specifies how payment installment is quantified.
type PaymentTermValueType string

const (
	PaymentTermValueBalance PaymentTermValueType = "balance"
	PaymentTermValuePercent PaymentTermValueType = "percent"
	PaymentTermValueFixed   PaymentTermValueType = "fixed"
)

// PaymentTermLine defines a single installment rule.
type PaymentTermLine struct {
	ID            int64                `json:"id"`
	PaymentTermID int64                `json:"payment_term_id"`
	ValueType     PaymentTermValueType `json:"value_type"`
	ValueAmount   float64              `json:"value_amount"`
	Days          int                  `json:"days"`
	DayOfMonth    int                  `json:"day_of_month"`
}

// PaymentTerm represents customer or vendor credit terms (account.payment.term in Odoo).
type PaymentTerm struct {
	ID        int64             `json:"id"`
	Name      i18n.TranslationString            `json:"name"`
	Note      i18n.TranslationString            `json:"note,omitempty"`
	Active    bool              `json:"active"`
	Lines     []PaymentTermLine `json:"lines,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Validate checks PaymentTerm constraints.
func (pt *PaymentTerm) Validate() error {
	if len(pt.Name) == 0 {
		return platformerrors.Validation("payment term name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	for i, l := range pt.Lines {
		switch l.ValueType {
		case PaymentTermValueBalance, PaymentTermValuePercent, PaymentTermValueFixed:
		default:
			return platformerrors.Validation("invalid payment term line value type", map[string]string{
				fmt.Sprintf("lines[%d].value_type", i): "must be 'balance', 'percent', or 'fixed'",
			})
		}
		if l.Days < 0 {
			return platformerrors.Validation("payment term days cannot be negative", map[string]string{
				fmt.Sprintf("lines[%d].days", i): "must be >= 0",
			})
		}
	}

	return nil
}

// ComputeDueDate calculates the final maturity date given an invoice issuance date.
func (pt *PaymentTerm) ComputeDueDate(invoiceDate time.Time) time.Time {
	if len(pt.Lines) == 0 {
		return invoiceDate
	}

	maxDays := 0
	for _, line := range pt.Lines {
		if line.Days > maxDays {
			maxDays = line.Days
		}
	}

	return invoiceDate.AddDate(0, 0, maxDays)
}
