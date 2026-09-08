package purchaseusecase_test

import (
	"context"
	"testing"
	"time"

	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	"cashflow_backend/internal/domain/purchase"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
)

func TestCreateRequisition_Valid(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)

	start := time.Now().UTC()
	end := start.Add(30 * 24 * time.Hour)
	vendorID := int64(10)

	req, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Name:      "RQ/2026/0001",
		Type:      purchase.RequisitionBlanketOrder,
		VendorID:  &vendorID,
		UserID:    20,
		DateStart: &start,
		DateEnd:   &end,
		Lines: []purchaseusecase.CreateRequisitionLineInput{
			{ProductID: 101, ProductQty: 5, PriceUnit: 25.5},
		},
	})
	if err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}
	if req.State != purchase.RequisitionDraft {
		t.Fatalf("expected draft state, got %s", req.State)
	}
	if len(req.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(req.Lines))
	}
}

func TestCreateRequisition_RejectsEmptyLines(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)

	start := time.Now().UTC()
	end := start.Add(7 * 24 * time.Hour)
	vendorID := int64(10)

	_, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Name:      "RQ/2026/0002",
		Type:      purchase.RequisitionBlanketOrder,
		VendorID:  &vendorID,
		UserID:    20,
		DateStart: &start,
		DateEnd:   &end,
	})
	if err == nil {
		t.Fatal("expected validation error for empty lines")
	}
}

func TestConfirmRequisition_TransitionsToConfirmed(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)

	start := time.Now().UTC()
	end := start.Add(14 * 24 * time.Hour)
	vendorID := int64(30)
	created, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Name:      "RQ/2026/0003",
		Type:      purchase.RequisitionBlanketOrder,
		VendorID:  &vendorID,
		UserID:    40,
		DateStart: &start,
		DateEnd:   &end,
		Lines: []purchaseusecase.CreateRequisitionLineInput{
			{ProductID: 201, ProductQty: 2, PriceUnit: 12.0},
		},
	})
	if err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}

	confirmed, err := uc.ConfirmRequisition(ctx, created.ID)
	if err != nil {
		t.Fatalf("ConfirmRequisition() error = %v", err)
	}
	if confirmed.State != purchase.RequisitionConfirmed {
		t.Fatalf("expected confirmed state, got %s", confirmed.State)
	}
}

func TestCreatePurchaseOrderFromRequisition_LinksRequisitionID(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)

	start := time.Now().UTC()
	end := start.Add(10 * 24 * time.Hour)
	vendorID := int64(99)
	req, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Name:      "RQ/2026/0004",
		Type:      purchase.RequisitionTemplate,
		VendorID:  &vendorID,
		UserID:    88,
		DateStart: &start,
		DateEnd:   &end,
		Lines: []purchaseusecase.CreateRequisitionLineInput{
			{ProductID: 501, ProductQty: 7, PriceUnit: 9.5},
		},
	})
	if err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}
	if _, err := uc.ConfirmRequisition(ctx, req.ID); err != nil {
		t.Fatalf("ConfirmRequisition() error = %v", err)
	}

	po, err := uc.CreatePurchaseOrderFromRequisition(ctx, req.ID)
	if err != nil {
		t.Fatalf("CreatePurchaseOrderFromRequisition() error = %v", err)
	}
	if po.RequisitionID == nil || *po.RequisitionID != req.ID {
		t.Fatalf("expected requisition_id %d to be linked, got %#v", req.ID, po.RequisitionID)
	}
	if po.Lines[0].ProductQty != 7 {
		t.Fatalf("purchase template should copy line quantity, got %v", po.Lines[0].ProductQty)
	}
	loaded, err := uc.GetRequisitionByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetRequisitionByID() error = %v", err)
	}
	if len(loaded.PurchaseOrderIDs) != 1 || loaded.PurchaseOrderIDs[0] != po.ID {
		t.Fatalf("expected linked purchase order IDs [%d], got %v", po.ID, loaded.PurchaseOrderIDs)
	}
}

func TestCreatePurchaseOrderFromBlanketOrderStartsAtZeroQuantity(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)
	vendorID := int64(100)
	req, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Type: purchase.RequisitionBlanketOrder, VendorID: &vendorID, UserID: 1,
		Lines: []purchaseusecase.CreateRequisitionLineInput{{ProductID: 1, ProductQty: 8, PriceUnit: 4}},
	})
	if err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}
	if _, err := uc.ConfirmRequisition(ctx, req.ID); err != nil {
		t.Fatalf("ConfirmRequisition() error = %v", err)
	}
	po, err := uc.CreatePurchaseOrderFromRequisition(ctx, req.ID)
	if err != nil {
		t.Fatalf("CreatePurchaseOrderFromRequisition() error = %v", err)
	}
	if po.Lines[0].ProductQty != 0 {
		t.Fatalf("blanket order RFQ should start at zero quantity, got %v", po.Lines[0].ProductQty)
	}
}

