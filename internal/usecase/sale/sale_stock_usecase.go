package saleusecase

import (
	"context"
	"fmt"
	"log/slog"

	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// SaleStockUseCase handles the integration between Sales and Inventory.
type SaleStockUseCase struct {
	repo      sale.Repository
	stockRepo stock.Repository
	stockUC   *stockusecase.UseCase
	logger    *slog.Logger
}

// NewSaleStockUseCase creates a new SaleStockUseCase.
func NewSaleStockUseCase(
	repo sale.Repository,
	stockRepo stock.Repository,
	stockUC *stockusecase.UseCase,
	logger *slog.Logger,
) *SaleStockUseCase {
	return &SaleStockUseCase{
		repo:      repo,
		stockRepo: stockRepo,
		stockUC:   stockUC,
		logger:    logger,
	}
}

// CreateDeliveriesFromOrder confirmed Sale Order confirmation: creates a ProcurementGroup,
// generates an outgoing StockPicking (delivery), creates StockMoves for each line,
// and links them back to the Sale Order.
func (uc *SaleStockUseCase) CreateDeliveriesFromOrder(ctx context.Context, order *sale.SaleOrder) error {
	if order.State != sale.OrderStateSale && order.State != sale.OrderStateDone {
		return platformerrors.Conflict(fmt.Sprintf("cannot create deliveries for order in state '%s'; must be confirmed", order.State))
	}

	// 1. Create Procurement Group
	pg := &stock.ProcurementGroup{
		Name:      order.Name,
		CompanyID: 0, // Default or from order
	}
	if order.CompanyID != nil {
		pg.CompanyID = *order.CompanyID
	}
	if err := uc.stockRepo.CreateProcurementGroup(ctx, pg); err != nil {
		return err
	}
	order.ProcurementGroupID = &pg.ID

	// 2. Prepare StockPicking (Delivery Order)
	// We use the stock usecase to benefit from its location resolution logic.
	movesInput := make([]stockusecase.CreateMoveInput, 0, len(order.Lines))
	for _, l := range order.Lines {
		// Skip service products or lines with zero qty (already validated in order confirm)
		lID := l.ID
		movesInput = append(movesInput, stockusecase.CreateMoveInput{
			ProductID:  l.ProductID,
			ProductQty: l.ProductUomQty,
			Name:       string(l.Name),
			ProductUom: l.ProductUom,
			SaleLineID: &lID,
		})
	}

	if len(movesInput) == 0 {
		return nil
	}

	pickingInput := stockusecase.CreatePickingInput{
		PickingType:    stock.PickingTypeOutgoing,
		PartnerID:      &order.PartnerID,
		ScheduledDate:  order.DateOrder,
		Origin:         order.Name,
		SourceOrderID:  &order.ID,
		CarrierID:      order.CarrierID,
		ShippingWeight: order.ShippingWeight,
		CompanyID:      order.CompanyID,
		Note:           order.Note,
		Moves:          movesInput,
	}

	picking, err := uc.stockUC.CreatePicking(ctx, pickingInput)
	if err != nil {
		return err
	}

	// Link procurement group to picking
	picking.ProcurementGroupID = &pg.ID
	if err := uc.stockRepo.UpdatePicking(ctx, picking); err != nil {
		return err
	}

	// 3. Update Order with Picking ID and Procurement Group
	order.PickingIDs = append(order.PickingIDs, picking.ID)
	order.DeliveryStatus = "nothing" // Initial status

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return err
	}

	uc.logger.InfoContext(ctx, "deliveries created for sale order",
		"order_id", order.ID,
		"picking_id", picking.ID,
		"procurement_group_id", pg.ID,
	)

	return nil
}

// OnPickingValidated implements stockusecase.StockIntegrationHook.
func (uc *SaleStockUseCase) OnPickingValidated(ctx context.Context, picking *stock.StockPicking) error {
	if picking.PickingType != stock.PickingTypeOutgoing || picking.SourceOrderID == nil {
		return nil
	}
	return uc.UpdateDeliveredQuantities(ctx, picking.ID)
}

// OnPickingCancelled implements stockusecase.StockIntegrationHook.
func (uc *SaleStockUseCase) OnPickingCancelled(ctx context.Context, picking *stock.StockPicking) error {
	return nil
}

