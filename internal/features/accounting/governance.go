package accounting

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// MakerCheckerSettings controls segregation of duties for manual journal entries.
// Threshold is expressed in the configured functional currency.
type MakerCheckerSettings struct {
	Threshold          decimal.Decimal
	FunctionalCurrency string
}

func (s *MakerCheckerSettings) Validate() error {
	if s.Threshold.IsNegative() {
		return errors.New("maker-checker threshold cannot be negative")
	}
	currency, err := normalizeCurrency(s.FunctionalCurrency, "SAR")
	if err != nil {
		return fmt.Errorf("maker-checker functional currency: %w", err)
	}
	s.FunctionalCurrency = currency
	return nil
}

func (s MakerCheckerSettings) enabled() bool {
	return s.Threshold.IsPositive()
}