func TestCloseRequisition_RejectsOpenPurchaseOrders(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)
	vendorID := int64(600)
	req, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Name: "RQ/2026/0005", Type: purchase.RequisitionBlanketOrder, VendorID: &vendorID,
		UserID: 1, Lines: []purchaseusecase.CreateRequisitionLineInput{{ProductID: 1, ProductQty: 1, PriceUnit: 5}},
	})
	if err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}
	if _, err := uc.ConfirmRequisition(ctx, req.ID); err != nil {
		t.Fatalf("ConfirmRequisition() error = %v", err)
	}
	if _, err := uc.CreatePurchaseOrderFromRequisition(ctx, req.ID); err != nil {
		t.Fatalf("CreatePurchaseOrderFromRequisition() error = %v", err)
	}
	if _, err := uc.CloseRequisition(ctx, req.ID); err == nil {
		t.Fatal("expected close to reject open purchase order")
	}
	loaded, err := uc.GetRequisitionByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetRequisitionByID() error = %v", err)
	}
	if loaded.State != purchase.RequisitionConfirmed {
		t.Fatalf("failed close must preserve confirmed state, got %s", loaded.State)
	}
}

func TestCancelRequisitionCancelsDraftPurchaseOrders(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)
	vendorID := int64(601)
	req, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Name: "RQ/2026/0007", Type: purchase.RequisitionTemplate, VendorID: &vendorID,
		UserID: 1, Lines: []purchaseusecase.CreateRequisitionLineInput{{ProductID: 2, ProductQty: 1}},
	})
	if err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}
	if _, err := uc.ConfirmRequisition(ctx, req.ID); err != nil {
		t.Fatalf("ConfirmRequisition() error = %v", err)
	}
	po, err := uc.CreatePurchaseOrderFromRequisition(ctx, req.ID)
	if err != nil {
		t.Fatalf("CreatePurchaseOrderFromRequisition() error = %v", err)
	}
	if _, err := uc.CancelRequisition(ctx, req.ID); err != nil {
		t.Fatalf("CancelRequisition() error = %v", err)
	}
	loadedPO, err := poRepo.GetOrderByID(ctx, po.ID)
	if err != nil {
		t.Fatalf("GetOrderByID() error = %v", err)
	}
	if loadedPO.State != purchase.OrderStateCancel {
		t.Fatalf("expected draft RFQ to be cancelled, got %s", loadedPO.State)
	}
}

func TestUpdateAndDeleteRequisition(t *testing.T) {
	ctx := context.Background()
	reqRepo := purchasestorage.NewMemoryRequisitionRepo()
	poRepo := purchasestorage.NewMemoryRepo()
	uc := purchaseusecase.NewRequisitionUseCase(reqRepo, poRepo)
	vendorID := int64(700)
	req, err := uc.CreateRequisition(ctx, purchaseusecase.CreatePurchaseRequisitionInput{
		Name: "RQ/2026/0006", Type: purchase.RequisitionBlanketOrder, VendorID: &vendorID,
		UserID: 1, Lines: []purchaseusecase.CreateRequisitionLineInput{{ProductID: 2, ProductQty: 1}},
	})
	if err != nil {
		t.Fatalf("CreateRequisition() error = %v", err)
	}
	updated, err := uc.UpdateRequisition(ctx, req.ID, purchaseusecase.CreatePurchaseRequisitionInput{
		Name: "RQ/2026/0006-UPDATED", Type: purchase.RequisitionTemplate, VendorID: &vendorID,
		UserID: 2, Description: "updated", Lines: []purchaseusecase.CreateRequisitionLineInput{{ProductID: 3, ProductQty: 4}},
	})
	if err != nil {
		t.Fatalf("UpdateRequisition() error = %v", err)
	}
	if updated.Name != "RQ/2026/0006-UPDATED" || updated.Lines[0].ProductID != 3 {
		t.Fatalf("requisition was not updated: %#v", updated)
	}
	if err := uc.DeleteRequisition(ctx, req.ID); err != nil {
		t.Fatalf("DeleteRequisition() error = %v", err)
	}
}
