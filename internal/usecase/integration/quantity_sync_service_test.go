package integration_test

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/usecase/integration"
)

type mockSaleRepo struct {
	order *sale.SaleOrder
	sale.Repository
}

func (m *mockSaleRepo) GetOrderByID(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	return m.order, nil
}

func (m *mockSaleRepo) UpdateOrder(ctx context.Context, order *sale.SaleOrder) error {
	m.order = order
	return nil
}

type mockPurchaseRepo struct {
	order *purchase.PurchaseOrder
	purchase.Repository
}

func (m *mockPurchaseRepo) GetOrderByID(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	return m.order, nil
}

func (m *mockPurchaseRepo) UpdateOrder(ctx context.Context, order *purchase.PurchaseOrder) error {
	m.order = order
	return nil
}

type mockStockRepo struct {
	pickings map[int64]*stock.StockPicking
	stock.Repository
}

func (m *mockStockRepo) GetPickingByID(ctx context.Context, id int64) (*stock.StockPicking, error) {
	if p, ok := m.pickings[id]; ok {
		return p, nil
	}
	return nil, nil
}

type mockAccountingRepo struct {
	moves map[int64]*accounting.AccountMove
	accounting.Repository
}

func (m *mockAccountingRepo) GetMoveByID(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	if mv, ok := m.moves[id]; ok {
		return mv, nil
	}
	return nil, nil
}

func TestQuantitySyncService_SyncDeliveredQty(t *testing.T) {
	ctx := context.Background()

	lineID := int64(10)
	saleOrder := &sale.SaleOrder{
		ID:         1,
		PickingIDs: []int64{101, 102},
		Lines: []sale.SaleOrderLine{
			{
				ID:            lineID,
				ProductID:     50,
				ProductUomQty: 10,
				QtyDelivered:  0,
			},
		},
	}

	pickings := map[int64]*stock.StockPicking{
		101: {
			ID:          101,
			PickingType: stock.PickingTypeOutgoing,
			Moves: []stock.StockMove{
				{
					ProductID:    50,
					SaleLineID:   &lineID,
					ProductQty:   6,
					QuantityDone: 6,
					State:        stock.MoveStateDone,
				},
			},
		},
		102: {
			ID:          102,
			PickingType: stock.PickingTypeOutgoing,
			Moves: []stock.StockMove{
				{
					ProductID:    50,
					SaleLineID:   &lineID,
					ProductQty:   4,
					QuantityDone: 4,
					State:        stock.MoveStateDone,
				},
			},
		},
	}

	saleRepo := &mockSaleRepo{order: saleOrder}
	stockRepo := &mockStockRepo{pickings: pickings}
	svc := integration.NewQuantitySyncService(saleRepo, nil, stockRepo, nil, nil)

	if err := svc.SyncDeliveredQty(ctx, 1); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}

	if saleOrder.Lines[0].QtyDelivered != 10 {
		t.Errorf("expected delivered qty 10, got %.2f", saleOrder.Lines[0].QtyDelivered)
	}
}

func TestQuantitySyncService_SyncReceivedQty(t *testing.T) {
	ctx := context.Background()

	lineID := int64(20)
	purchaseOrder := &purchase.PurchaseOrder{
		ID:         2,
		PickingIDs: []int64{201},
		Lines: []purchase.PurchaseOrderLine{
			{
				ID:          lineID,
				ProductID:   70,
				ProductQty:  15,
				QtyReceived: 0,
			},
		},
	}

	pickings := map[int64]*stock.StockPicking{
		201: {
			ID:          201,
			PickingType: stock.PickingTypeIncoming,
			Moves: []stock.StockMove{
				{
					ProductID:      70,
					PurchaseLineID: &lineID,
					ProductQty:     15,
					QuantityDone:   12,
					State:          stock.MoveStateDone,
				},
			},
		},
	}

	purchaseRepo := &mockPurchaseRepo{order: purchaseOrder}
	stockRepo := &mockStockRepo{pickings: pickings}
	svc := integration.NewQuantitySyncService(nil, purchaseRepo, stockRepo, nil, nil)

	if err := svc.SyncReceivedQty(ctx, 2); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}

	if purchaseOrder.Lines[0].QtyReceived != 12 {
		t.Errorf("expected received qty 12, got %.2f", purchaseOrder.Lines[0].QtyReceived)
	}
}

func TestQuantitySyncService_SyncInvoicedQty(t *testing.T) {
	ctx := context.Background()

	moves := map[int64]*accounting.AccountMove{
		1: {
			ID:       1,
			MoveType: accounting.MoveTypeOutInvoice,
			State:    accounting.MoveStatePosted,
			Date:     time.Now(),
		},
	}

	acctRepo := &mockAccountingRepo{moves: moves}
	svc := integration.NewQuantitySyncService(nil, nil, nil, acctRepo, nil)

	if err := svc.SyncInvoicedQty(ctx, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
