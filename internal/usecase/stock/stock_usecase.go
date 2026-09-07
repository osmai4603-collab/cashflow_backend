package stockusecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateLocationInput struct {
	Name           string              `json:"name"`
	Usage          stock.LocationUsage `json:"usage"`
	ParentID       *int64              `json:"parent_id"`
	ScrapLocation  bool                `json:"scrap_location"`
	ReturnLocation bool                `json:"return_location"`
	CompanyID      *int64              `json:"company_id"`
}

type UpdateLocationInput struct {
	Name           *string              `json:"name"`
	Usage          *stock.LocationUsage `json:"usage"`
	ParentID       *int64               `json:"parent_id"`
	ScrapLocation  *bool                `json:"scrap_location"`
	ReturnLocation *bool                `json:"return_location"`
	CompanyID      *int64               `json:"company_id"`
}

type CreateWarehouseInput struct {
	Name           string `json:"name"`
	Code           string `json:"code"`
	LotStockID     int64  `json:"lot_stock_id"`
	ViewLocationID *int64 `json:"view_location_id"`
	PartnerID      *int64 `json:"partner_id"`
	CompanyID      *int64 `json:"company_id"`
}

type UpdateWarehouseInput struct {
	Name           *string `json:"name"`
	Code           *string `json:"code"`
	LotStockID     *int64  `json:"lot_stock_id"`
	ViewLocationID *int64  `json:"view_location_id"`
	PartnerID      *int64  `json:"partner_id"`
	CompanyID      *int64  `json:"company_id"`
}

type CreateMoveInput struct {
	ProductID      int64   `json:"product_id"`
	ProductQty     float64 `json:"product_qty"`
	LocationID     *int64  `json:"location_id"`      // If omitted, defaults to picking's LocationID
	LocationDestID *int64  `json:"location_dest_id"` // If omitted, defaults to picking's LocationDestID
	Name           string  `json:"name"`
	ProductUom     *int64  `json:"product_uom"`
	Sequence       int     `json:"sequence"`
	SaleLineID     *int64  `json:"sale_line_id"`
	PurchaseLineID *int64  `json:"purchase_line_id"`
}

type CreatePickingInput struct {
	PickingType    stock.PickingType `json:"picking_type"`
	PartnerID      *int64            `json:"partner_id"`
	LocationID     *int64            `json:"location_id"`      // Default resolved by picking type if omitted
	LocationDestID *int64            `json:"location_dest_id"` // Default resolved by picking type if omitted
	ScheduledDate  time.Time         `json:"scheduled_date"`
	Origin         string            `json:"origin"`
	SourceOrderID  *int64            `json:"source_order_id"`
	CompanyID      *int64            `json:"company_id"`
	Note           string            `json:"note"`
	Moves          []CreateMoveInput `json:"moves"`
}

type UpdatePickingInput struct {
	PartnerID      *int64            `json:"partner_id"`
	LocationID     *int64            `json:"location_id"`
	LocationDestID *int64            `json:"location_dest_id"`
	ScheduledDate  *time.Time        `json:"scheduled_date"`
	Origin         *string           `json:"origin"`
	Note           *string           `json:"note"`
	Moves          []CreateMoveInput `json:"moves"`
}

type ValidatePickingInput struct {
	EffectiveDate *time.Time `json:"effective_date"`
}

