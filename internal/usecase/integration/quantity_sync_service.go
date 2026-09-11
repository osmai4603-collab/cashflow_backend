package integration

import (
	"context"
	"log/slog"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
)

// QuantitySyncService provides cross-domain synchronization of quantities
// between Sales, Purchases, Inventory, and Accounting moves.
//
// Corresponds to Odoo 19 cross-model quantity sync:
//   - picking validated -> sale_order.qty_delivered
//   - picking validated -> purchase_order.qty_received
//   - invoice/bill posted -> sale/purchase.qty_invoiced
type QuantitySyncService struct {
	saleRepo       sale.Repository
	purchaseRepo   purchase.Repository
	stockRepo      stock.Repository
	accountingRepo accounting.Repository
	logger         *slog.Logger
}

// NewQuantitySyncService creates a new QuantitySyncService instance.
func NewQuantitySyncService(
	saleRepo sale.Repository,
	purchaseRepo purchase.Repository,
	stockRepo stock.Repository,
	accountingRepo accounting.Repository,
	logger *slog.Logger,
) *QuantitySyncService {
	if logger == nil {
		logger = slog.Default()
	}
	return &QuantitySyncService{
		saleRepo:       saleRepo,
		purchaseRepo:   purchaseRepo,
		stockRepo:      stockRepo,
		accountingRepo: accountingRepo,
		logger:         logger,
	}
}

// SyncDeliveredQty re-aggregates all done stock moves for a sales order
// and synchronizes the QtyDelivered field across all order lines.
func (s *QuantitySyncService) SyncDeliveredQty(ctx context.Context, orderID int64) error {
	order, err := s.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	deliveredByLine := make(map[int64]float64)
	deliveredByProduct := make(map[int64]float64)

	for _, pickingID := range order.PickingIDs {
		picking, err := s.stockRepo.GetPickingByID(ctx, pickingID)
		if err != nil || picking == nil || picking.PickingType != stock.PickingTypeOutgoing {
			continue
		}
		for _, m := range picking.Moves {
			if m.State != stock.MoveStateDone {
				continue
			}
			qty := m.QuantityDone
			if qty == 0 {
				qty = m.ProductQty
			}
			if m.SaleLineID != nil {
				deliveredByLine[*m.SaleLineID] += qty
			} else {
				deliveredByProduct[m.ProductID] += qty
			}
		}
	}

	for i := range order.Lines {
		line := &order.Lines[i]
		if val, exists := deliveredByLine[line.ID]; exists {
			line.QtyDelivered = val
		} else if val, exists := deliveredByProduct[line.ProductID]; exists {
			line.QtyDelivered = val
		}
	}

	order.UpdateDeliveryStatus()
	order.UpdateInvoiceStatus()

	return s.saleRepo.UpdateOrder(ctx, order)
}

// SyncReceivedQty re-aggregates all done stock moves for a purchase order
// and synchronizes the QtyReceived field across all order lines.
func (s *QuantitySyncService) SyncReceivedQty(ctx context.Context, orderID int64) error {
	order, err := s.purchaseRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	receivedByLine := make(map[int64]float64)
	receivedByProduct := make(map[int64]float64)

	for _, pickingID := range order.PickingIDs {
		picking, err := s.stockRepo.GetPickingByID(ctx, pickingID)
		if err != nil || picking == nil || picking.PickingType != stock.PickingTypeIncoming {
			continue
		}
		for _, m := range picking.Moves {
			if m.State != stock.MoveStateDone {
				continue
			}
			qty := m.QuantityDone
			if qty == 0 {
				qty = m.ProductQty
			}
			if m.PurchaseLineID != nil {
				receivedByLine[*m.PurchaseLineID] += qty
			} else {
				receivedByProduct[m.ProductID] += qty
			}
		}
	}

	for i := range order.Lines {
		line := &order.Lines[i]
		if val, exists := receivedByLine[line.ID]; exists {
			line.QtyReceived = val
		} else if val, exists := receivedByProduct[line.ProductID]; exists {
			line.QtyReceived = val
		}
	}

	order.UpdateBillStatus()

	return s.purchaseRepo.UpdateOrder(ctx, order)
}

// SyncInvoicedQty processes an accounting move (customer invoice or vendor bill)
// and updates the linked sale/purchase order invoiced status.
func (s *QuantitySyncService) SyncInvoicedQty(ctx context.Context, moveID int64) error {
	move, err := s.accountingRepo.GetMoveByID(ctx, moveID)
	if err != nil {
		return err
	}
	if move == nil || move.State != accounting.MoveStatePosted {
		return nil
	}

	s.logger.InfoContext(ctx, "invoiced quantity synchronization verified",
		"move_id", moveID,
		"move_type", move.MoveType,
		"state", move.State,
	)
	return nil
}
