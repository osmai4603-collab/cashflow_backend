package purchaseusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/stock"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

func TestPurchaseStockUseCase_CreateReceiptsFromOrder(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 1. Setup
	purchaseRepo := purchasestorage.NewMemoryRepo()
	stockRepo := stockstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()

	stockUC := stockusecase.New(stockRepo, partnerRepo, productRepo, nil, nil, logger)
	uc := purchaseusecase.NewPurchaseStockUseCase(purchaseRepo, stockRepo, stockUC, logger)

	// 2. Prepare Data
	vendor := &partner.Partner{
		ID:         5,
		Name:       "Test Vendor",
		IsSupplier: true,
		Active:     true,
	}
	if err := partnerRepo.Create(ctx, vendor); err != nil {
		t.Fatalf("failed to seed vendor: %v", err)
	}

	prod := &product.ProductTemplate{
		Name:      "Inventory Item",
		Type:      product.ProductTypeGoods,
		SalePrice: 100.0,
		CostPrice: 60.0,
		PurchaseOK: true,
		Active:    true,
	}
	if err := productRepo.CreateTemplate(ctx, prod); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	order := &purchase.PurchaseOrder{
		ID:        1,
		Name:      "PO/2026/00001",
		PartnerID: vendor.ID,
		DateOrder: time.Now(),
		State:     purchase.OrderStatePurchase,
		Lines: []purchase.PurchaseOrderLine{
			{
				ID:         20,
				ProductID:  prod.ID,
				ProductQty: 10,
				Name:       "Inventory Item",
			},
		},
	}
	purchaseRepo.CreateOrder(ctx, order)

	// 3. Execute
	err := uc.CreateReceiptsFromOrder(ctx, order)
	if err != nil {
		t.Fatalf("CreateReceiptsFromOrder failed: %v", err)
	}

	// 4. Verify
	if len(order.PickingIDs) != 1 {
		t.Errorf("expected 1 picking ID, got %d", len(order.PickingIDs))
	}
	if order.ProcurementGroupID == nil {
		t.Error("expected procurement group ID to be set")
	}

	picking, err := stockRepo.GetPickingByID(ctx, order.PickingIDs[0])
	if err != nil {
		t.Fatalf("failed to get picking: %v", err)
	}
	if picking.PickingType != stock.PickingTypeIncoming {
		t.Errorf("expected incoming picking, got %s", picking.PickingType)
	}
	if len(picking.Moves) != 1 {
		t.Errorf("expected 1 move, got %d", len(picking.Moves))
	}
	if picking.Moves[0].PurchaseLineID == nil || *picking.Moves[0].PurchaseLineID != order.Lines[0].ID {
		t.Errorf("expected purchase line ID %d, got %v", order.Lines[0].ID, picking.Moves[0].PurchaseLineID)
	}
}