// UpdateDeliveredQuantities updates the QtyDelivered on the linked SaleOrder lines
// after a delivery picking has been validated (ActionValidate).
func (uc *SaleStockUseCase) UpdateDeliveredQuantities(ctx context.Context, pickingID int64) error {
	picking, err := uc.stockRepo.GetPickingByID(ctx, pickingID)
	if err != nil {
		return err
	}
	if picking == nil || picking.SourceOrderID == nil {
		return nil
	}

	order, err := uc.repo.GetOrderByID(ctx, *picking.SourceOrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	modified := false
	for _, m := range picking.Moves {
		for i := range order.Lines {
			line := &order.Lines[i]
			isMatch := false
			if m.SaleLineID != nil && *m.SaleLineID == line.ID {
				isMatch = true
			} else if m.SaleLineID == nil && line.ProductID == m.ProductID {
				isMatch = true
			}

			if isMatch {
				qty := m.QuantityDone
				if qty == 0 && m.State == stock.MoveStateDone {
					qty = m.ProductQty
				}
				line.QtyDelivered += qty
				modified = true
				break
			}
		}
	}

	if modified {
		order.UpdateDeliveryStatus()
		order.UpdateInvoiceStatus()
		if err := uc.repo.UpdateOrder(ctx, order); err != nil {
			uc.logger.ErrorContext(ctx, "failed to update sale order delivered quantities", "order_id", order.ID, "error", err)
			return err
		}
		uc.logger.InfoContext(ctx, "sale order delivered quantities updated",
			"order_id", order.ID,
			"delivery_status", order.DeliveryStatus,
		)
	}

	return nil
}

// CreateBackorder generates a new delivery picking for any unfulfilled quantities
// if a picking was validated with QuantityDone < ProductQty.
func (uc *SaleStockUseCase) CreateBackorder(ctx context.Context, pickingID int64) (*stock.StockPicking, error) {
	picking, err := uc.stockRepo.GetPickingByID(ctx, pickingID)
	if err != nil {
		return nil, err
	}
	if picking == nil {
		return nil, platformerrors.NotFound("picking not found")
	}

	var backorderMoves []stockusecase.CreateMoveInput
	for _, m := range picking.Moves {
		remaining := m.ProductQty - m.QuantityDone
		if remaining > 0 {
			backorderMoves = append(backorderMoves, stockusecase.CreateMoveInput{
				ProductID:      m.ProductID,
				ProductQty:     remaining,
				Name:           fmt.Sprintf("Backorder of %s", m.Name),
				ProductUom:     m.ProductUom,
				SaleLineID:     m.SaleLineID,
				PurchaseLineID: m.PurchaseLineID,
			})
		}
	}

	if len(backorderMoves) == 0 {
		return nil, nil // All moves fulfilled, no backorder required
	}

	backorderInput := stockusecase.CreatePickingInput{
		PickingType:    picking.PickingType,
		PartnerID:      picking.PartnerID,
		ScheduledDate:  picking.ScheduledDate,
		Origin:         fmt.Sprintf("Backorder of %s", picking.Name),
		SourceOrderID:  picking.SourceOrderID,
		CarrierID:      picking.CarrierID,
		ShippingWeight: picking.ShippingWeight,
		CompanyID:      picking.CompanyID,
		Note:           picking.Note,
		Moves:          backorderMoves,
	}

	backorder, err := uc.stockUC.CreatePicking(ctx, backorderInput)
	if err != nil {
		return nil, err
	}

	backorder.BackorderOfID = &picking.ID
	backorder.ProcurementGroupID = picking.ProcurementGroupID
	if err := uc.stockRepo.UpdatePicking(ctx, backorder); err != nil {
		return nil, err
	}

	if picking.SourceOrderID != nil {
		if order, err := uc.repo.GetOrderByID(ctx, *picking.SourceOrderID); err == nil && order != nil {
			order.PickingIDs = append(order.PickingIDs, backorder.ID)
			_ = uc.repo.UpdateOrder(ctx, order)
		}
	}

	uc.logger.InfoContext(ctx, "backorder picking created",
		"original_picking_id", picking.ID,
		"backorder_picking_id", backorder.ID,
		"remaining_moves", len(backorderMoves),
	)

	return backorder, nil
}

// CancelDeliveries cancels all open/uncompleted delivery pickings linked to a sale order.
func (uc *SaleStockUseCase) CancelDeliveries(ctx context.Context, orderID int64) error {
	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return platformerrors.NotFound("sale order not found")
	}

	for _, pickingID := range order.PickingIDs {
		p, err := uc.stockRepo.GetPickingByID(ctx, pickingID)
		if err != nil || p == nil {
			continue
		}
		// Cancel only pickings that are not already done or cancelled
		if p.State != stock.PickingStateDone && p.State != stock.PickingStateCancel {
			if _, err := uc.stockUC.CancelPicking(ctx, p.ID); err != nil {
				uc.logger.WarnContext(ctx, "failed to cancel delivery for order", "order_id", order.ID, "picking_id", p.ID, "error", err)
			}
		}
	}

	uc.logger.InfoContext(ctx, "unfulfilled deliveries cancelled for sale order", "order_id", order.ID)
	return nil
}
