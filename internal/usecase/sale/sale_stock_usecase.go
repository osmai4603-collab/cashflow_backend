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
			Name:       l.Name,
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
