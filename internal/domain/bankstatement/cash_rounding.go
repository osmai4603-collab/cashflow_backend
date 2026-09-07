package bankstatement

import (
	"math"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// RoundingMethod is the rounding strategy applied by a cash rounding
// (account.cash.rounding in Odoo).
type RoundingMethod string

const (
	RoundingMethodUP       RoundingMethod = "UP"
	RoundingMethodDOWN     RoundingMethod = "DOWN"
	RoundingMethodHalfUp   RoundingMethod = "HALF-UP"
	RoundingMethodHalfDown RoundingMethod = "HALF-DOWN"
)

// RoundingStrategy determines how the rounding difference is attached.
type RoundingStrategy string

const (
	RoundingStrategyAddInvoiceLine RoundingStrategy = "add_invoice_line"
	RoundingStrategyBiggestTax     RoundingStrategy = "biggest_tax"
)

// CashRounding rounds invoice amounts to a given granularity
// (account.cash.rounding in Odoo).
type CashRounding struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"`
	RoundingMethod  RoundingMethod   `json:"rounding_method"`
	Rounding        float64          `json:"rounding"` // granularity, e.g. 0.05
	Strategy        RoundingStrategy `json:"strategy"`
	ProfitAccountID *int64           `json:"profit_account_id,omitempty"`
	LossAccountID   *int64           `json:"loss_account_id,omitempty"`
	Active          bool             `json:"active"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// Validate checks the required constraints of a cash rounding configuration.
func (c *CashRounding) Validate() error {
	if c.Name == "" {
		return platformerrors.Validation("cash rounding name is required", map[string]string{"name": "cannot be empty"})
	}
	if c.Rounding <= 0 {
		return platformerrors.Validation("invalid rounding granularity", map[string]string{"rounding": "must be greater than zero"})
	}
	switch c.RoundingMethod {
	case "", RoundingMethodUP, RoundingMethodDOWN, RoundingMethodHalfUp, RoundingMethodHalfDown:
	default:
		return platformerrors.Validation("invalid rounding method", map[string]string{"rounding_method": string(c.RoundingMethod)})
	}
	switch c.Strategy {
	case "", RoundingStrategyAddInvoiceLine, RoundingStrategyBiggestTax:
	default:
		return platformerrors.Validation("invalid rounding strategy", map[string]string{"strategy": string(c.Strategy)})
	}
	if c.RoundingMethod == "" {
		c.RoundingMethod = RoundingMethodHalfUp
	}
	if c.Strategy == "" {
		c.Strategy = RoundingStrategyAddInvoiceLine
	}
	return nil
}

// roundHalf rounds half-way cases away from zero.
func roundHalf(v float64) float64 {
	if v < 0 {
		return -math.Floor(-v + 0.5)
	}
	return math.Floor(v + 0.5)
}

// roundHalfDown rounds half-way cases toward zero on positive numbers (toward negative infinity on negatives).
func roundHalfDown(v float64) float64 {
	if v < 0 {
		return -math.Ceil(-v - 0.5)
	}
	return math.Ceil(v - 0.5)
}

// Round rounds the given amount to the configured granularity using the configured method.
func (c *CashRounding) Round(amount float64) float64 {
	if c.Rounding <= 0 {
		return amount
	}
	factor := 1 / c.Rounding
	scaled := amount * factor
	var rounded float64
	switch c.RoundingMethod {
	case RoundingMethodUP:
		rounded = math.Ceil(scaled)
	case RoundingMethodDOWN:
		rounded = math.Floor(scaled)
	case RoundingMethodHalfUp:
		rounded = roundHalf(scaled)
	case RoundingMethodHalfDown:
		rounded = roundHalfDown(scaled)
	default:
		rounded = roundHalf(scaled)
	}
	return roundAmount(rounded / factor)
}

// RoundDiff returns the difference to add so that amount + diff lands on the rounded value.
func (c *CashRounding) RoundDiff(amount float64) float64 {
	return roundAmount(c.Round(amount) - amount)
}
