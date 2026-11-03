package money

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestRoundToCurrency(t *testing.T) {
	tests := []struct {
		name         string
		amount       string
		currencyCode string
		want         string
	}{
		{name: "two decimal places", amount: "12.345", currencyCode: "SAR", want: "12.35"},
		{name: "case insensitive code", amount: "12.344", currencyCode: "usd", want: "12.34"},
		{name: "three decimal places", amount: "1.2345", currencyCode: "KWD", want: "1.235"},
		{name: "zero decimal places", amount: "10.5", currencyCode: "JPY", want: "11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RoundToCurrency(decimal.RequireFromString(tt.amount), tt.currencyCode)
			if err != nil {
				t.Fatalf("RoundToCurrency() error = %v", err)
			}
			if want := decimal.RequireFromString(tt.want); !got.Equal(want) {
				t.Errorf("RoundToCurrency() = %s, want %s", got, want)
			}
		})
	}
}

func TestRoundToCurrencyRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name         string
		amount       string
		currencyCode string
		wantErr      error
	}{
		{name: "unsupported currency", amount: "1", currencyCode: "XYZ", wantErr: ErrInvalidCurrency},
		{name: "negative amount", amount: "-1", currencyCode: "SAR", wantErr: ErrNegativeAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RoundToCurrency(decimal.RequireFromString(tt.amount), tt.currencyCode)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RoundToCurrency() error = %v, want %v", err, tt.wantErr)
			}
			if !got.IsZero() {
				t.Errorf("RoundToCurrency() amount = %s, want zero on error", got)
			}
		})
	}
}
