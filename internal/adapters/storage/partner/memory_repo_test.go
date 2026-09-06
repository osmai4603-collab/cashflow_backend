package partnerstorage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := partnerstorage.NewMemoryRepo()

	// 1. Create
	p := &partner.Partner{
		Name:       "Test Partner",
		Email:      "test@example.com",
		Type:       partner.PartnerTypeCompany,
		IsCustomer: true,
		IsSupplier: false,
		City:       "Riyadh",
		Country:    "Saudi Arabia",
	}

	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("unexpected error creating partner: %v", err)
	}

	if p.ID <= 0 {
		t.Errorf("expected positive ID, got %d", p.ID)
	}
	if !p.Active {
		t.Errorf("expected partner to be active")
	}

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching partner: %v", err)
	}
	if fetched.Name != p.Name {
		t.Errorf("expected name %q, got %q", p.Name, fetched.Name)
	}
	if fetched.City != "Riyadh" {
		t.Errorf("expected city Riyadh, got %q", fetched.City)
	}

	// 3. Update
	fetched.City = "Jeddah"
	fetched.IsSupplier = true
	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("unexpected error updating partner: %v", err)
	}

	updated, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected error re-fetching updated partner: %v", err)
	}
	if updated.City != "Jeddah" {
		t.Errorf("expected updated city Jeddah, got %q", updated.City)
	}
	if !updated.IsSupplier {
		t.Errorf("expected IsSupplier to be true")
	}

	// 4. Soft Delete
	if err := repo.Delete(ctx, p.ID); err != nil {
		t.Fatalf("unexpected error deleting partner: %v", err)
	}

	// Should not be retrievable via GetByID
	_, err = repo.GetByID(ctx, p.ID)
	if err == nil {
		t.Errorf("expected error getting soft-deleted partner, got nil")
	}
}

func TestMemoryRepo_ListAndFilter(t *testing.T) {
	ctx := context.Background()
	repo := partnerstorage.NewMemoryRepo()

	// Seed multiple partners
	partners := []*partner.Partner{
		{Name: "Customer Alpha", Type: partner.PartnerTypeCompany, IsCustomer: true, IsSupplier: false, City: "Riyadh"},
		{Name: "Customer Beta", Type: partner.PartnerTypeIndividual, IsCustomer: true, IsSupplier: false, City: "Dammam"},
		{Name: "Supplier Gamma", Type: partner.PartnerTypeCompany, IsCustomer: false, IsSupplier: true, City: "Jeddah"},
		{Name: "Both Delta", Type: partner.PartnerTypeCompany, IsCustomer: true, IsSupplier: true, City: "Riyadh"},
	}

	for _, p := range partners {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("failed to seed partner: %v", err)
		}
	}

	// 1. List All Active
	pageReq := pagination.PageRequest{Page: 1, Limit: 10, SortBy: "name", SortOrder: "asc"}
	res, err := repo.List(ctx, nil, pageReq)
	if err != nil {
		t.Fatalf("unexpected error listing partners: %v", err)
	}
	if res.TotalItems != 4 {
		t.Errorf("expected 4 total items, got %d", res.TotalItems)
	}
	if len(res.Items) != 4 {
		t.Errorf("expected 4 returned items, got %d", len(res.Items))
	}
	if res.Items[0].Name != "Both Delta" {
		t.Errorf("expected first sorted item 'Both Delta', got %q", res.Items[0].Name)
	}

	// 2. List Customers Only
	custRes, err := repo.ListCustomers(ctx, pageReq)
	if err != nil {
		t.Fatalf("unexpected error listing customers: %v", err)
	}
	if custRes.TotalItems != 3 {
		t.Errorf("expected 3 customers, got %d", custRes.TotalItems)
	}

	// 3. List Suppliers Only
	suppRes, err := repo.ListSuppliers(ctx, pageReq)
	if err != nil {
		t.Fatalf("unexpected error listing suppliers: %v", err)
	}
	if suppRes.TotalItems != 2 {
		t.Errorf("expected 2 suppliers, got %d", suppRes.TotalItems)
	}

	// 4. Custom Filter: City = Riyadh
	riyadhFilter := filter.NewFilter(
		filter.Criterion{Field: "city", Operator: filter.OpEqual, Value: "Riyadh"},
	)
	cityRes, err := repo.List(ctx, riyadhFilter, pageReq)
	if err != nil {
		t.Fatalf("unexpected error listing with filter: %v", err)
	}
	if cityRes.TotalItems != 2 {
		t.Errorf("expected 2 partners in Riyadh, got %d", cityRes.TotalItems)
	}

	// 5. Pagination Limit & Offset
	pagedReq := pagination.PageRequest{Page: 2, Limit: 2, SortBy: "id", SortOrder: "asc"}
	pagedRes, err := repo.List(ctx, nil, pagedReq)
	if err != nil {
		t.Fatalf("unexpected error listing page 2: %v", err)
	}
	if pagedRes.TotalPages != 2 {
		t.Errorf("expected 2 total pages, got %d", pagedRes.TotalPages)
	}
	if len(pagedRes.Items) != 2 {
		t.Errorf("expected 2 items on page 2, got %d", len(pagedRes.Items))
	}
}

func TestMemoryRepo_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := partnerstorage.NewMemoryRepo()

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			p := &partner.Partner{
				Name: fmt.Sprintf("Partner %d", n),
				Type: partner.PartnerTypeIndividual,
			}
			_ = repo.Create(ctx, p)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 20})
		}()
	}

	wg.Wait()

	res, err := repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error listing after concurrent test: %v", err)
	}
	if res.TotalItems != numGoroutines {
		t.Errorf("expected %d total items, got %d", numGoroutines, res.TotalItems)
	}
}
