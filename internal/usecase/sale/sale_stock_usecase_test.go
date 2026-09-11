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
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
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
	customer := &partner.Partner{
		ID:         1,
		Name:       "Test Customer",
		IsCustomer: true,
		Active:     true,
	}
	if err := partnerRepo.Create(ctx, customer); err != nil {
		t.Fatalf("failed to seed customer: %v", err)
	}

	prod := &product.ProductTemplate{
		Name:      "Test Product",
		Type:      product.ProductTypeGoods,
		SalePrice: 100.0,
		CostPrice: 60.0,
		SaleOK:    true,
		Active:    true,
	}
	if err := productRepo.CreateTemplate(ctx, prod); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}
	productID := prod.ID
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
	if picking.Moves[0].SaleLineID == nil || *picking.Moves[0].SaleLineID != order.Lines[0].ID {
		t.Errorf("expected sale line ID %d, got %v", order.Lines[0].ID, picking.Moves[0].SaleLineID)
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

func TestSaleStockUseCase_UpdateDeliveredQuantitiesAndBackorder(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	saleRepo := salestorage.NewMemoryRepo()
	stockRepo := stockstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()

	stockUC := stockusecase.New(stockRepo, partnerRepo, productRepo, saleRepo, nil, logger)
	uc := saleusecase.NewSaleStockUseCase(saleRepo, stockRepo, stockUC, logger)

	customer := &partner.Partner{
		Name:       "Customer B",
		IsCustomer: true,
		Active:     true,
	}
	_ = partnerRepo.Create(ctx, customer)

	prod := &product.ProductTemplate{
		Name:      "Product B",
		Type:      product.ProductTypeGoods,
		SalePrice: 50.0,
		Active:    true,
	}
	_ = productRepo.CreateTemplate(ctx, prod)

	order := &sale.SaleOrder{
		PartnerID: customer.ID,
		DateOrder: time.Now(),
		State:     sale.OrderStateSale,
		Lines: []sale.SaleOrderLine{
			{
				ProductID:     prod.ID,
				ProductUomQty: 10,
				Name:          "Product B",
			},
		},
	}
	_ = saleRepo.CreateOrder(ctx, order)
	_ = uc.CreateDeliveriesFromOrder(ctx, order)

	pickingID := order.PickingIDs[0]
	picking, _ := stockRepo.GetPickingByID(ctx, pickingID)

	// Simulate partial delivery of 6 out of 10
	picking.Moves[0].QuantityDone = 6
	picking.Moves[0].State = stock.MoveStateDone
	_ = stockRepo.UpdateMove(ctx, &picking.Moves[0])

	// 1. UpdateDeliveredQuantities
	if err := uc.UpdateDeliveredQuantities(ctx, pickingID); err != nil {
		t.Fatalf("UpdateDeliveredQuantities failed: %v", err)
	}

	updatedOrder, _ := saleRepo.GetOrderByID(ctx, order.ID)
	if updatedOrder.Lines[0].QtyDelivered != 6 {
		t.Errorf("expected qty_delivered=6, got %.2f", updatedOrder.Lines[0].QtyDelivered)
	}
	if updatedOrder.DeliveryStatus != string(sale.DeliveryStatusStarted) {
		t.Errorf("expected delivery status 'started', got %s", updatedOrder.DeliveryStatus)
	}

	// 2. CreateBackorder (remaining 4)
	backorder, err := uc.CreateBackorder(ctx, pickingID)
	if err != nil {
		t.Fatalf("CreateBackorder failed: %v", err)
	}
	if backorder == nil {
		t.Fatal("expected backorder picking, got nil")
	}
	if len(backorder.Moves) != 1 || backorder.Moves[0].ProductQty != 4 {
		t.Fatalf("expected backorder move with qty 4, got %+v", backorder.Moves)
	}

	// Order should now have 2 pickings
	updatedOrder, _ = saleRepo.GetOrderByID(ctx, order.ID)
	if len(updatedOrder.PickingIDs) != 2 {
		t.Errorf("expected order to track 2 pickings, got %d", len(updatedOrder.PickingIDs))
	}

	// 3. CancelDeliveries
	if err := uc.CancelDeliveries(ctx, order.ID); err != nil {
		t.Fatalf("CancelDeliveries failed: %v", err)
	}
	boPicking, _ := stockRepo.GetPickingByID(ctx, backorder.ID)
	if boPicking.State != stock.PickingStateCancel {
		t.Errorf("expected backorder to be cancelled, got %s", boPicking.State)
	}
}

