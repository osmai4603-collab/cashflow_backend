package money

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	ErrInvalidCurrency = errors.New("unsupported or invalid ISO 4217 currency")
	ErrNegativeAmount  = errors.New("negative amount is not permitted")
)

// CurrencySpec describes a currency's ISO 4217 minor-unit precision.
type CurrencySpec struct {
	Code        string
	Exponent    int32
	Description string
}

var SupportedCurrencies = map[string]CurrencySpec{
	"SAR": {Code: "SAR", Exponent: 2, Description: "Saudi Riyal"},
	"USD": {Code: "USD", Exponent: 2, Description: "US Dollar"},
	"EUR": {Code: "EUR", Exponent: 2, Description: "Euro"},
	"GBP": {Code: "GBP", Exponent: 2, Description: "British Pound"},
	"KWD": {Code: "KWD", Exponent: 3, Description: "Kuwaiti Dinar"},
	"BHD": {Code: "BHD", Exponent: 3, Description: "Bahraini Dinar"},
	"JPY": {Code: "JPY", Exponent: 0, Description: "Japanese Yen"},
}

// RoundToCurrency rounds amount to the currency's minor-unit precision.
func RoundToCurrency(amount decimal.Decimal, currencyCode string) (decimal.Decimal, error) {
	currency, ok := SupportedCurrencies[strings.ToUpper(currencyCode)]
	if !ok {
		return decimal.Zero, ErrInvalidCurrency
	}
	if amount.IsNegative() {
		return decimal.Zero, ErrNegativeAmount
	}

	return amount.Round(currency.Exponent), nil
}
