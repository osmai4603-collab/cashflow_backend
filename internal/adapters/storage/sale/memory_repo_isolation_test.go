package salestorage

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/platform/audit"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepoCompanyIsolation(t *testing.T) {
	repo := NewMemoryRepo()
	companyOne := audit.WithCompanyID(context.Background(), 1)
	companyTwo := audit.WithCompanyID(context.Background(), 2)

	first := &sale.SaleOrder{Name: "SO/1", CompanyID: nil}
	if err := repo.CreateOrder(companyOne, first); err != nil {
		t.Fatalf("create first order: %v", err)
	}
	second := &sale.SaleOrder{Name: "SO/2", CompanyID: nil}
	if err := repo.CreateOrder(companyTwo, second); err != nil {
		t.Fatalf("create second order: %v", err)
	}

	if _, err := repo.GetOrderByID(companyTwo, first.ID); err == nil {
		t.Fatal("expected cross-company order to be hidden")
	}
	page, err := repo.ListOrders(companyTwo, nil, pagination.PageRequest{Limit: 20})
	if err != nil {
		t.Fatalf("list company two: %v", err)
	}
	if page.TotalItems != 1 || page.Items[0].ID != second.ID {
		t.Fatalf("company two orders = %+v, want only %d", page.Items, second.ID)
	}
}
