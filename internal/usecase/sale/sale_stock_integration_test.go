package saleusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/stock"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	saleusecase "cashflow_backend/internal/usecase/sale"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

func setupIntegrationTestEnvironment(t *testing.T) (*saleusecase.UseCase, *stockusecase.UseCase, *stockstorage.MemoryRepo, int64, int64) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	// 1. Repositories
	saleRepo := salestorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()
	accountingRepo := accountingstorage.NewMemoryRepo()
	stockRepo := stockstorage.NewMemoryRepo()

	// 2. UseCases
	accountingUC := accountingusecase.New(accountingRepo, logger)
	stockUC := stockusecase.New(stockRepo, partnerRepo, productRepo, saleRepo, nil, logger)

	// Create Sale UseCase with Stock integration
	saleUC := saleusecase.New(saleRepo, partnerRepo, productRepo, accountingRepo, accountingUC, logger, stockRepo, stockUC)

	// 3. Seed Partner (Customer)
	customer := &partner.Partner{
		Name:       "Test Customer",
		IsCustomer: true,
		Active:     true,
	}
	if err := partnerRepo.Create(ctx, customer); err != nil {
		t.Fatalf("failed to seed customer: %v", err)
	}

	// 4. Seed Product (Goods)
	prod := &product.ProductTemplate{
		Name:      "Physical Product",
		Type:      product.ProductTypeGoods,
		SalePrice: 100.0,
		CostPrice: 60.0,
		SaleOK:    true,
		Active:    true,
	}
	if err := productRepo.CreateTemplate(ctx, prod); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	return saleUC, stockUC, stockRepo, customer.ID, prod.ID
}

func TestSaleStockIntegration_ConfirmOrderCreatesDelivery(t *testing.T) {
	ctx := context.Background()
	saleUC, stockUC, stockRepo, customerID, productID := setupIntegrationTestEnvironment(t)

	// 1. Create Sale Order
	order, err := saleUC.CreateOrder(ctx, saleusecase.CreateSaleOrderInput{
		PartnerID: customerID,
		DateOrder: time.Now(),
		Lines: []saleusecase.CreateSaleOrderLineInput{
			{
				ProductID:     productID,
				ProductUomQty: 5.0,
				Name:          "Line 1",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// 2. Confirm Order
	order, err = saleUC.ConfirmOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to confirm order: %v", err)
	}

	// 3. Verify Integration
	if order.ProcurementGroupID == nil {
		t.Fatal("expected procurement group ID to be set on order")
	}

	if len(order.PickingIDs) == 0 {
		t.Fatal("expected at least one picking ID to be linked to the order")
	}

	pickingID := order.PickingIDs[0]
	picking, err := stockUC.GetPicking(ctx, pickingID)
	if err != nil {
		t.Fatalf("failed to fetch picking: %v", err)
	}

	if picking.PickingType != stock.PickingTypeOutgoing {
		t.Errorf("expected outgoing picking, got %s", picking.PickingType)
	}

	if picking.Origin != order.Name {
		t.Errorf("expected picking origin %s, got %s", order.Name, picking.Origin)
	}

	if len(picking.Moves) != 1 {
		t.Fatalf("expected 1 stock move, got %d", len(picking.Moves))
	}

	move := picking.Moves[0]
	if move.ProductID != productID {
		t.Errorf("expected move product %d, got %d", productID, move.ProductID)
	}

	if move.ProductQty != 5.0 {
		t.Errorf("expected move qty 5.0, got %f", move.ProductQty)
	}

	if move.SaleLineID == nil || *move.SaleLineID != order.Lines[0].ID {
		t.Errorf("expected move to be linked to sale line %d", order.Lines[0].ID)
	}

	// 4. Verify Procurement Group
	pg, err := stockRepo.GetProcurementGroupByID(ctx, *order.ProcurementGroupID)
	if err != nil {
		t.Fatalf("failed to fetch procurement group: %v", err)
	}
	if pg.Name != order.Name {
		t.Errorf("expected pg name %s, got %s", order.Name, pg.Name)
	}
}
