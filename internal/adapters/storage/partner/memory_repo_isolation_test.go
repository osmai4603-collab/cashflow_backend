package partnerstorage

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/platform/audit"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepoCompanyIsolation(t *testing.T) {
	repo := NewMemoryRepo()
	companyOne := audit.WithCompanyID(context.Background(), 1)
	companyTwo := audit.WithCompanyID(context.Background(), 2)

	first := &partner.Partner{Name: "Company One", Type: partner.PartnerTypeCompany}
	if err := repo.Create(companyOne, first); err != nil {
		t.Fatalf("create first partner: %v", err)
	}
	second := &partner.Partner{Name: "Company Two", Type: partner.PartnerTypeCompany}
	if err := repo.Create(companyTwo, second); err != nil {
		t.Fatalf("create second partner: %v", err)
	}

	if _, err := repo.GetByID(companyTwo, first.ID); err == nil {
		t.Fatal("expected cross-company read to be hidden")
	}
	page, err := repo.List(companyTwo, nil, pagination.PageRequest{Limit: 20})
	if err != nil {
		t.Fatalf("list company two: %v", err)
	}
	if page.TotalItems != 1 || page.Items[0].ID != second.ID {
		t.Fatalf("company two partners = %+v, want only %d", page.Items, second.ID)
	}
}
