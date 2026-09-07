package purchaseusecase

import (
	"context"
	"fmt"
	"log/slog"

	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// PurchaseStockUseCase handles the integration between Purchase and Inventory.
type PurchaseStockUseCase struct {
	repo      purchase.Repository
	stockRepo stock.Repository
	stockUC   *stockusecase.UseCase
	logger    *slog.Logger
}

// NewPurchaseStockUseCase creates a new PurchaseStockUseCase.
func NewPurchaseStockUseCase(
	repo purchase.Repository,
	stockRepo stock.Repository,
	stockUC *stockusecase.UseCase,
	logger *slog.Logger,
) *PurchaseStockUseCase {
	return &PurchaseStockUseCase{
		repo:      repo,
		stockRepo: stockRepo,
		stockUC:   stockUC,
		logger:    logger,
	}
}

// CreateReceiptsFromOrder handles Purchase Order confirmation: creates a ProcurementGroup,
// generates an incoming StockPicking (receipt), creates StockMoves for each line,
// and links them back to the Purchase Order.
func (uc *PurchaseStockUseCase) CreateReceiptsFromOrder(ctx context.Context, order *purchase.PurchaseOrder) error {
	if order.State != purchase.OrderStatePurchase && order.State != purchase.OrderStateDone {
		return platformerrors.Conflict(fmt.Sprintf("cannot create receipts for purchase order in state '%s'; must be confirmed", order.State))
	}

	// 1. Create Procurement Group
	pg := &stock.ProcurementGroup{
		Name:      order.Name,
		CompanyID: 0,
	}
	if order.CompanyID != nil {
		pg.CompanyID = *order.CompanyID
	}
	if err := uc.stockRepo.CreateProcurementGroup(ctx, pg); err != nil {
		return err
	}
	order.ProcurementGroupID = &pg.ID

	// 2. Prepare StockPicking (Incoming Receipt)
	movesInput := make([]stockusecase.CreateMoveInput, 0, len(order.Lines))
	for _, l := range order.Lines {
		lID := l.ID
		movesInput = append(movesInput, stockusecase.CreateMoveInput{
			ProductID:      l.ProductID,
			ProductQty:     l.ProductQty,
			Name:           l.Name,
			ProductUom:     l.ProductUom,
			PurchaseLineID: &lID,
		})
	}

	if len(movesInput) == 0 {
		return nil
	}

	pickingInput := stockusecase.CreatePickingInput{
		PickingType:    stock.PickingTypeIncoming,
		PartnerID:      &order.PartnerID,
		ScheduledDate:  order.DateOrder,
		Origin:         order.Name,
		SourceOrderID:  &order.ID,
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
	order.ReceiptStatus = "nothing"

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return err
	}

	uc.logger.InfoContext(ctx, "receipts created for purchase order",
		"order_id", order.ID,
		"picking_id", picking.ID,
		"procurement_group_id", pg.ID,
	)

	return nil
}
