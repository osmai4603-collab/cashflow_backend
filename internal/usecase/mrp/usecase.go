package mrpusecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/mrp"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	sequenceusecase "cashflow_backend/internal/usecase/sequence"
)

type Usecase struct {
	repo        mrp.Repository
	sequence    sequenceusecase.UseCase
	stockRepo   stock.Repository
	accounting  *accountingusecase.UseCase
	productRepo product.Repository
}

func NewUsecase(repo mrp.Repository, seq sequenceusecase.UseCase, stockRepo stock.Repository, acc *accountingusecase.UseCase, products ...product.Repository) *Usecase {
	u := &Usecase{
		repo:       repo,
		sequence:   seq,
		stockRepo:  stockRepo,
		accounting: acc,
	}
	if len(products) > 0 {
		u.productRepo = products[0]
	}
	return u
}

// ─────────────────────────────────────────────────────────────────────────────
// Workcenter Usecases
// ─────────────────────────────────────────────────────────────────────────────

type CreateWorkcenterInput struct {
	Name           string  `json:"name"`
	Code           string  `json:"code"`
	Sequence       int     `json:"sequence"`
	CompanyID      int64   `json:"company_id"`
	TimeStart      float64 `json:"time_start"`
	TimeStop       float64 `json:"time_stop"`
	TimeEfficiency float64 `json:"time_efficiency"`
	Capacity       float64 `json:"capacity"`
	CostPerHour    float64 `json:"cost_per_hour"`
}

func (u *Usecase) CreateWorkcenter(ctx context.Context, userID int64, in CreateWorkcenterInput) (*mrp.Workcenter, error) {
	wc := &mrp.Workcenter{
		Name:           in.Name,
		Code:           in.Code,
		Active:         true,
		Sequence:       in.Sequence,
		CompanyID:      in.CompanyID,
		TimeStart:      in.TimeStart,
		TimeStop:       in.TimeStop,
		TimeEfficiency: in.TimeEfficiency,
		Capacity:       in.Capacity,
		CostPerHour:    in.CostPerHour,
		Audit: audit.Fields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: &userID,
			UpdatedBy: &userID,
		},
	}

	if err := wc.Validate(); err != nil {
		return nil, err
	}

	if err := u.repo.CreateWorkcenter(ctx, wc); err != nil {
		return nil, err
	}
	return wc, nil
}

func (u *Usecase) GetWorkcenter(ctx context.Context, id int64) (*mrp.Workcenter, error) {
	return u.repo.GetWorkcenterByID(ctx, id)
}

func (u *Usecase) ListWorkcenters(ctx context.Context, f mrp.WorkcenterFilter) ([]*mrp.Workcenter, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 80
	}
	return u.repo.ListWorkcenters(ctx, f)
}

// ─────────────────────────────────────────────────────────────────────────────
// BoM Usecases
// ─────────────────────────────────────────────────────────────────────────────

type CreateBoMInput struct {
	Code           string                 `json:"code"`
	ProductID      int64                  `json:"product_id"`
	ProductQty     float64                `json:"product_qty"`
	UoMID          int64                  `json:"uom_id"`
	Type           mrp.BomType            `json:"type"`
	ReadyToProduce string                 `json:"ready_to_produce"`
	Consumption    string                 `json:"consumption"`
	CompanyID      int64                  `json:"company_id"`
	Lines          []CreateBomLineInput   `json:"lines"`
	Operations     []CreateOperationInput `json:"operations"`
}

type CreateBomLineInput struct {
	ProductID   int64   `json:"product_id"`
	Quantity    float64 `json:"quantity"`
	UoMID       int64   `json:"uom_id"`
	OperationID *int64  `json:"operation_id"`
	Sequence    int     `json:"sequence"`
}

type CreateOperationInput struct {
	WorkcenterID    int64   `json:"workcenter_id"`
	Name            string  `json:"name"`
	Sequence        int     `json:"sequence"`
	TimeMode        string  `json:"time_mode"`
	TimeCycleManual float64 `json:"time_cycle_manual"`
}