type StockAdjustmentInput struct {
	ProductID   int64   `json:"product_id"`
	LocationID  int64   `json:"location_id"`
	NewQuantity float64 `json:"new_quantity"`
	Note        string  `json:"note"`
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Implementation
// ─────────────────────────────────────────────────────────────────────────────

type UseCase struct {
	repo          stock.Repository
	partnerRepo   partner.Repository
	productRepo   product.Repository
	saleRepo      sale.Repository
	purchaseRepo  purchase.Repository
	accountingSvc AccountingGateway
	logger        *slog.Logger
}

func New(
	repo stock.Repository,
	partnerRepo partner.Repository,
	productRepo product.Repository,
	saleRepo sale.Repository,
	purchaseRepo purchase.Repository,
	optional ...any,
) *UseCase {
	var accountingSvc AccountingGateway
	var logger *slog.Logger
	for _, value := range optional {
		switch typed := value.(type) {
		case AccountingGateway:
			accountingSvc = typed
		case *slog.Logger:
			logger = typed
		}
	}
	if logger == nil {
		logger = slog.Default()
	}
	if accountingSvc == nil {
		accountingSvc = noopAccountingGateway{}
	}
	return &UseCase{
		repo:          repo,
		partnerRepo:   partnerRepo,
		productRepo:   productRepo,
		saleRepo:      saleRepo,
		purchaseRepo:  purchaseRepo,
		accountingSvc: accountingSvc,
		logger:        logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Locations
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateLocation(ctx context.Context, in CreateLocationInput) (*stock.StockLocation, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, platformerrors.Validation("location name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	loc := &stock.StockLocation{
		Name:           name,
		Usage:          in.Usage,
		ParentID:       in.ParentID,
		ScrapLocation:  in.ScrapLocation,
		ReturnLocation: in.ReturnLocation,
		CompanyID:      in.CompanyID,
		Active:         true,
	}

	if in.ParentID != nil && *in.ParentID > 0 {
		parent, err := uc.repo.GetLocationByID(ctx, *in.ParentID)
		if err != nil {
			return nil, platformerrors.Validation("parent location does not exist", map[string]string{
				"parent_id": fmt.Sprintf("location #%d not found", *in.ParentID),
			})
		}
		loc.ComputeCompleteName(parent.CompleteName)
	} else {
		loc.ComputeCompleteName("")
	}

	if err := loc.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateLocation(ctx, loc); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock location created", "id", loc.ID, "complete_name", loc.CompleteName)
	return loc, nil
}

func (uc *UseCase) GetLocation(ctx context.Context, id int64) (*stock.StockLocation, error) {
	return uc.repo.GetLocationByID(ctx, id)
}

func (uc *UseCase) ListLocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockLocation], error) {
	return uc.repo.ListLocations(ctx, f, page)
}

func (uc *UseCase) UpdateLocation(ctx context.Context, id int64, in UpdateLocationInput) (*stock.StockLocation, error) {
	loc, err := uc.repo.GetLocationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		loc.Name = strings.TrimSpace(*in.Name)
	}
	if in.Usage != nil {
		loc.Usage = *in.Usage
	}
	if in.ScrapLocation != nil {
		loc.ScrapLocation = *in.ScrapLocation
	}
	if in.ReturnLocation != nil {
		loc.ReturnLocation = *in.ReturnLocation
	}
	if in.CompanyID != nil {
		loc.CompanyID = in.CompanyID
	}
	if in.ParentID != nil {
		if *in.ParentID == loc.ID {
			return nil, platformerrors.Conflict("location cannot be its own parent")
		}
		parent, err := uc.repo.GetLocationByID(ctx, *in.ParentID)
		if err != nil {
			return nil, err
		}
		loc.ParentID = in.ParentID
		loc.ComputeCompleteName(parent.CompleteName)
	} else if in.Name != nil {
		// Recompute name if parent was unchanged
		if loc.ParentID != nil {
			if parent, err := uc.repo.GetLocationByID(ctx, *loc.ParentID); err == nil {
				loc.ComputeCompleteName(parent.CompleteName)
			}
		} else {
			loc.ComputeCompleteName("")
		}
	}

	if err := loc.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateLocation(ctx, loc); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock location updated", "id", loc.ID, "name", loc.Name)
	return loc, nil
}

