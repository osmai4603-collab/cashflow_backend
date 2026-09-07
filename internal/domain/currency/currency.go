package currency

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Currency represents a monetary currency (res.currency in Odoo).
type Currency struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Symbol        string    `json:"symbol"`
	DecimalPlaces int16     `json:"decimal_places"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Validate ensures the currency entity satisfies all domain invariants.
func (c *Currency) Validate() error {
	trimmedName := strings.TrimSpace(c.Name)
	if trimmedName == "" {
		return platformerrors.Validation("currency code is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if len(trimmedName) != 3 {
		return platformerrors.Validation("currency code must be exactly 3 characters", map[string]string{
			"name": "must be a 3-letter ISO 4217 code",
		})
	}
	c.Name = strings.ToUpper(trimmedName)

	trimmedFullName := strings.TrimSpace(c.FullName)
	if trimmedFullName == "" {
		return platformerrors.Validation("currency full name is required", map[string]string{
			"full_name": "cannot be empty",
		})
	}
	c.FullName = trimmedFullName

	if strings.TrimSpace(c.Symbol) == "" {
		return platformerrors.Validation("currency symbol is required", map[string]string{
			"symbol": "cannot be empty",
		})
	}

	if c.DecimalPlaces < 0 || c.DecimalPlaces > 10 {
		return platformerrors.Validation("invalid decimal places", map[string]string{
			"decimal_places": "must be between 0 and 10",
		})
	}

	return nil
}

// Round rounds the given amount to the currency's decimal places.
func (c *Currency) Round(amount float64) float64 {
	p := float64(1)
	for i := int16(0); i < c.DecimalPlaces; i++ {
		p *= 10
	}
	return float64(int(amount*p+0.5)) / p
}

// CurrencyRate represents a historical exchange rate (res.currency.rate in Odoo).
type CurrencyRate struct {
	ID         int64        `json:"id"`
	CurrencyID int64        `json:"currency_id"`
	Rate       float64      `json:"rate"`
	Date       string       `json:"date"`
	CompanyID  *int64       `json:"company_id,omitempty"`
	Audit      audit.Fields `json:"audit"`
}

// Validate ensures the currency rate entity satisfies all domain invariants.
func (cr *CurrencyRate) Validate() error {
	if cr.CurrencyID <= 0 {
		return platformerrors.Validation("currency is required", map[string]string{
			"currency_id": "must be a valid currency",
		})
	}

	if cr.Rate <= 0 {
		return platformerrors.Validation("exchange rate must be positive", map[string]string{
			"rate": "must be greater than zero",
		})
	}

	if strings.TrimSpace(cr.Date) == "" {
		return platformerrors.Validation("rate date is required", map[string]string{
			"date": "cannot be empty",
		})
	}

	return nil
}
