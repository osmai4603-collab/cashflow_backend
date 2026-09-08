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
	productID := int64(201)
	order := &purchase.PurchaseOrder{
		ID:        1,
		Name:      "PO/2026/00001",
		PartnerID: 5,
		DateOrder: time.Now(),
		State:     purchase.OrderStatePurchase,
		Lines: []purchase.PurchaseOrderLine{
			{
				ID:         20,
				ProductID:  productID,
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
	if *picking.Moves[0].PurchaseLineID != 20 {
		t.Errorf("expected purchase line ID 20, got %v", picking.Moves[0].PurchaseLineID)
	}
}