func (uc *UseCase) DeleteLocation(ctx context.Context, id int64) error {
	return uc.repo.DeleteLocation(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Warehouses
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateWarehouse(ctx context.Context, in CreateWarehouseInput) (*stock.Warehouse, error) {
	name := strings.TrimSpace(in.Name)
	code := strings.ToUpper(strings.TrimSpace(in.Code))

	if name == "" || code == "" {
		return nil, platformerrors.Validation("warehouse name and code are required", nil)
	}

	if in.LotStockID <= 0 {
		return nil, platformerrors.Validation("default stock location is required", map[string]string{
			"lot_stock_id": "must reference a valid internal location",
		})
	}

	// Verify lot stock exists
	lotLoc, err := uc.repo.GetLocationByID(ctx, in.LotStockID)
	if err != nil {
		return nil, platformerrors.Validation("lot stock location not found", map[string]string{
			"lot_stock_id": "location does not exist",
		})
	}
	if lotLoc.Usage != stock.LocationUsageInternal {
		return nil, platformerrors.Validation("lot stock location must be of internal usage", map[string]string{
			"lot_stock_id": fmt.Sprintf("location has usage '%s', expected 'internal'", lotLoc.Usage),
		})
	}

	wh := &stock.Warehouse{
		Name:           name,
		Code:           code,
		LotStockID:     in.LotStockID,
		ViewLocationID: in.ViewLocationID,
		PartnerID:      in.PartnerID,
		CompanyID:      in.CompanyID,
		Active:         true,
	}

	if err := wh.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateWarehouse(ctx, wh); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "warehouse created", "id", wh.ID, "code", wh.Code)
	return wh, nil
}

func (uc *UseCase) GetWarehouse(ctx context.Context, id int64) (*stock.Warehouse, error) {
	return uc.repo.GetWarehouseByID(ctx, id)
}

func (uc *UseCase) ListWarehouses(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.Warehouse], error) {
	return uc.repo.ListWarehouses(ctx, f, page)
}

func (uc *UseCase) UpdateWarehouse(ctx context.Context, id int64, in UpdateWarehouseInput) (*stock.Warehouse, error) {
	wh, err := uc.repo.GetWarehouseByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		wh.Name = strings.TrimSpace(*in.Name)
	}
	if in.Code != nil && strings.TrimSpace(*in.Code) != "" {
		wh.Code = strings.ToUpper(strings.TrimSpace(*in.Code))
	}
	if in.LotStockID != nil && *in.LotStockID > 0 {
		lotLoc, err := uc.repo.GetLocationByID(ctx, *in.LotStockID)
		if err != nil {
			return nil, err
		}
		if lotLoc.Usage != stock.LocationUsageInternal {
			return nil, platformerrors.Validation("lot stock location must be of internal usage", nil)
		}
		wh.LotStockID = *in.LotStockID
	}
	if in.ViewLocationID != nil {
		wh.ViewLocationID = in.ViewLocationID
	}
	if in.PartnerID != nil {
		wh.PartnerID = in.PartnerID
	}
	if in.CompanyID != nil {
		wh.CompanyID = in.CompanyID
	}

	if err := wh.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateWarehouse(ctx, wh); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "warehouse updated", "id", wh.ID, "code", wh.Code)
	return wh, nil
}

func (uc *UseCase) DeleteWarehouse(ctx context.Context, id int64) error {
	return uc.repo.DeleteWarehouse(ctx, id)
}

