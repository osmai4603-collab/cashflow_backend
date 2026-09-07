package currency_test

import (
	"context"
	"testing"

	domaincurrency "cashflow_backend/internal/domain/currency"
	platformcurrency "cashflow_backend/internal/platform/currency"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type fakeRateProvider struct {
	rates map[int64][]domaincurrency.CurrencyRate
}

func (f *fakeRateProvider) GetRateOnDate(ctx context.Context, currencyID int64, date string, companyID *int64) (*domaincurrency.CurrencyRate, error) {
	var best *domaincurrency.CurrencyRate
	for i := range f.rates[currencyID] {
		r := f.rates[currencyID][i]
		if r.Date > date {
			continue
		}
		if best == nil || r.Date > best.Date {
			cp := r
			best = &cp
		}
	}
	if best == nil {
		return nil, platformerrors.NotFound("no rate")
	}
	return best, nil
}

func TestConverter_Convert(t *testing.T) {
	ctx := context.Background()
	provider := &fakeRateProvider{
		rates: map[int64][]domaincurrency.CurrencyRate{
			1: {{CurrencyID: 1, Rate: 1.0, Date: "2024-01-01"}},
			2: {{CurrencyID: 2, Rate: 3.75, Date: "2024-01-01"}},
		},
	}
	conv := platformcurrency.NewConverter(provider)

	// amount_TO = amount_FROM * rate_TO / rate_FROM
	got, err := conv.Convert(ctx, 100, 1, 2, "2024-01-01", nil)
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}
	if got != 375 {
		t.Errorf("expected 375, got %f", got)
	}

	// Same currency passes through unchanged
	got, err = conv.Convert(ctx, 88.5, 1, 1, "2024-01-01", nil)
	if err != nil {
		t.Fatalf("unexpected same-currency error: %v", err)
	}
	if got != 88.5 {
		t.Errorf("expected 88.5, got %f", got)
	}
}

func TestConverter_ConvertWithFallback(t *testing.T) {
	ctx := context.Background()
	provider := &fakeRateProvider{
		rates: map[int64][]domaincurrency.CurrencyRate{
			1: {{CurrencyID: 1, Rate: 1.0, Date: "2024-01-01"}},
			2: {{CurrencyID: 2, Rate: 3.9, Date: "2024-06-01"}},
		},
	}
	conv := platformcurrency.NewConverter(provider)

	// No date passed: uses latest available rates
	got, err := conv.ConvertWithFallback(ctx, 10, 1, 2, nil)
	if err != nil {
		t.Fatalf("unexpected fallback conversion error: %v", err)
	}
	if got != 39 {
		t.Errorf("expected 39, got %f", got)
	}
}

func TestConverter_Errors(t *testing.T) {
	ctx := context.Background()
	provider := &fakeRateProvider{rates: map[int64][]domaincurrency.CurrencyRate{}}
	conv := platformcurrency.NewConverter(provider)

	// Missing rate for target currency
	_, err := conv.Convert(ctx, 10, 1, 2, "2024-01-01", nil)
	if err == nil {
		t.Fatalf("expected error for missing rates, got nil")
	}

	// Nil provider
	convNil := platformcurrency.NewConverter(nil)
	_, err = convNil.Convert(ctx, 10, 1, 2, "2024-01-01", nil)
	if err == nil {
		t.Fatalf("expected error for nil provider, got nil")
	}
}