func (u *Usecase) CreateBoM(ctx context.Context, userID int64, in CreateBoMInput) (*mrp.BillOfMaterials, error) {
	bom := &mrp.BillOfMaterials{
		Code:           in.Code,
		ProductID:      in.ProductID,
		ProductQty:     in.ProductQty,
		UoMID:          in.UoMID,
		Type:           in.Type,
		ReadyToProduce: in.ReadyToProduce,
		Consumption:    in.Consumption,
		Active:         true,
		CompanyID:      in.CompanyID,
		Audit: audit.Fields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: &userID,
			UpdatedBy: &userID,
		},
	}

	for _, l := range in.Lines {
		bom.Lines = append(bom.Lines, mrp.BomLine{
			ProductID:   l.ProductID,
			Quantity:    l.Quantity,
			UoMID:       l.UoMID,
			OperationID: l.OperationID,
			Sequence:    l.Sequence,
		})
	}

	for _, o := range in.Operations {
		bom.Operations = append(bom.Operations, mrp.RoutingOperation{
			WorkcenterID:    o.WorkcenterID,
			Name:            o.Name,
			Sequence:        o.Sequence,
			TimeMode:        o.TimeMode,
			TimeCycleManual: o.TimeCycleManual,
		})
	}

	if err := bom.Validate(); err != nil {
		return nil, err
	}

	if err := u.repo.CreateBoM(ctx, bom); err != nil {
		return nil, err
	}
	return bom, nil
}

func (u *Usecase) GetBoM(ctx context.Context, id int64) (*mrp.BillOfMaterials, error) {
	return u.repo.GetBoMByID(ctx, id)
}

func (u *Usecase) ListBoMs(ctx context.Context, f mrp.BoMFilter) ([]*mrp.BillOfMaterials, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 80
	}
	return u.repo.ListBoMs(ctx, f)
}

// ─────────────────────────────────────────────────────────────────────────────
// Production Order Usecases
// ─────────────────────────────────────────────────────────────────────────────

type CreateProductionInput struct {
	ProductID      int64      `json:"product_id"`
	ProductQty     float64    `json:"product_qty"`
	BomID          int64      `json:"bom_id"`
	DateDeadline   *time.Time `json:"date_deadline"`
	DateStart      time.Time  `json:"date_start"`
	CompanyID      int64      `json:"company_id"`
	PickingTypeID  int64      `json:"picking_type_id"`
	LocationSrcID  int64      `json:"location_src_id"`
	LocationDestID int64      `json:"location_dest_id"`
	Origin         string     `json:"origin"`
}

func (u *Usecase) CreateProduction(ctx context.Context, userID int64, in CreateProductionInput) (*mrp.ProductionOrder, error) {
	// Generate Name
	name := "New"
	if u.sequence != nil {
		var err error
		name, err = u.sequence.GenerateNext(ctx, "mrp.production", time.Now())
		if err != nil {
			// Fallback or handle error
			name = "MO/TEMP/" + time.Now().Format("20060102150405")
		}
	}

	mo := &mrp.ProductionOrder{
		Name:           name,
		ProductID:      in.ProductID,
		ProductQty:     in.ProductQty,
		BomID:          in.BomID,
		DateDeadline:   in.DateDeadline,
		DateStart:      in.DateStart,
		State:          mrp.ProductionStateDraft,
		CompanyID:      in.CompanyID,
		PickingTypeID:  in.PickingTypeID,
		LocationSrcID:  in.LocationSrcID,
		LocationDestID: in.LocationDestID,
		Origin:         in.Origin,
		Audit: audit.Fields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: &userID,
			UpdatedBy: &userID,
		},
	}

	if err := mo.Validate(); err != nil {
		return nil, err
	}

	if err := u.repo.CreateProduction(ctx, mo); err != nil {
		return nil, err
	}
	return mo, nil
}

