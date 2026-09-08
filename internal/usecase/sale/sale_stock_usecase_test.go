package saleusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
	saleusecase "cashflow_backend/internal/usecase/sale"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

func TestSaleStockUseCase_CreateDeliveriesFromOrder(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 1. Setup
	saleRepo := salestorage.NewMemoryRepo()
	stockRepo := stockstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()

	stockUC := stockusecase.New(stockRepo, partnerRepo, productRepo, saleRepo, nil, logger)
	uc := saleusecase.NewSaleStockUseCase(saleRepo, stockRepo, stockUC, logger)

	// 2. Prepare Data
	productID := int64(101)
	order := &sale.SaleOrder{
		ID:        1,
		Name:      "SO/2026/00001",
		PartnerID: 1,
		DateOrder: time.Now(),
		State:     sale.OrderStateSale,
		Lines: []sale.SaleOrderLine{
			{
				ID:            10,
				ProductID:     productID,
				ProductUomQty: 5,
				Name:          "Test Product",
			},
		},
	}
	saleRepo.CreateOrder(ctx, order)

	// 3. Execute
	err := uc.CreateDeliveriesFromOrder(ctx, order)
	if err != nil {
		t.Fatalf("CreateDeliveriesFromOrder failed: %v", err)
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
	if picking.PickingType != stock.PickingTypeOutgoing {
		t.Errorf("expected outgoing picking, got %s", picking.PickingType)
	}
	if len(picking.Moves) != 1 {
		t.Errorf("expected 1 move, got %d", len(picking.Moves))
	}
	if *picking.Moves[0].SaleLineID != 10 {
		t.Errorf("expected sale line ID 10, got %v", picking.Moves[0].SaleLineID)
	}
}

func TestSaleStockUseCase_Conflict(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	uc := saleusecase.NewSaleStockUseCase(nil, nil, nil, logger)

	order := &sale.SaleOrder{
		State: sale.OrderStateDraft,
	}

	err := uc.CreateDeliveriesFromOrder(ctx, order)
	if err == nil {
		t.Fatal("expected conflict error for draft order, got nil")
	}
}
