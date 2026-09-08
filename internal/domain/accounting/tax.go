package accounting

import (
	"fmt"
	"math"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// TaxType defines how the tax amount is calculated.
type TaxType string

const (
	TaxTypePercent TaxType = "percent"
	TaxTypeFixed   TaxType = "fixed"
)

// TaxScope defines where the tax is applicable (Sales or Purchases).
type TaxScope string

const (
	TaxScopeSale     TaxScope = "sale"
	TaxScopePurchase TaxScope = "purchase"
	TaxScopeNone     TaxScope = "none"
)

// Tax represents a tax rate configuration (account.tax in Odoo).
type Tax struct {
	ID              int64                   `json:"id"`
	Name            i18n.TranslationString `json:"name"`
	Type            TaxType                 `json:"type"`
	TypeTaxUse      TaxScope                `json:"type_tax_use"`
	Amount          float64                 `json:"amount"` // e.g. 15.0 for 15%
	AmountType      TaxType                 `json:"amount_type,omitempty"`
	IncludeBase     bool                    `json:"include_base,omitempty"`
	CountryCode     string                  `json:"country_code,omitempty"`
	AccountID       int64                   `json:"account_id"`
	RefundAccountID *int64                  `json:"refund_account_id,omitempty"`
	PriceInclude    bool                    `json:"price_include"` // whether price includes tax
	Active          bool                    `json:"active"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// Validate checks Tax constraints.
func (t *Tax) Validate() error {
	if len(t.Name) == 0 {
		return platformerrors.Validation("tax name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if t.Type != TaxTypePercent && t.Type != TaxTypeFixed {
		return platformerrors.Validation("invalid tax type", map[string]string{
			"type": fmt.Sprintf("must be '%s' or '%s'", TaxTypePercent, TaxTypeFixed),
		})
	}

	if t.TypeTaxUse != TaxScopeSale && t.TypeTaxUse != TaxScopePurchase && t.TypeTaxUse != TaxScopeNone {
		return platformerrors.Validation("invalid tax scope", map[string]string{
			"type_tax_use": fmt.Sprintf("must be '%s', '%s', or '%s'", TaxScopeSale, TaxScopePurchase, TaxScopeNone),
		})
	}

	if t.Amount < 0 {
		return platformerrors.Validation("tax amount cannot be negative", map[string]string{
			"amount": "must be greater than or equal to 0",
		})
	}

	if t.AccountID <= 0 {
		return platformerrors.Validation("tax account is required", map[string]string{
			"account_id": "must reference a valid account",
		})
	}

	return nil
}

// TaxCalculationResult holds the outcome of a tax computation.
type TaxCalculationResult struct {
	UntaxedAmount float64 `json:"untaxed_amount"`
	TaxAmount     float64 `json:"tax_amount"`
	TotalAmount   float64 `json:"total_amount"`
}

// Compute calculates tax and untaxed bases given a unit or line amount.
func (t *Tax) Compute(baseAmount float64) TaxCalculationResult {
	baseAmount = roundTo4(baseAmount)

	if t.PriceInclude {
		// Price contains the tax: e.g. 115 with 15% tax => Untaxed is 100, Tax is 15
		switch t.Type {
		case TaxTypePercent:
			factor := 1.0 + (t.Amount / 100.0)
			untaxed := roundTo4(baseAmount / factor)
			tax := roundTo4(baseAmount - untaxed)
			return TaxCalculationResult{
				UntaxedAmount: untaxed,
				TaxAmount:     tax,
				TotalAmount:   baseAmount,
			}
		case TaxTypeFixed:
			tax := roundTo4(t.Amount)
			untaxed := roundTo4(baseAmount - tax)
			if untaxed < 0 {
				untaxed = 0
			}
			return TaxCalculationResult{
				UntaxedAmount: untaxed,
				TaxAmount:     tax,
				TotalAmount:   baseAmount,
			}
		}
	}

	// Price does NOT contain tax (Tax added on top): e.g. 100 with 15% tax => Tax is 15, Total is 115
	switch t.Type {
	case TaxTypePercent:
		tax := roundTo4(baseAmount * (t.Amount / 100.0))
		total := roundTo4(baseAmount + tax)
		return TaxCalculationResult{
			UntaxedAmount: baseAmount,
			TaxAmount:     tax,
			TotalAmount:   total,
		}
	case TaxTypeFixed:
		tax := roundTo4(t.Amount)
		total := roundTo4(baseAmount + tax)
		return TaxCalculationResult{
			UntaxedAmount: baseAmount,
			TaxAmount:     tax,
			TotalAmount:   total,
		}
	}

	return TaxCalculationResult{
		UntaxedAmount: baseAmount,
		TaxAmount:     0,
		TotalAmount:   baseAmount,
	}
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
