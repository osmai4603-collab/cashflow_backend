package currency

import (
	"context"
	"fmt"

	domaincurrency "cashflow_backend/internal/domain/currency"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// RateProvider supplies historical exchange rates. The domain
// currency.RateRepository satisfies this interface.
type RateProvider interface {
	GetRateOnDate(ctx context.Context, currencyID int64, date string, companyID *int64) (*domaincurrency.CurrencyRate, error)
}

// Converter performs historical currency conversion against anchor-based
// exchange rates. The stored rate represents "units of this currency per one
// unit of the anchor (company) currency" — e.g. rate=3.75 for SAR means
// 1 USD = 3.75 SAR when USD is the anchor.
type Converter struct {
	rates RateProvider
}

// NewConverter constructs a Converter backed by the given RateProvider.
func NewConverter(rates RateProvider) *Converter {
	return &Converter{rates: rates}
}

// Convert converts an amount from fromCurrencyID to toCurrencyID using the
// exchange rates effective on (or before) the given date.
//
// amount_TO = amount_FROM * rate_TO / rate_FROM
func (c *Converter) Convert(ctx context.Context, amount float64, fromCurrencyID, toCurrencyID int64, date string, companyID *int64) (float64, error) {
	if c.rates == nil {
		return 0, platformerrors.Internal("currency converter has no rate provider")
	}

	if fromCurrencyID == toCurrencyID {
		return amount, nil
	}

	if date == "" {
		date = "9999-12-31"
	}

	fromRate, err := c.rates.GetRateOnDate(ctx, fromCurrencyID, date, companyID)
	if err != nil {
		return 0, platformerrors.Validation(
			fmt.Sprintf("no exchange rate available for currency %d on %s", fromCurrencyID, date),
			nil,
		)
	}

	toRate, err := c.rates.GetRateOnDate(ctx, toCurrencyID, date, companyID)
	if err != nil {
		return 0, platformerrors.Validation(
			fmt.Sprintf("no exchange rate available for currency %d on %s", toCurrencyID, date),
			nil,
		)
	}

	if fromRate.Rate <= 0 {
		return 0, platformerrors.Validation("invalid zero or negative source exchange rate", nil)
	}

	return amount * toRate.Rate / fromRate.Rate, nil
}

// ConvertWithFallback converts using the latest available rate when no
// historical rate exists for the requested date.
func (c *Converter) ConvertWithFallback(ctx context.Context, amount float64, fromCurrencyID, toCurrencyID int64, companyID *int64) (float64, error) {
	return c.Convert(ctx, amount, fromCurrencyID, toCurrencyID, "", companyID)
}
