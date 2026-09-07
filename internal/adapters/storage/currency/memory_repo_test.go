package currencystorage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	currencystorage "cashflow_backend/internal/adapters/storage/currency"
	"cashflow_backend/internal/domain/currency"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestCurrencyMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := currencystorage.NewMemoryRepo()

	c := &currency.Currency{Name: "USD", FullName: "US Dollar", Symbol: "$", DecimalPlaces: 2}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("unexpected error creating currency: %v", err)
	}
	if c.ID <= 0 || !c.Active {
		t.Errorf("expected positive id and active=true")
	}

	fetched, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching currency: %v", err)
	}
	if fetched.Symbol != "$" {
		t.Errorf("expected symbol $, got %q", fetched.Symbol)
	}

	if err := repo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("unexpected error deleting currency: %v", err)
	}
	_, err = repo.GetByID(ctx, c.ID)
	if err == nil {
		t.Errorf("expected error getting soft-deleted currency, got nil")
	}
}

func TestCurrencyMemoryRepo_ListAndFilter(t *testing.T) {
	ctx := context.Background()
	repo := currencystorage.NewMemoryRepo()

	for _, c := range []*currency.Currency{
		{Name: "USD", FullName: "US Dollar", Symbol: "$", DecimalPlaces: 2},
		{Name: "EUR", FullName: "Euro", Symbol: "EUR", DecimalPlaces: 2},
		{Name: "SAR", FullName: "Saudi Riyal", Symbol: "SAR", DecimalPlaces: 2},
	} {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("failed to seed currency: %v", err)
		}
	}

	page := pagination.PageRequest{Page: 1, Limit: 10}
	res, err := repo.List(ctx, nil, page)
	if err != nil {
		t.Fatalf("unexpected error listing currencies: %v", err)
	}
	if res.TotalItems != 3 {
		t.Errorf("expected 3 currencies, got %d", res.TotalItems)
	}

	nameFilter := filter.NewFilter(filter.Criterion{Field: "name", Operator: filter.OpILike, Value: "us"})
	filtered, err := repo.List(ctx, nameFilter, page)
	if err != nil {
		t.Fatalf("unexpected error filtering currencies: %v", err)
	}
	if filtered.TotalItems != 1 {
		t.Errorf("expected 1 currency matching 'us', got %d", filtered.TotalItems)
	}
}

func TestCurrencyRateRepo_GetRateOnDate(t *testing.T) {
	ctx := context.Background()
	repo := currencystorage.NewMemoryRateRepo()

	// USD is the anchor: rate=1.0
	if err := repo.Create(ctx, &currency.CurrencyRate{CurrencyID: 1, Rate: 1.0, Date: "2024-01-01"}); err != nil {
		t.Fatalf("failed to create USD rate: %v", err)
	}
	// SAR history: 3.75 then 3.80
	if err := repo.Create(ctx, &currency.CurrencyRate{CurrencyID: 2, Rate: 3.75, Date: "2024-06-01"}); err != nil {
		t.Fatalf("failed to create SAR rate v1: %v", err)
	}
	if err := repo.Create(ctx, &currency.CurrencyRate{CurrencyID: 2, Rate: 3.80, Date: "2024-09-01"}); err != nil {
		t.Fatalf("failed to create SAR rate v2: %v", err)
	}

	// Historical lookup before the later rate
	rate, err := repo.GetRateOnDate(ctx, 2, "2024-07-01", nil)
	if err != nil {
		t.Fatalf("unexpected error getting rate on date: %v", err)
	}
	if rate.Rate != 3.75 {
		t.Errorf("expected 3.75 on 2024-07-01, got %f", rate.Rate)
	}

	// Future date picks the latest
	latest, err := repo.GetLatestRate(ctx, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error getting latest rate: %v", err)
	}
	if latest.Rate != 3.80 {
		t.Errorf("expected latest rate 3.80, got %f", latest.Rate)
	}

	// No rate before earliest date
	_, err = repo.GetRateOnDate(ctx, 2, "2020-01-01", nil)
	if err == nil {
		t.Errorf("expected error for date before earliest rate, got nil")
	}

	// ListRates sorted newest first
	res, err := repo.ListByCurrency(ctx, 2, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing rates: %v", err)
	}
	if res.TotalItems != 2 {
		t.Errorf("expected 2 rates, got %d", res.TotalItems)
	}
	if res.Items[0].Rate != 3.80 {
		t.Errorf("expected first item newest 3.80, got %f", res.Items[0].Rate)
	}
}

func TestCurrencyAndRateRepo_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := currencystorage.NewMemoryRepo()
	rateRepo := currencystorage.NewMemoryRateRepo()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_ = repo.Create(ctx, &currency.Currency{Name: fmt.Sprintf("C%02d", i), FullName: "x", Symbol: "x", DecimalPlaces: 2})
			_ = rateRepo.Create(ctx, &currency.CurrencyRate{CurrencyID: 1, Rate: 1, Date: fmt.Sprintf("2024-01-%02d", i%28+1)})
		}(i)
	}
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 25})
			_, _ = rateRepo.ListByCurrency(ctx, 1, pagination.PageRequest{Page: 1, Limit: 25})
		}()
	}
	wg.Wait()

	res, err := repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error after concurrent access: %v", err)
	}
	if res.TotalItems != n {
		t.Errorf("expected %d currencies, got %d", n, res.TotalItems)
	}
}