func (u *Usecase) ConfirmProduction(ctx context.Context, userID int64, id int64) (*mrp.ProductionOrder, error) {
	mo, err := u.repo.GetProductionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if mo.State != mrp.ProductionStateDraft {
		return mo, nil
	}

	// 1. Get BoM to load operations
	bom, err := u.repo.GetBoMByID(ctx, mo.BomID)
	if err != nil {
		return nil, err
	}
	components := make([]mrp.ExplodedComponent, 0, len(bom.Lines))
	if u.productRepo != nil && bom.UoMID > 0 {
		components, err = mrp.ExplodeBoM(bom, mo.ProductQty,
			func(productID int64) (*mrp.BillOfMaterials, error) {
				child, childErr := u.repo.GetBoMByID(ctx, productID)
				var appErr *platformerrors.AppError
				if childErr != nil && errors.As(childErr, &appErr) && appErr.Code == platformerrors.CodeNotFound {
					return nil, nil
				}
				return child, childErr
			},
			func(uomID int64) (mrp.UoM, error) {
				uom, uomErr := u.productRepo.GetUoMByID(ctx, uomID)
				if uomErr != nil {
					return mrp.UoM{}, uomErr
				}
				return mrp.UoM{ID: uom.ID, Category: uom.Category, Ratio: uom.Ratio, Rounding: uom.Rounding}, nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("explode BoM: %w", err)
		}
	} else {
		for _, line := range bom.Lines {
			components = append(components, mrp.ExplodedComponent{
				ProductID: line.ProductID, UoMID: line.UoMID,
				Quantity: line.Quantity * mo.ProductQty, OperationID: line.OperationID,
				SourceBomID: bom.ID,
			})
		}
	}

	// 2. Create Workorders from BoM Operations
	for _, op := range bom.Operations {
		wo := &mrp.Workorder{
			ProductionID:     mo.ID,
			WorkcenterID:     op.WorkcenterID,
			OperationID:      op.ID,
			Name:             op.Name,
			Sequence:         op.Sequence,
			State:            mrp.WorkorderStateReady,
			DurationExpected: op.TimeCycleManual,
			Audit: audit.Fields{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: &userID,
				UpdatedBy: &userID,
			},
		}
		if err := u.repo.CreateWorkorder(ctx, wo); err != nil {
			return nil, err
		}
	}

	// 3. Create Stock Moves for Raw Materials
	prodLoc, err := u.stockRepo.GetLocationByUsage(ctx, stock.LocationUsageProduction)
	if err != nil {
		return nil, fmt.Errorf("failed to find production location: %w", err)
	}

	allMaterialsReady := true
	for _, component := range components {
		move := &stock.StockMove{
			Name:           fmt.Sprintf("Raw Material: %s", mo.Name),
			ProductID:      component.ProductID,
			ProductQty:     component.Quantity,
			ProductUom:     &component.UoMID,
			LocationID:     mo.LocationSrcID,
			LocationDestID: prodLoc.ID,
			State:          stock.MoveStateConfirmed,
			ProductionID:   &mo.ID,
			Date:           time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := u.stockRepo.CreateMove(ctx, move); err != nil {
			return nil, err
		}
		if err := u.stockRepo.ReserveMove(ctx, move); err != nil {
			return nil, fmt.Errorf("reserve raw material %d: %w", move.ProductID, err)
		}
		if move.ReservedQuantity < move.ProductQty {
			allMaterialsReady = false
		}
	}

	// 4. Create Stock Move for Finished Product
	finishMove := &stock.StockMove{
		Name:                 mo.Name,
		ProductID:            mo.ProductID,
		ProductQty:           mo.ProductQty,
		ProductUom:           &mo.UoMID,
		LocationID:           prodLoc.ID,
		LocationDestID:       mo.LocationDestID,
		State:                stock.MoveStateConfirmed,
		ProductionFinishedID: &mo.ID,
		Date:                 time.Now(),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	if err := u.stockRepo.CreateMove(ctx, finishMove); err != nil {
		return nil, err
	}
	if allMaterialsReady {
		mo.ReservationState = mrp.ReservationStateReady
	} else {
		mo.ReservationState = mrp.ReservationStateWaiting
	}

	mo.State = mrp.ProductionStateConfirmed
	mo.Audit.UpdatedAt = time.Now()
	mo.Audit.UpdatedBy = &userID

	if err := u.repo.UpdateProduction(ctx, mo); err != nil {
		return nil, err
	}

	return mo, nil
}

func (u *Usecase) GetProduction(ctx context.Context, id int64) (*mrp.ProductionOrder, error) {
	return u.repo.GetProductionByID(ctx, id)
}

func (u *Usecase) ListProductions(ctx context.Context, f mrp.ProductionFilter) ([]*mrp.ProductionOrder, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 80
	}
	return u.repo.ListProductions(ctx, f)
}

// ─────────────────────────────────────────────────────────────────────────────
// Workorder Usecases
// ─────────────────────────────────────────────────────────────────────────────

func (u *Usecase) StartWorkorder(ctx context.Context, userID int64, id int64) (*mrp.Workorder, error) {
	wo, err := u.repo.GetWorkorderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if wo.State == mrp.WorkorderStateDone {
		return nil, fmt.Errorf("cannot start completed work order")
	}
	if wo.State == mrp.WorkorderStateCancel {
		return nil, fmt.Errorf("cannot start cancelled work order")
	}
	if wo.State == mrp.WorkorderStateBlocked {
		return nil, fmt.Errorf("cannot start blocked work order")
	}

	now := time.Now()
	wo.State = mrp.WorkorderStateProgress
	wo.DateStart = &now
	wo.Audit.UpdatedAt = now
	wo.Audit.UpdatedBy = &userID

	if err := u.repo.UpdateWorkorder(ctx, wo); err != nil {
		return nil, err
	}

	// Update MO state to In Progress if it's the first WO started
	mo, err := u.repo.GetProductionByID(ctx, wo.ProductionID)
	if err == nil && mo.State == mrp.ProductionStateConfirmed {
		mo.State = mrp.ProductionStateProgress
		u.repo.UpdateProduction(ctx, mo)
	}

	return wo, nil
}

func (u *Usecase) DoneWorkorder(ctx context.Context, userID int64, id int64, duration float64) (*mrp.Workorder, error) {
	wo, err := u.repo.GetWorkorderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if wo.State == mrp.WorkorderStateDone {
		return nil, fmt.Errorf("work order is already completed")
	}
	if wo.State == mrp.WorkorderStateCancel {
		return nil, fmt.Errorf("cannot complete cancelled work order")
	}
	if wo.State != mrp.WorkorderStateProgress {
		return nil, fmt.Errorf("cannot complete work order in state %s", wo.State)
	}
	if duration < 0 {
		return nil, fmt.Errorf("work order duration cannot be negative")
	}

	now := time.Now()
	wo.State = mrp.WorkorderStateDone
	wo.DateFinished = &now
	wo.Duration = duration
	wo.Audit.UpdatedAt = now
	wo.Audit.UpdatedBy = &userID

	if err := u.repo.UpdateWorkorder(ctx, wo); err != nil {
		return nil, err
	}

	return wo, nil
}

func (u *Usecase) ProduceProduction(ctx context.Context, userID int64, id int64) (*mrp.ProductionOrder, error) {
	return u.ProduceProductionQty(ctx, userID, id, 0)
}

// ProduceProductionQty completes the requested quantity of an MO. A zero
// quantity uses qty_producing, falling back to the full planned quantity.
func (u *Usecase) ProduceProductionQty(ctx context.Context, userID int64, id int64, quantity float64) (*mrp.ProductionOrder, error) {
	mo, err := u.repo.GetProductionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if mo.State != mrp.ProductionStateConfirmed && mo.State != mrp.ProductionStateProgress {
		return nil, fmt.Errorf("invalid state for production: %s", mo.State)
	}
	if quantity == 0 {
		quantity = mo.QtyProducing
		if quantity == 0 {
			quantity = mo.ProductQty
		}
	}
	if quantity <= 0 || quantity > mo.ProductQty {
		return nil, fmt.Errorf("production quantity must be greater than zero and no more than %g", mo.ProductQty)
	}

	workorders, err := u.repo.ListWorkorders(ctx, mo.ID)
	if err != nil {
		return nil, err
	}
	for _, wo := range workorders {
		if wo.State != mrp.WorkorderStateDone {
			return nil, fmt.Errorf("work order %d is not completed", wo.ID)
		}
	}
	quantityRatio := quantity / mo.ProductQty

	// 1. Get raw material moves and calculate cost
	rawFilter := filter.NewFilter(filter.Criterion{Field: "production_id", Operator: filter.OpEqual, Value: mo.ID})
	rawMovesRes, err := u.stockRepo.ListMoves(ctx, rawFilter, pagination.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}

	var totalRawCost float64
	completedMoves := make([]stock.StockMove, 0, len(rawMovesRes.Items))
	for _, m := range rawMovesRes.Items {
		if m.State != stock.MoveStateDone && m.State != stock.MoveStateCancel {
			rawQuantity := m.ProductQty * quantityRatio
			if err := m.ActionDone(rawQuantity); err != nil {
				return nil, fmt.Errorf("complete raw material move %d: %w", m.ID, err)
			}
			// Ensure move is valued
			if m.StandardPrice == 0 {
				return nil, fmt.Errorf("raw material move %d has no standard price", m.ID)
			}
			m.Value = rawQuantity * m.StandardPrice
			totalRawCost += m.Value
			completedMoves = append(completedMoves, m)
		}
	}

	// 2. Calculate Workcenter costs
	var totalWcCost float64
	for _, wo := range workorders {
		wc, err := u.repo.GetWorkcenterByID(ctx, wo.WorkcenterID)
		if err != nil {
			return nil, fmt.Errorf("get workcenter %d: %w", wo.WorkcenterID, err)
		}
		if wc != nil {
			totalWcCost += (wo.Duration / 60.0) * wc.CostPerHour
		}
	}

	totalCost := totalRawCost + totalWcCost

	// 3. Get finished product moves
	finFilter := filter.NewFilter(filter.Criterion{Field: "production_finished_id", Operator: filter.OpEqual, Value: mo.ID})
	finMovesRes, err := u.stockRepo.ListMoves(ctx, finFilter, pagination.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}

	// 4. Validate finished product moves and set value
	for _, m := range finMovesRes.Items {
		if m.State != stock.MoveStateDone && m.State != stock.MoveStateCancel {
			if err := m.ActionDone(quantity); err != nil {
				return nil, fmt.Errorf("complete finished product move %d: %w", m.ID, err)
			}
			m.Value = totalCost // Assign total production cost to finished move
			m.StandardPrice = totalCost / quantity
			completedMoves = append(completedMoves, m)
		}
	}
	if err := u.stockRepo.ValidateMovesTx(ctx, completedMoves); err != nil {
		return nil, fmt.Errorf("post production stock moves: %w", err)
	}
	for i := range completedMoves {
		if err := u.stockRepo.UpdateMoveValue(ctx, &completedMoves[i]); err != nil {
			return nil, fmt.Errorf("persist production valuation for move %d: %w", completedMoves[i].ID, err)
		}
	}

	now := time.Now()
	mo.QtyProducing = quantity
	mo.QtyProduced += quantity
	if mo.QtyProduced >= mo.ProductQty {
		mo.QtyProduced = mo.ProductQty
		mo.State = mrp.ProductionStateDone
	} else {
		mo.State = mrp.ProductionStateToClose
	}
	mo.DateFinished = &now
	mo.Audit.UpdatedAt = now
	mo.Audit.UpdatedBy = &userID

	if err := u.repo.UpdateProduction(ctx, mo); err != nil {
		return nil, err
	}

	return mo, nil
}

func (u *Usecase) ListWorkorders(ctx context.Context, productionID int64) ([]*mrp.Workorder, error) {
	return u.repo.ListWorkorders(ctx, productionID)
}

// ─────────────────────────────────────────────────────────────────────────────
// Unbuild Usecases
// ─────────────────────────────────────────────────────────────────────────────

type CreateUnbuildInput struct {
	ProductID      int64   `json:"product_id"`
	BomID          int64   `json:"bom_id"`
	MOID           *int64  `json:"mo_id"`
	Quantity       float64 `json:"quantity"`
	UoMID          int64   `json:"uom_id"`
	LocationID     int64   `json:"location_id"`
	DestLocationID int64   `json:"dest_location_id"`
	CompanyID      int64   `json:"company_id"`
}

func (u *Usecase) CreateUnbuild(ctx context.Context, userID int64, in CreateUnbuildInput) (*mrp.UnbuildOrder, error) {
	name := "New"
	if u.sequence != nil {
		name, _ = u.sequence.GenerateNext(ctx, "mrp.unbuild", time.Now())
	}

	uo := &mrp.UnbuildOrder{
		Name:           name,
		ProductID:      in.ProductID,
		BomID:          in.BomID,
		MOID:           in.MOID,
		Quantity:       in.Quantity,
		UoMID:          in.UoMID,
		LocationID:     in.LocationID,
		DestLocationID: in.DestLocationID,
		State:          mrp.ProductionStateDone, // Unbuild happens immediately
		CompanyID:      in.CompanyID,
		Audit: audit.Fields{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: &userID,
			UpdatedBy: &userID,
		},
	}

	// 1. Find Production Location
	prodLoc, err := u.stockRepo.GetLocationByUsage(ctx, stock.LocationUsageProduction)
	if err != nil {
		return nil, err
	}

	// 2. Consume Finished Good
	consumeMove := &stock.StockMove{
		Name:           fmt.Sprintf("Unbuild Consume: %s", uo.Name),
		ProductID:      uo.ProductID,
		ProductQty:     uo.Quantity,
		ProductUom:     &uo.UoMID,
		LocationID:     uo.LocationID,
		LocationDestID: prodLoc.ID,
		State:          stock.MoveStateDone,
		Date:           time.Now(),
	}
	u.stockRepo.CreateMove(ctx, consumeMove)

	// 3. Produce Raw Materials (Reverse BoM)
	bom, err := u.repo.GetBoMByID(ctx, uo.BomID)
	if err == nil {
		for _, line := range bom.Lines {
			produceMove := &stock.StockMove{
				Name:           fmt.Sprintf("Unbuild Produce: %s", uo.Name),
				ProductID:      line.ProductID,
				ProductQty:     line.Quantity * uo.Quantity,
				ProductUom:     &line.UoMID,
				LocationID:     prodLoc.ID,
				LocationDestID: uo.DestLocationID,
				State:          stock.MoveStateDone,
				Date:           time.Now(),
			}
			u.stockRepo.CreateMove(ctx, produceMove)
		}
	}

	if err := u.repo.CreateUnbuild(ctx, uo); err != nil {
		return nil, err
	}
	return uo, nil
}

func (u *Usecase) GetUnbuild(ctx context.Context, id int64) (*mrp.UnbuildOrder, error) {
	return u.repo.GetUnbuildByID(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Reporting Usecases
// ─────────────────────────────────────────────────────────────────────────────

type WorkcenterOEEReport struct {
	WorkcenterID   int64   `json:"workcenter_id"`
	WorkcenterName string  `json:"workcenter_name"`
	TotalExpected  float64 `json:"total_expected"`
	TotalActual    float64 `json:"total_actual"`
	OEE            float64 `json:"oee"`
}

func (u *Usecase) GetWorkcenterOEEReport(ctx context.Context, companyID int64) ([]WorkcenterOEEReport, error) {
	wcs, _, err := u.repo.ListWorkcenters(ctx, mrp.WorkcenterFilter{CompanyID: &companyID})
	if err != nil {
		return nil, err
	}

	var report []WorkcenterOEEReport
	for _, wc := range wcs {
		item := WorkcenterOEEReport{
			WorkcenterID:   wc.ID,
			WorkcenterName: wc.Name,
		}
		// In a real system, we would query and aggregate mrp_workorders
		report = append(report, item)
	}

	return report, nil
}