func (uc *UseCase) GetDefaultWarehouse(ctx context.Context) (*stock.Warehouse, error) {
	return uc.repo.GetDefaultWarehouse(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Pickings
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreatePicking(ctx context.Context, in CreatePickingInput) (*stock.StockPicking, error) {
	if in.PickingType == "" {
		in.PickingType = stock.PickingTypeOutgoing
	}

	// Resolve default source & destination locations if not specified
	var srcLocID, destLocID int64
	if in.LocationID != nil && *in.LocationID > 0 {
		srcLocID = *in.LocationID
	}
	if in.LocationDestID != nil && *in.LocationDestID > 0 {
		destLocID = *in.LocationDestID
	}

	// Fallback to standard locations based on PickingType
	if srcLocID <= 0 || destLocID <= 0 {
		switch in.PickingType {
		case stock.PickingTypeIncoming:
			if srcLocID <= 0 {
				if suppLoc, err := uc.repo.GetLocationByUsage(ctx, stock.LocationUsageSupplier); err == nil {
					srcLocID = suppLoc.ID
				} else {
					srcLocID = 2 // standard seed fallback
				}
			}
			if destLocID <= 0 {
				if wh, err := uc.repo.GetDefaultWarehouse(ctx); err == nil {
					destLocID = wh.LotStockID
				} else {
					destLocID = 8 // standard seed fallback
				}
			}
		case stock.PickingTypeOutgoing:
			if srcLocID <= 0 {
				if wh, err := uc.repo.GetDefaultWarehouse(ctx); err == nil {
					srcLocID = wh.LotStockID
				} else {
					srcLocID = 8 // standard seed fallback
				}
			}
			if destLocID <= 0 {
				if custLoc, err := uc.repo.GetLocationByUsage(ctx, stock.LocationUsageCustomer); err == nil {
					destLocID = custLoc.ID
				} else {
					destLocID = 3 // standard seed fallback
				}
			}
		case stock.PickingTypeInternal:
			if srcLocID <= 0 {
				srcLocID = 8
			}
			if destLocID <= 0 {
				destLocID = 9 // e.g. WH/Input or another internal
			}
		}
	}

	if in.PartnerID != nil && *in.PartnerID > 0 && uc.partnerRepo != nil {
		p, err := uc.partnerRepo.GetByID(ctx, *in.PartnerID)
		if err != nil {
			return nil, err
		}
		if !p.Active {
			return nil, platformerrors.Conflict("partner is inactive")
		}
	}

	if in.ScheduledDate.IsZero() {
		in.ScheduledDate = time.Now().UTC()
	}

	picking := &stock.StockPicking{
		Name:           "/",
		PickingType:    in.PickingType,
		State:          stock.PickingStateDraft,
		PartnerID:      in.PartnerID,
		LocationID:     srcLocID,
		LocationDestID: destLocID,
		ScheduledDate:  in.ScheduledDate,
		Origin:         in.Origin,
		SourceOrderID:  in.SourceOrderID,
		CompanyID:      in.CompanyID,
		Note:           in.Note,
		Active:         true,
	}

	// Prepare moves
	var moves []stock.StockMove
	for i, min := range in.Moves {
		if min.ProductID <= 0 {
			return nil, platformerrors.Validation(fmt.Sprintf("move #%d: product is required", i+1), nil)
		}
		if min.ProductQty <= 0 {
			return nil, platformerrors.Validation(fmt.Sprintf("move #%d: quantity must be greater than 0", i+1), nil)
		}

		// Validate product
		var prodName string
		var uomID *int64
		if uc.productRepo != nil {
			prod, err := uc.productRepo.GetTemplateByID(ctx, min.ProductID)
			if err != nil {
				return nil, err
			}
			if !prod.Active {
				return nil, platformerrors.Conflict(fmt.Sprintf("product '%s' is inactive", prod.Name))
			}
			if prod.Type == product.ProductTypeService {
				return nil, platformerrors.Conflict(fmt.Sprintf("product '%s' is a service and does not generate stock moves", prod.Name))
			}
			prodName = prod.Name
			uomID = prod.UoMID
		}

		moveName := strings.TrimSpace(min.Name)
		if moveName == "" {
			if prodName != "" {
				moveName = prodName
			} else {
				moveName = fmt.Sprintf("Item #%d move", min.ProductID)
			}
		}

		mSrc := srcLocID
		if min.LocationID != nil && *min.LocationID > 0 {
			mSrc = *min.LocationID
		}
		mDest := destLocID
		if min.LocationDestID != nil && *min.LocationDestID > 0 {
			mDest = *min.LocationDestID
		}

		moveUom := uomID
		if min.ProductUom != nil && *min.ProductUom > 0 {
			moveUom = min.ProductUom
		}

		seq := min.Sequence
		if seq == 0 {
			seq = (i + 1) * 10
		}

		move := stock.StockMove{
			Sequence:       seq,
			Name:           moveName,
			ProductID:      min.ProductID,
			ProductUom:     moveUom,
			ProductQty:     min.ProductQty,
			LocationID:     mSrc,
			LocationDestID: mDest,
			State:          stock.MoveStateDraft,
			SaleLineID:     min.SaleLineID,
			PurchaseLineID: min.PurchaseLineID,
			Date:           in.ScheduledDate,
		}

		if err := move.Validate(); err != nil {
			return nil, err
		}
		moves = append(moves, move)
	}

	picking.Moves = moves
	if err := picking.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreatePicking(ctx, picking); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock picking created", "id", picking.ID, "type", picking.PickingType, "moves_count", len(picking.Moves))
	return picking, nil
}

func (uc *UseCase) GetPicking(ctx context.Context, id int64) (*stock.StockPicking, error) {
	return uc.repo.GetPickingByID(ctx, id)
}

func (uc *UseCase) GetPickingByName(ctx context.Context, name string) (*stock.StockPicking, error) {
	return uc.repo.GetPickingByName(ctx, name)
}

func (uc *UseCase) ListPickings(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockPicking], error) {
	return uc.repo.ListPickings(ctx, f, page)
}

func (uc *UseCase) UpdatePicking(ctx context.Context, id int64, in UpdatePickingInput) (*stock.StockPicking, error) {
	picking, err := uc.repo.GetPickingByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if picking.State != stock.PickingStateDraft && picking.State != stock.PickingStateConfirmed {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot edit picking in state '%s'; must be draft or confirmed", picking.State))
	}

	if in.PartnerID != nil {
		picking.PartnerID = in.PartnerID
	}
	if in.LocationID != nil && *in.LocationID > 0 {
		picking.LocationID = *in.LocationID
	}
	if in.LocationDestID != nil && *in.LocationDestID > 0 {
		picking.LocationDestID = *in.LocationDestID
	}
	if in.ScheduledDate != nil && !in.ScheduledDate.IsZero() {
		picking.ScheduledDate = *in.ScheduledDate
	}
	if in.Origin != nil {
		picking.Origin = *in.Origin
	}
	if in.Note != nil {
		picking.Note = *in.Note
	}

	if in.Moves != nil {
		var moves []stock.StockMove
		for i, min := range in.Moves {
			mSrc := picking.LocationID
			if min.LocationID != nil && *min.LocationID > 0 {
				mSrc = *min.LocationID
			}
			mDest := picking.LocationDestID
			if min.LocationDestID != nil && *min.LocationDestID > 0 {
				mDest = *min.LocationDestID
			}

			seq := min.Sequence
			if seq == 0 {
				seq = (i + 1) * 10
			}

			m := stock.StockMove{
				Sequence:       seq,
				Name:           min.Name,
				ProductID:      min.ProductID,
				ProductUom:     min.ProductUom,
				ProductQty:     min.ProductQty,
				LocationID:     mSrc,
				LocationDestID: mDest,
				State:          stock.MoveState(picking.State),
				SaleLineID:     min.SaleLineID,
				PurchaseLineID: min.PurchaseLineID,
				Date:           picking.ScheduledDate,
			}
			if err := m.Validate(); err != nil {
				return nil, err
			}
			moves = append(moves, m)
		}
		picking.Moves = moves
	}

	if err := picking.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdatePicking(ctx, picking); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock picking updated", "id", picking.ID, "name", picking.Name)
	return picking, nil
}

func (uc *UseCase) DeletePicking(ctx context.Context, id int64) error {
	picking, err := uc.repo.GetPickingByID(ctx, id)
	if err != nil {
		return err
	}

	if picking.State != stock.PickingStateDraft && picking.State != stock.PickingStateCancel {
		return platformerrors.Conflict(fmt.Sprintf("cannot delete picking in state '%s'; must be draft or cancelled", picking.State))
	}

	return uc.repo.DeletePicking(ctx, id)
}

// ConfirmPicking assigns a formal sequence number and advances picking to confirmed.
func (uc *UseCase) ConfirmPicking(ctx context.Context, id int64) (*stock.StockPicking, error) {
	picking, err := uc.repo.GetPickingByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var seq string
	if picking.Name == "" || picking.Name == "/" {
		s, err := uc.repo.NextSequence(ctx, picking.PickingType, picking.ScheduledDate.Year())
		if err != nil {
			return nil, err
		}
		seq = s
	}

	if err := picking.ActionConfirm(seq); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdatePicking(ctx, picking); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock picking confirmed", "id", picking.ID, "name", picking.Name)
	return picking, nil
}

// ValidatePicking executes the picking transfer, updates quants, and updates linked sale/purchase order lines.
func (uc *UseCase) ValidatePicking(ctx context.Context, id int64, in ValidatePickingInput) (*stock.StockPicking, error) {
	picking, err := uc.repo.GetPickingByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if picking.State == stock.PickingStateDone {
		return nil, platformerrors.Conflict("stock picking has already been validated")
	}
	if picking.State == stock.PickingStateCancel {
		return nil, platformerrors.Conflict("cannot validate a cancelled stock picking")
	}

	// Auto-confirm if in draft
	if picking.State == stock.PickingStateDraft {
		var seq string
		if picking.Name == "" || picking.Name == "/" {
			s, err := uc.repo.NextSequence(ctx, picking.PickingType, picking.ScheduledDate.Year())
			if err != nil {
				return nil, err
			}
			seq = s
		}
		if err := picking.ActionConfirm(seq); err != nil {
			return nil, err
		}
	}

	// Check stock availability if source location is internal
	srcLoc, err := uc.repo.GetLocationByID(ctx, picking.LocationID)
	if err == nil && srcLoc.Usage == stock.LocationUsageInternal {
		for _, m := range picking.Moves {
			q, err := uc.repo.GetQuant(ctx, m.ProductID, m.LocationID)
			if err == nil {
				if q.Quantity < m.ProductQty {
					return nil, platformerrors.Conflict(fmt.Sprintf("insufficient stock for product #%d at location '%s': on hand %.2f, required %.2f",
						m.ProductID, srcLoc.CompleteName, q.Quantity, m.ProductQty))
				}
			}
		}
	}

	effectiveDate := time.Now().UTC()
	if in.EffectiveDate != nil && !in.EffectiveDate.IsZero() {
		effectiveDate = in.EffectiveDate.UTC()
	}

	if err := picking.ActionValidate(effectiveDate); err != nil {
		return nil, err
	}

	// Atomic persistence and quant movement
	if err := uc.repo.ValidatePickingTx(ctx, picking); err != nil {
		return nil, err
	}

	// Phase 12: value the moves and post stock valuation entries when boundaries are configured.
	if err := uc.ValuatePicking(ctx, picking); err != nil {
		uc.logger.ErrorContext(ctx, "stock valuation failed after picking validation", "id", picking.ID, "error", err)
		return nil, err
	}

	// Update linked Sale Order delivered quantities if applicable
	if picking.SourceOrderID != nil && picking.PickingType == stock.PickingTypeOutgoing && uc.saleRepo != nil {
		if so, err := uc.saleRepo.GetOrderByID(ctx, *picking.SourceOrderID); err == nil {
			modified := false
			for _, m := range picking.Moves {
				for i := range so.Lines {
					if (m.SaleLineID != nil && *m.SaleLineID == so.Lines[i].ID) || (m.SaleLineID == nil && so.Lines[i].ProductID == m.ProductID) {
						so.Lines[i].QtyDelivered += m.QuantityDone
						modified = true
						break
					}
				}
			}
			if modified {
				_ = uc.saleRepo.UpdateOrder(ctx, so)
			}
		}
	}

	// Update linked Purchase Order received quantities if applicable
	if picking.SourceOrderID != nil && picking.PickingType == stock.PickingTypeIncoming && uc.purchaseRepo != nil {
		if po, err := uc.purchaseRepo.GetOrderByID(ctx, *picking.SourceOrderID); err == nil {
			modified := false
			for _, m := range picking.Moves {
				for i := range po.Lines {
					if (m.PurchaseLineID != nil && *m.PurchaseLineID == po.Lines[i].ID) || (m.PurchaseLineID == nil && po.Lines[i].ProductID == m.ProductID) {
						po.Lines[i].QtyReceived += m.QuantityDone
						modified = true
						break
					}
				}
			}
			if modified {
				_ = uc.purchaseRepo.UpdateOrder(ctx, po)
			}
		}
	}

	uc.logger.InfoContext(ctx, "stock picking validated", "id", picking.ID, "name", picking.Name, "date_done", picking.DateDone)
	return picking, nil
}

// CancelPicking marks picking and its moves as cancelled.
func (uc *UseCase) CancelPicking(ctx context.Context, id int64) (*stock.StockPicking, error) {
	picking, err := uc.repo.GetPickingByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := picking.ActionCancel(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdatePicking(ctx, picking); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock picking cancelled", "id", picking.ID, "name", picking.Name)
	return picking, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Queries & Adjustments
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) GetOnHandStock(ctx context.Context, productID *int64, locationID *int64, warehouseID *int64) ([]stock.StockOnHandItem, error) {
	return uc.repo.GetOnHandStock(ctx, productID, locationID, warehouseID)
}

func (uc *UseCase) ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockMove], error) {
	return uc.repo.ListMoves(ctx, f, page)
}

