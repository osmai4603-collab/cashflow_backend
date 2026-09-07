package currencyusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	currencystorage "cashflow_backend/internal/adapters/storage/currency"
	platformcurrency "cashflow_backend/internal/platform/currency"
	"cashflow_backend/internal/platform/pagination"
	currencyusecase "cashflow_backend/internal/usecase/currency"
)

func setupTestUseCase() (*currencyusecase.CurrencyUseCase, *currencystorage.MemoryRepo, *currencystorage.MemoryRateRepo) {
	repo := currencystorage.NewMemoryRepo()
	rateRepo := currencystorage.NewMemoryRateRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	converter := platformcurrency.NewConverter(rateRepo)
	uc := currencyusecase.New(repo, rateRepo, converter, logger)
	return uc, repo, rateRepo
}

func TestCurrencyUseCase_CreateAndListCurrency(t *testing.T) {
	uc, _, _ := setupTestUseCase()
	ctx := context.Background()

	// 1. Success with normalization
	c, err := uc.CreateCurrency(ctx, currencyusecase.CreateCurrencyInput{
		Name:     "sar",
		FullName: "Saudi Riyal",
		Symbol:   "SAR",
	})
	if err != nil {
		t.Fatalf("unexpected error creating currency: %v", err)
	}
	if c.ID <= 0 {
		t.Errorf("expected positive currency ID, got %d", c.ID)
	}
	if c.Name != "SAR" {
		t.Errorf("expected normalized uppercase name SAR, got %q", c.Name)
	}
	if c.DecimalPlaces != 2 {
		t.Errorf("expected default decimal_places 2, got %d", c.DecimalPlaces)
	}

	// 2. Validation failures
	_, err = uc.CreateCurrency(ctx, currencyusecase.CreateCurrencyInput{
		Name:     "USD",
		FullName: "US Dollar",
		Symbol:   "$",
	})
	if err != nil {
		t.Fatalf("unexpected error creating a second currency: %v", err)
	}
	_, err = uc.CreateCurrency(ctx, currencyusecase.CreateCurrencyInput{Name: "us"})
	if err == nil {
		t.Fatalf("expected error for 2-letter code, got nil")
	}

	// 3. List
	res, err := uc.ListCurrencies(ctx, nil, pageReq())
	if err != nil {
		t.Fatalf("unexpected error listing currencies: %v", err)
	}
	if res.TotalItems != 2 {
		t.Errorf("expected 2 currencies, got %d", res.TotalItems)
	}
}

func TestCurrencyUseCase_CreateRate(t *testing.T) {
	uc, _, _ := setupTestUseCase()
	ctx := context.Background()

	sar, err := uc.CreateCurrency(ctx, currencyusecase.CreateCurrencyInput{
		Name: "SAR", FullName: "Saudi Riyal", Symbol: "SAR",
	})
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}

	// 1. Success
	rate, err := uc.CreateRate(ctx, currencyusecase.CreateRateInput{
		CurrencyID: sar.ID,
		Rate:       3.75,
		Date:       "2024-01-15",
	})
	if err != nil {
		t.Fatalf("unexpected error creating rate: %v", err)
	}
	if rate.ID <= 0 || rate.Rate != 3.75 {
		t.Errorf("expected positive id and rate 3.75, got id=%d rate=%f", rate.ID, rate.Rate)
	}

	// 2. Validation: nonexistent currency
	_, err = uc.CreateRate(ctx, currencyusecase.CreateRateInput{
		CurrencyID: 9999,
		Rate:       1,
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatalf("expected error for nonexistent currency, got nil")
	}

	// 3. Validation: non-positive rate
	_, err = uc.CreateRate(ctx, currencyusecase.CreateRateInput{
		CurrencyID: sar.ID,
		Rate:       0,
		Date:       "2024-01-15",
	})
	if err == nil {
		t.Fatalf("expected error for zero rate, got nil")
	}

	// 4. ListRates
	res, err := uc.ListRates(ctx, sar.ID, pageReq())
	if err != nil {
		t.Fatalf("unexpected error listing rates: %v", err)
	}
	if res.TotalItems != 1 {
		t.Errorf("expected 1 rate, got %d", res.TotalItems)
	}
}

func TestCurrencyUseCase_Convert(t *testing.T) {
	uc, _, _ := setupTestUseCase()
	ctx := context.Background()

	usd, err := uc.CreateCurrency(ctx, currencyusecase.CreateCurrencyInput{
		Name: "USD", FullName: "US Dollar", Symbol: "$",
	})
	if err != nil {
		t.Fatalf("failed to create USD: %v", err)
	}
	sar, err := uc.CreateCurrency(ctx, currencyusecase.CreateCurrencyInput{
		Name: "SAR", FullName: "Saudi Riyal", Symbol: "SAR",
	})
	if err != nil {
		t.Fatalf("failed to create SAR: %v", err)
	}

	for _, r := range []currencyusecase.CreateRateInput{
		{CurrencyID: usd.ID, Rate: 1.0, Date: "2024-01-01"},
		{CurrencyID: sar.ID, Rate: 3.75, Date: "2024-01-01"},
	} {
		if _, err := uc.CreateRate(ctx, r); err != nil {
			t.Fatalf("failed to create rate: %v", err)
		}
	}

	// 100 USD -> SAR should be 375 (USD is anchor rate=1.0)
	res, err := uc.Convert(ctx, currencyusecase.ConvertInput{
		Amount:       100,
		FromCurrency: usd.ID,
		ToCurrency:   sar.ID,
		Date:         "2024-01-01",
	})
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}
	if res.Converted != 375 {
		t.Errorf("expected 375 SAR, got %f", res.Converted)
	}

	// Same currency amount is unchanged
	res2, err := uc.Convert(ctx, currencyusecase.ConvertInput{
		Amount:       42.5,
		FromCurrency: usd.ID,
		ToCurrency:   usd.ID,
		Date:         "2024-01-01",
	})
	if err != nil {
		t.Fatalf("unexpected same-currency conversion error: %v", err)
	}
	if res2.Converted != 42.5 {
		t.Errorf("expected unchanged 42.5, got %f", res2.Converted)
	}

	// Negative amount validation
	_, err = uc.Convert(ctx, currencyusecase.ConvertInput{
		Amount:       -1,
		FromCurrency: usd.ID,
		ToCurrency:   sar.ID,
	})
	if err == nil {
		t.Fatalf("expected error for negative amount, got nil")
	}
}

func pageReq() pagination.PageRequest {
	return pagination.PageRequest{Page: 1, Limit: 10}
}
