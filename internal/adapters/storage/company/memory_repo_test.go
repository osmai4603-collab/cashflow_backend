package companystorage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	companystorage "cashflow_backend/internal/adapters/storage/company"
	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestCompanyMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := companystorage.NewMemoryRepo()

	c := &company.Company{Name: "Acme", CurrencyID: 1, City: "Riyadh"}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("unexpected error creating company: %v", err)
	}
	if c.ID <= 0 {
		t.Errorf("expected positive ID, got %d", c.ID)
	}
	if !c.Active {
		t.Errorf("expected company to be active")
	}

	fetched, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching company: %v", err)
	}
	if fetched.City != "Riyadh" {
		t.Errorf("expected city Riyadh, got %q", fetched.City)
	}

	fetched.Name = "Acme Inc"
	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("unexpected error updating company: %v", err)
	}
	updated, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("unexpected error re-fetching: %v", err)
	}
	if updated.Name != "Acme Inc" {
		t.Errorf("expected updated name, got %q", updated.Name)
	}

	if err := repo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("unexpected error deleting company: %v", err)
	}
	_, err = repo.GetByID(ctx, c.ID)
	if err == nil {
		t.Errorf("expected error getting soft-deleted company, got nil")
	}
}

func TestCompanyMemoryRepo_ListAndGetDefault(t *testing.T) {
	ctx := context.Background()
	repo := companystorage.NewMemoryRepo()

	docs := []*company.Company{
		{Name: "Alpha", CurrencyID: 1, Country: "SA"},
		{Name: "Beta", CurrencyID: 2, Country: "AE"},
		{Name: "Gamma", CurrencyID: 1, Country: "SA"},
	}
	for _, c := range docs {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("failed to seed company: %v", err)
		}
	}

	page := pagination.PageRequest{Page: 1, Limit: 3}
	res, err := repo.List(ctx, nil, page)
	if err != nil {
		t.Fatalf("unexpected error listing companies: %v", err)
	}
	if res.TotalItems != 3 {
		t.Errorf("expected 3 companies, got %d", res.TotalItems)
	}

	// Filter by country
	countryFilter := filter.NewFilter(filter.Criterion{Field: "country", Operator: filter.OpEqual, Value: "SA"})
	countryRes, err := repo.List(ctx, countryFilter, page)
	if err != nil {
		t.Fatalf("unexpected error filtering companies: %v", err)
	}
	if countryRes.TotalItems != 2 {
		t.Errorf("expected 2 companies in SA, got %d", countryRes.TotalItems)
	}

	// GetDefaultCompany returns id=1
	def, err := repo.GetDefaultCompany(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting default company: %v", err)
	}
	if def.ID != 1 {
		t.Errorf("expected default company id=1, got %d", def.ID)
	}
}

func TestCompanyMemoryRepo_LandedCostJournal(t *testing.T) {
	ctx := context.Background()
	repo := companystorage.NewMemoryRepo()
	journalID := int64(42)

	c := &company.Company{Name: "Freight Co", CurrencyID: 1, LandedCostJournalID: &journalID}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("unexpected error creating company: %v", err)
	}

	fetched, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching company: %v", err)
	}
	if fetched.LandedCostJournalID == nil || *fetched.LandedCostJournalID != journalID {
		t.Fatalf("expected landed cost journal ID %d, got %#v", journalID, fetched.LandedCostJournalID)
	}
}

func TestCompanyMemoryRepo_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := companystorage.NewMemoryRepo()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_ = repo.Create(ctx, &company.Company{Name: fmt.Sprintf("C%d", i), CurrencyID: 1})
		}(i)
	}
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 25})
		}()
	}
	wg.Wait()

	res, err := repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error after concurrent access: %v", err)
	}
	if res.TotalItems != n {
		t.Errorf("expected %d companies, got %d", n, res.TotalItems)
	}
}