// AdjustStock adjusts the physical count of a product at a location via an audit-compliant adjustment move.
func (uc *UseCase) AdjustStock(ctx context.Context, in StockAdjustmentInput) (*stock.StockQuant, error) {
	if in.ProductID <= 0 {
		return nil, platformerrors.Validation("product is required", nil)
	}
	if in.LocationID <= 0 {
		return nil, platformerrors.Validation("location is required", nil)
	}
	if in.NewQuantity < 0 {
		return nil, platformerrors.Validation("new quantity cannot be negative", nil)
	}

	// Verify target location
	loc, err := uc.repo.GetLocationByID(ctx, in.LocationID)
	if err != nil {
		return nil, err
	}

	// Get current quant
	quant, err := uc.repo.GetQuant(ctx, in.ProductID, in.LocationID)
	if err != nil {
		return nil, err
	}

	delta := in.NewQuantity - quant.Quantity
	if delta == 0 {
		return quant, nil
	}

	// Fetch or fallback to inventory adjustment virtual location
	adjLoc, err := uc.repo.GetLocationByUsage(ctx, stock.LocationUsageInventory)
	var adjLocID int64
	if err == nil {
		adjLocID = adjLoc.ID
	} else {
		adjLocID = 5 // standard virtual inventory adjustment location
	}

	var srcID, destID int64
	var moveQty float64
	var moveName string
	if delta > 0 {
		srcID = adjLocID
		destID = in.LocationID
		moveQty = delta
		moveName = fmt.Sprintf("Inventory Gain: %s (new count %.2f)", in.Note, in.NewQuantity)
	} else {
		srcID = in.LocationID
		destID = adjLocID
		moveQty = -delta
		moveName = fmt.Sprintf("Inventory Loss: %s (new count %.2f)", in.Note, in.NewQuantity)
	}

	now := time.Now().UTC()
	picking := &stock.StockPicking{
		Name:           fmt.Sprintf("INV/ADJ/%d/%d-%d", now.Year(), in.ProductID, now.UnixNano()),
		PickingType:    stock.PickingTypeInternal,
		State:          stock.PickingStateDraft,
		LocationID:     srcID,
		LocationDestID: destID,
		ScheduledDate:  now,
		Origin:         "Inventory Adjustment",
		Note:           in.Note,
		Active:         true,
		Moves: []stock.StockMove{
			{
				Name:           moveName,
				ProductID:      in.ProductID,
				ProductQty:     moveQty,
				QuantityDone:   moveQty,
				LocationID:     srcID,
				LocationDestID: destID,
				State:          stock.MoveStateDraft,
				Date:           now,
			},
		},
	}

	if err := uc.repo.CreatePicking(ctx, picking); err != nil {
		return nil, err
	}

	if err := uc.repo.ValidatePickingTx(ctx, picking); err != nil {
		return nil, err
	}

	// Phase 12: value the adjustment move so the valuation trail stays consistent.
	if err := uc.ValuatePicking(ctx, picking); err != nil {
		uc.logger.ErrorContext(ctx, "stock adjustment valuation failed", "product_id", in.ProductID, "error", err)
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock adjusted successfully",
		"product_id", in.ProductID, "location", loc.CompleteName, "old_qty", quant.Quantity, "new_qty", in.NewQuantity)

	return uc.repo.GetQuant(ctx, in.ProductID, in.LocationID)
}
