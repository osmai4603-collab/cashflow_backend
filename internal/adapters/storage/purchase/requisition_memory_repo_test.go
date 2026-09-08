package purchasestorage

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/platform/audit"
)

func TestMemoryRequisitionRepoCompanyIsolation(t *testing.T) {
	repo := NewMemoryRequisitionRepo()
	companyOne := audit.WithCompanyID(context.Background(), 1)
	companyTwo := audit.WithCompanyID(context.Background(), 2)
	vendorID := int64(10)
	requisition := &purchase.PurchaseRequisition{
		Type:     purchase.RequisitionTemplate,
		VendorID: &vendorID,
		Lines:    []purchase.PurchaseRequisitionLine{{ProductID: 1, ProductQty: 1}},
	}

	if err := repo.CreateRequisition(companyOne, requisition); err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}
	if _, err := repo.GetRequisitionByID(companyTwo, requisition.ID); err == nil {
		t.Fatal("expected a requisition from another company to be hidden")
	}
	if _, err := repo.GetRequisitionByID(companyOne, requisition.ID); err != nil {
		t.Fatalf("same-company requisition should be readable: %v", err)
	}
}

func TestMemoryRequisitionRepoSupplierInfoLifecycle(t *testing.T) {
	repo := NewMemoryRequisitionRepo()
	vendorID := int64(10)
	requisition := &purchase.PurchaseRequisition{
		ID:       7,
		Type:     purchase.RequisitionBlanketOrder,
		VendorID: &vendorID,
		Lines:    []purchase.PurchaseRequisitionLine{{ID: 11, ProductID: 1, ProductQty: 2, PriceUnit: 5}},
	}

	if err := repo.CreateForRequisition(context.Background(), requisition); err != nil {
		t.Fatalf("CreateForRequisition() error = %v", err)
	}
	if len(repo.supplierInfos[requisition.ID]) != 1 || repo.supplierInfos[requisition.ID][0].Price != 5 {
		t.Fatalf("expected one supplier info with price 5, got %#v", repo.supplierInfos[requisition.ID])
	}
	requisition.Lines[0].PriceUnit = 6
	if err := repo.UpdatePricesForRequisition(context.Background(), requisition); err != nil {
		t.Fatalf("UpdatePricesForRequisition() error = %v", err)
	}
	if repo.supplierInfos[requisition.ID][0].Price != 6 {
		t.Fatalf("expected supplier price 6, got %v", repo.supplierInfos[requisition.ID][0].Price)
	}
	if err := repo.DeleteForRequisition(context.Background(), requisition.ID); err != nil {
		t.Fatalf("DeleteForRequisition() error = %v", err)
	}
	if len(repo.supplierInfos[requisition.ID]) != 0 {
		t.Fatal("expected supplier info to be deleted")
	}
}
