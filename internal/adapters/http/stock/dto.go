package stockhttp

import (
	"time"

	"cashflow_backend/internal/domain/stock"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateLocationRequest struct {
	Name           string              `json:"name"`
	Usage          stock.LocationUsage `json:"usage"`
	ParentID       *int64              `json:"parent_id,omitempty"`
	ScrapLocation  bool                `json:"scrap_location"`
	ReturnLocation bool                `json:"return_location"`
	CompanyID      *int64              `json:"company_id,omitempty"`
}

func (r CreateLocationRequest) ToInput() stockusecase.CreateLocationInput {
	return stockusecase.CreateLocationInput{
		Name:           r.Name,
		Usage:          r.Usage,
		ParentID:       r.ParentID,
		ScrapLocation:  r.ScrapLocation,
		ReturnLocation: r.ReturnLocation,
		CompanyID:      r.CompanyID,
	}
}

type UpdateLocationRequest struct {
	Name           *string              `json:"name,omitempty"`
	Usage          *stock.LocationUsage `json:"usage,omitempty"`
	ParentID       *int64               `json:"parent_id,omitempty"`
	ScrapLocation  *bool                `json:"scrap_location,omitempty"`
	ReturnLocation *bool                `json:"return_location,omitempty"`
	CompanyID      *int64               `json:"company_id,omitempty"`
}

func (r UpdateLocationRequest) ToInput() stockusecase.UpdateLocationInput {
	return stockusecase.UpdateLocationInput{
		Name:           r.Name,
		Usage:          r.Usage,
		ParentID:       r.ParentID,
		ScrapLocation:  r.ScrapLocation,
		ReturnLocation: r.ReturnLocation,
		CompanyID:      r.CompanyID,
	}
}

type CreateWarehouseRequest struct {
	Name           string `json:"name"`
	Code           string `json:"code"`
	LotStockID     int64  `json:"lot_stock_id"`
	ViewLocationID *int64 `json:"view_location_id,omitempty"`
	PartnerID      *int64 `json:"partner_id,omitempty"`
	CompanyID      *int64 `json:"company_id,omitempty"`
}

func (r CreateWarehouseRequest) ToInput() stockusecase.CreateWarehouseInput {
	return stockusecase.CreateWarehouseInput{
		Name:           r.Name,
		Code:           r.Code,
		LotStockID:     r.LotStockID,
		ViewLocationID: r.ViewLocationID,
		PartnerID:      r.PartnerID,
		CompanyID:      r.CompanyID,
	}
}

type UpdateWarehouseRequest struct {
	Name           *string `json:"name,omitempty"`
	Code           *string `json:"code,omitempty"`
	LotStockID     *int64  `json:"lot_stock_id,omitempty"`
	ViewLocationID *int64  `json:"view_location_id,omitempty"`
	PartnerID      *int64  `json:"partner_id,omitempty"`
	CompanyID      *int64  `json:"company_id,omitempty"`
}

func (r UpdateWarehouseRequest) ToInput() stockusecase.UpdateWarehouseInput {
	return stockusecase.UpdateWarehouseInput{
		Name:           r.Name,
		Code:           r.Code,
		LotStockID:     r.LotStockID,
		ViewLocationID: r.ViewLocationID,
		PartnerID:      r.PartnerID,
		CompanyID:      r.CompanyID,
	}
}

type CreateMoveRequest struct {
	ProductID      int64   `json:"product_id"`
	ProductQty     float64 `json:"product_qty"`
	LocationID     *int64  `json:"location_id,omitempty"`
	LocationDestID *int64  `json:"location_dest_id,omitempty"`
	Name           string  `json:"name,omitempty"`
	ProductUom     *int64  `json:"product_uom,omitempty"`
	Sequence       int     `json:"sequence,omitempty"`
	SaleLineID     *int64  `json:"sale_line_id,omitempty"`
	PurchaseLineID *int64  `json:"purchase_line_id,omitempty"`
}

type CreatePickingRequest struct {
	PickingType    stock.PickingType   `json:"picking_type"`
	PartnerID      *int64              `json:"partner_id,omitempty"`
	LocationID     *int64              `json:"location_id,omitempty"`
	LocationDestID *int64              `json:"location_dest_id,omitempty"`
	ScheduledDate  *time.Time          `json:"scheduled_date,omitempty"`
	Origin         string              `json:"origin,omitempty"`
	SourceOrderID  *int64              `json:"source_order_id,omitempty"`
	CompanyID      *int64              `json:"company_id,omitempty"`
	Note           string              `json:"note,omitempty"`
	Moves          []CreateMoveRequest `json:"moves"`
}

func (r CreatePickingRequest) ToInput() stockusecase.CreatePickingInput {
	var sched time.Time
	if r.ScheduledDate != nil {
		sched = *r.ScheduledDate
	}

	moves := make([]stockusecase.CreateMoveInput, len(r.Moves))
	for i, m := range r.Moves {
		moves[i] = stockusecase.CreateMoveInput{
			ProductID:      m.ProductID,
			ProductQty:     m.ProductQty,
			LocationID:     m.LocationID,
			LocationDestID: m.LocationDestID,
			Name:           m.Name,
			ProductUom:     m.ProductUom,
			Sequence:       m.Sequence,
			SaleLineID:     m.SaleLineID,
			PurchaseLineID: m.PurchaseLineID,
		}
	}

	return stockusecase.CreatePickingInput{
		PickingType:    r.PickingType,
		PartnerID:      r.PartnerID,
		LocationID:     r.LocationID,
		LocationDestID: r.LocationDestID,
		ScheduledDate:  sched,
		Origin:         r.Origin,
		SourceOrderID:  r.SourceOrderID,
		CompanyID:      r.CompanyID,
		Note:           r.Note,
		Moves:          moves,
	}
}

type UpdatePickingRequest struct {
	PartnerID      *int64              `json:"partner_id,omitempty"`
	LocationID     *int64              `json:"location_id,omitempty"`
	LocationDestID *int64              `json:"location_dest_id,omitempty"`
	ScheduledDate  *time.Time          `json:"scheduled_date,omitempty"`
	Origin         *string             `json:"origin,omitempty"`
	Note           *string             `json:"note,omitempty"`
	Moves          []CreateMoveRequest `json:"moves,omitempty"`
}

func (r UpdatePickingRequest) ToInput() stockusecase.UpdatePickingInput {
	var moves []stockusecase.CreateMoveInput
	if r.Moves != nil {
		moves = make([]stockusecase.CreateMoveInput, len(r.Moves))
		for i, m := range r.Moves {
			moves[i] = stockusecase.CreateMoveInput{
				ProductID:      m.ProductID,
				ProductQty:     m.ProductQty,
				LocationID:     m.LocationID,
				LocationDestID: m.LocationDestID,
				Name:           m.Name,
				ProductUom:     m.ProductUom,
				Sequence:       m.Sequence,
				SaleLineID:     m.SaleLineID,
				PurchaseLineID: m.PurchaseLineID,
			}
		}
	}

	return stockusecase.UpdatePickingInput{
		PartnerID:      r.PartnerID,
		LocationID:     r.LocationID,
		LocationDestID: r.LocationDestID,
		ScheduledDate:  r.ScheduledDate,
		Origin:         r.Origin,
		Note:           r.Note,
		Moves:          moves,
	}
}

type ValidatePickingRequest struct {
	EffectiveDate *time.Time `json:"effective_date,omitempty"`
}

func (r ValidatePickingRequest) ToInput() stockusecase.ValidatePickingInput {
	return stockusecase.ValidatePickingInput{
		EffectiveDate: r.EffectiveDate,
	}
}

type StockAdjustmentRequest struct {
	ProductID   int64   `json:"product_id"`
	LocationID  int64   `json:"location_id"`
	NewQuantity float64 `json:"new_quantity"`
	Note        string  `json:"note,omitempty"`
}

func (r StockAdjustmentRequest) ToInput() stockusecase.StockAdjustmentInput {
	return stockusecase.StockAdjustmentInput{
		ProductID:   r.ProductID,
		LocationID:  r.LocationID,
		NewQuantity: r.NewQuantity,
		Note:        r.Note,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Response DTOs
// ─────────────────────────────────────────────────────────────────────────────

type LocationResponse struct {
	ID             int64               `json:"id"`
	Name           string              `json:"name"`
	CompleteName   string              `json:"complete_name"`
	Usage          stock.LocationUsage `json:"usage"`
	ParentID       *int64              `json:"parent_id,omitempty"`
	ScrapLocation  bool                `json:"scrap_location"`
	ReturnLocation bool                `json:"return_location"`
	CompanyID      *int64              `json:"company_id,omitempty"`
	Active         bool                `json:"active"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

func ToLocationResponse(l *stock.StockLocation) LocationResponse {
	return LocationResponse{
		ID:             l.ID,
			Name:           string(l.Name),
			CompleteName:   string(l.CompleteName),
		Usage:          l.Usage,
		ParentID:       l.ParentID,
		ScrapLocation:  l.ScrapLocation,
		ReturnLocation: l.ReturnLocation,
		CompanyID:      l.CompanyID,
		Active:         l.Active,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
	}
}

type WarehouseResponse struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	LotStockID     int64     `json:"lot_stock_id"`
	ViewLocationID *int64    `json:"view_location_id,omitempty"`
	PartnerID      *int64    `json:"partner_id,omitempty"`
	CompanyID      *int64    `json:"company_id,omitempty"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func ToWarehouseResponse(w *stock.Warehouse) WarehouseResponse {
	return WarehouseResponse{
		ID:             w.ID,
			Name:           string(w.Name),
		Code:           w.Code,
		LotStockID:     w.LotStockID,
		ViewLocationID: w.ViewLocationID,
		PartnerID:      w.PartnerID,
		CompanyID:      w.CompanyID,
		Active:         w.Active,
		CreatedAt:      w.CreatedAt,
		UpdatedAt:      w.UpdatedAt,
	}
}

type MoveResponse struct {
	ID             int64           `json:"id"`
	PickingID      *int64          `json:"picking_id,omitempty"`
	Sequence       int             `json:"sequence"`
	Name           string          `json:"name"`
	ProductID      int64           `json:"product_id"`
	ProductUom     *int64          `json:"product_uom,omitempty"`
	ProductQty     float64         `json:"product_qty"`
	QuantityDone   float64         `json:"quantity_done"`
	LocationID     int64           `json:"location_id"`
	LocationDestID int64           `json:"location_dest_id"`
	State          stock.MoveState `json:"state"`
	SaleLineID     *int64          `json:"sale_line_id,omitempty"`
	PurchaseLineID *int64          `json:"purchase_line_id,omitempty"`
	Date           time.Time       `json:"date"`
}

func ToMoveResponse(m *stock.StockMove) MoveResponse {
	return MoveResponse{
		ID:             m.ID,
		PickingID:      m.PickingID,
		Sequence:       m.Sequence,
		Name:           m.Name,
		ProductID:      m.ProductID,
		ProductUom:     m.ProductUom,
		ProductQty:     m.ProductQty,
		QuantityDone:   m.QuantityDone,
		LocationID:     m.LocationID,
		LocationDestID: m.LocationDestID,
		State:          m.State,
		SaleLineID:     m.SaleLineID,
		PurchaseLineID: m.PurchaseLineID,
		Date:           m.Date,
	}
}

type PickingResponse struct {
	ID             int64              `json:"id"`
	Name           string             `json:"name"`
	PickingType    stock.PickingType  `json:"picking_type"`
	State          stock.PickingState `json:"state"`
	PartnerID      *int64             `json:"partner_id,omitempty"`
	LocationID     int64              `json:"location_id"`
	LocationDestID int64              `json:"location_dest_id"`
	ScheduledDate  time.Time          `json:"scheduled_date"`
	DateDone       *time.Time         `json:"date_done,omitempty"`
	Origin         string             `json:"origin,omitempty"`
	SourceOrderID  *int64             `json:"source_order_id,omitempty"`
	CompanyID      *int64             `json:"company_id,omitempty"`
	Note           string             `json:"note,omitempty"`
	Active         bool               `json:"active"`
	Moves          []MoveResponse     `json:"moves,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

func ToPickingResponse(p *stock.StockPicking) PickingResponse {
	moves := make([]MoveResponse, len(p.Moves))
	for i, m := range p.Moves {
		moves[i] = ToMoveResponse(&m)
	}

	return PickingResponse{
		ID:             p.ID,
		Name:           p.Name,
		PickingType:    p.PickingType,
		State:          p.State,
		PartnerID:      p.PartnerID,
		LocationID:     p.LocationID,
		LocationDestID: p.LocationDestID,
		ScheduledDate:  p.ScheduledDate,
		DateDone:       p.DateDone,
		Origin:         p.Origin,
		SourceOrderID:  p.SourceOrderID,
		CompanyID:      p.CompanyID,
		Note:           p.Note,
		Active:         p.Active,
		Moves:          moves,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

type QuantResponse struct {
	ID                int64   `json:"id"`
	ProductID         int64   `json:"product_id"`
	LocationID        int64   `json:"location_id"`
	Quantity          float64 `json:"quantity"`
	ReservedQuantity  float64 `json:"reserved_quantity"`
	AvailableQuantity float64 `json:"available_quantity"`
}

func ToQuantResponse(q *stock.StockQuant) QuantResponse {
	return QuantResponse{
		ID:                q.ID,
		ProductID:         q.ProductID,
		LocationID:        q.LocationID,
		Quantity:          q.Quantity,
		ReservedQuantity:  q.ReservedQuantity,
		AvailableQuantity: q.AvailableQuantity(),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Valuation DTOs (Phase 12 — stock-account integration)
// ─────────────────────────────────────────────────────────────────────────────

type AdjustMoveValueRequest struct {
	Value       float64 `json:"value"`
	Description string  `json:"description"`
	UserID      int64   `json:"user_id"`
	CompanyID   int64   `json:"company_id"`
}

func (r AdjustMoveValueRequest) ToInput(moveID int64) stockusecase.AdjustMoveValueInput {
	return stockusecase.AdjustMoveValueInput{
		MoveID:      moveID,
		Value:       r.Value,
		Description: r.Description,
		UserID:      r.UserID,
		CompanyID:   r.CompanyID,
	}
}

type CreateAccountingPeriodRequest struct {
	Name      string    `json:"name"`
	DateFrom  time.Time `json:"date_from"`
	DateTo    time.Time `json:"date_to"`
	JournalID int64     `json:"journal_id"`
	CompanyID int64     `json:"company_id"`
}

func (r CreateAccountingPeriodRequest) ToInput() stockusecase.CreateAccountingPeriodInput {
	return stockusecase.CreateAccountingPeriodInput{
		Name:      r.Name,
		DateFrom:  r.DateFrom,
		DateTo:    r.DateTo,
		JournalID: r.JournalID,
		CompanyID: r.CompanyID,
	}
}

type ValuationSummaryResponse struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	LocationID  *int64  `json:"location_id,omitempty"`
	Location    string  `json:"location,omitempty"`
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	Value       float64 `json:"value"`
}

func ToValuationSummaryResponse(s stock.ValuationSummary) ValuationSummaryResponse {
	return ValuationSummaryResponse{
		ProductID:   s.ProductID,
		ProductName: s.ProductName,
		LocationID:  s.LocationID,
		Location:    s.Location,
		Quantity:    s.Quantity,
		UnitCost:    s.UnitCost,
		Value:       s.Value,
	}
}

type AccountingPeriodResponse struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	DateFrom      time.Time `json:"date_from"`
	DateTo        time.Time `json:"date_to"`
	State         string    `json:"state"`
	JournalID     int64     `json:"journal_id"`
	AccountMoveID *int64    `json:"account_move_id,omitempty"`
	CompanyID     int64     `json:"company_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func ToAccountingPeriodResponse(p *stock.AccountingPeriod) AccountingPeriodResponse {
	return AccountingPeriodResponse{
		ID:            p.ID,
		Name:          p.Name,
		DateFrom:      p.DateFrom,
		DateTo:        p.DateTo,
		State:         p.State,
		JournalID:     p.JournalID,
		AccountMoveID: p.AccountMoveID,
		CompanyID:     p.CompanyID,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Orderpoint DTOs (Phase 13 — auto reorder)
// ─────────────────────────────────────────────────────────────────────────────

type OrderpointRequest struct {
	ProductID        int64      `json:"product_id"`
	WarehouseID      int64      `json:"warehouse_id"`
	LocationID       int64      `json:"location_id"`
	VendorID         *int64     `json:"vendor_id,omitempty"`
	MinQty           float64    `json:"min_qty"`
	MaxQty           float64    `json:"max_qty"`
	QtyMultiple      float64    `json:"qty_multiple"`
	LeadDays         int        `json:"lead_days"`
	Trigger          string     `json:"trigger"`
	SnoozedUntil     *time.Time `json:"snoozed_until,omitempty"`
	QtyToOrderManual float64    `json:"qty_to_order_manual"`
	CompanyID        int64      `json:"company_id"`
}

func (r OrderpointRequest) ToCreateInput() stockusecase.CreateOrderpointInput {
	return stockusecase.CreateOrderpointInput{
		ProductID:        r.ProductID,
		WarehouseID:      r.WarehouseID,
		LocationID:       r.LocationID,
		VendorID:         r.VendorID,
		MinQty:           r.MinQty,
		MaxQty:           r.MaxQty,
		QtyMultiple:      r.QtyMultiple,
		LeadDays:         r.LeadDays,
		Trigger:          stock.OrderpointTrigger(r.Trigger),
		SnoozedUntil:     r.SnoozedUntil,
		QtyToOrderManual: r.QtyToOrderManual,
		CompanyID:        r.CompanyID,
	}
}

type OrderpointUpdateRequest struct {
	ProductID        *int64     `json:"product_id,omitempty"`
	WarehouseID      *int64     `json:"warehouse_id,omitempty"`
	LocationID       *int64     `json:"location_id,omitempty"`
	VendorID         *int64     `json:"vendor_id,omitempty"`
	MinQty           *float64   `json:"min_qty,omitempty"`
	MaxQty           *float64   `json:"max_qty,omitempty"`
	QtyMultiple      *float64   `json:"qty_multiple,omitempty"`
	LeadDays         *int       `json:"lead_days,omitempty"`
	Trigger          *string    `json:"trigger,omitempty"`
	SnoozedUntil     *time.Time `json:"snoozed_until,omitempty"`
	ClearSnooze      *bool      `json:"clear_snooze,omitempty"`
	QtyToOrderManual *float64   `json:"qty_to_order_manual,omitempty"`
	Active           *bool      `json:"active,omitempty"`
}

func (r OrderpointUpdateRequest) ToUpdateInput() stockusecase.UpdateOrderpointInput {
	input := stockusecase.UpdateOrderpointInput{
		ProductID:        r.ProductID,
		WarehouseID:      r.WarehouseID,
		LocationID:       r.LocationID,
		VendorID:         r.VendorID,
		MinQty:           r.MinQty,
		MaxQty:           r.MaxQty,
		QtyMultiple:      r.QtyMultiple,
		LeadDays:         r.LeadDays,
		SnoozedUntil:     r.SnoozedUntil,
		ClearSnooze:      r.ClearSnooze,
		QtyToOrderManual: r.QtyToOrderManual,
		Active:           r.Active,
	}
	if r.Trigger != nil {
		t := stock.OrderpointTrigger(*r.Trigger)
		input.Trigger = &t
	}
	return input
}

type OrderpointResponse struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	ProductID        int64      `json:"product_id"`
	WarehouseID      int64      `json:"warehouse_id"`
	LocationID       int64      `json:"location_id"`
	VendorID         *int64     `json:"vendor_id,omitempty"`
	MinQty           float64    `json:"min_qty"`
	MaxQty           float64    `json:"max_qty"`
	QtyMultiple      float64    `json:"qty_multiple"`
	LeadDays         int        `json:"lead_days"`
	Source           string     `json:"source"`
	Trigger          string     `json:"trigger"`
	SnoozedUntil     *time.Time `json:"snoozed_until,omitempty"`
	QtyOnHand        float64    `json:"qty_on_hand"`
	QtyForecast      float64    `json:"qty_forecast"`
	QtyToOrder       float64    `json:"qty_to_order"`
	QtyToOrderManual float64    `json:"qty_to_order_manual"`
	DeadlineDate     *time.Time `json:"deadline_date,omitempty"`
	Active           bool       `json:"active"`
	CompanyID        int64      `json:"company_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func ToOrderpointResponse(op *stock.Orderpoint) OrderpointResponse {
	return OrderpointResponse{
		ID:               op.ID,
		Name:             op.Name,
		ProductID:        op.ProductID,
		WarehouseID:      op.WarehouseID,
		LocationID:       op.LocationID,
		VendorID:         op.VendorID,
		MinQty:           op.MinQty,
		MaxQty:           op.MaxQty,
		QtyMultiple:      op.QtyMultiple,
		LeadDays:         op.LeadDays,
		Source:           string(op.Source),
		Trigger:          string(op.Trigger),
		SnoozedUntil:     op.SnoozedUntil,
		QtyOnHand:        op.QtyOnHand,
		QtyForecast:      op.QtyForecast,
		QtyToOrder:       op.QtyToOrder,
		QtyToOrderManual: op.QtyToOrderManual,
		DeadlineDate:     op.DeadlineDate,
		Active:           op.Active,
		CompanyID:        op.CompanyID,
		CreatedAt:        op.CreatedAt,
		UpdatedAt:        op.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Landed Cost DTOs (Phase 13 — stock landed costs)
// ─────────────────────────────────────────────────────────────────────────────

type LandedCostLineRequest struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name"`
	ProductID   int64   `json:"product_id"`
	AccountID   int64   `json:"account_id"`
	PriceUnit   float64 `json:"price_unit"`
	SplitMethod string  `json:"split_method"`
}

type LandedCostRequest struct {
	PickingIDs   []int64                 `json:"picking_ids"`
	Date         time.Time               `json:"date"`
	JournalID    int64                   `json:"journal_id"`
	VendorBillID *int64                  `json:"vendor_bill_id,omitempty"`
	Description  string                  `json:"description,omitempty"`
	CompanyID    int64                   `json:"company_id"`
	CostLines    []LandedCostLineRequest `json:"cost_lines"`
}

func (r LandedCostRequest) ToInput() stockusecase.CreateLandedCostInput {
	lines := make([]stockusecase.CreateLandedCostLineInput, len(r.CostLines))
	for i, l := range r.CostLines {
		lines[i] = l.ToInput()
	}
	return stockusecase.CreateLandedCostInput{
		PickingIDs:   r.PickingIDs,
		Date:         r.Date,
		JournalID:    r.JournalID,
		VendorBillID: r.VendorBillID,
		Description:  r.Description,
		CompanyID:    r.CompanyID,
		CostLines:    lines,
	}
}

func (r LandedCostLineRequest) ToInput() stockusecase.CreateLandedCostLineInput {
	return stockusecase.CreateLandedCostLineInput{
		Name:        r.Name,
		ProductID:   r.ProductID,
		AccountID:   r.AccountID,
		PriceUnit:   r.PriceUnit,
		SplitMethod: stock.SplitMethod(r.SplitMethod),
	}
}

type LandedCostUpdateRequest struct {
	PickingIDs  *[]int64                 `json:"picking_ids,omitempty"`
	Date        *time.Time               `json:"date,omitempty"`
	Description *string                  `json:"description,omitempty"`
	CostLines   *[]LandedCostLineRequest `json:"cost_lines,omitempty"`
}

func (r LandedCostUpdateRequest) ToUpdateInput() stockusecase.UpdateLandedCostInput {
	input := stockusecase.UpdateLandedCostInput{
		PickingIDs:  r.PickingIDs,
		Date:        r.Date,
		Description: r.Description,
	}
	if r.CostLines != nil {
		lines := make([]stockusecase.CreateLandedCostLineInput, len(*r.CostLines))
		for i, l := range *r.CostLines {
			lines[i] = l.ToInput()
		}
		input.CostLines = &lines
	}
	return input
}

type CreateLandedCostFromBillRequest struct {
	PickingIDs  []int64 `json:"picking_ids"`
	Description string  `json:"description,omitempty"`
	CompanyID   int64   `json:"company_id"`
}

type LandedCostLineResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ProductID   int64   `json:"product_id"`
	AccountID   int64   `json:"account_id"`
	PriceUnit   float64 `json:"price_unit"`
	SplitMethod string  `json:"split_method"`
}

type ValuationAdjustmentResponse struct {
	ID               int64   `json:"id"`
	LandedCostID     int64   `json:"landed_cost_id"`
	CostLineID       int64   `json:"cost_line_id"`
	MoveID           int64   `json:"move_id"`
	ProductID        int64   `json:"product_id"`
	Quantity         float64 `json:"quantity"`
	Weight           float64 `json:"weight"`
	Volume           float64 `json:"volume"`
	FormerCost       float64 `json:"former_cost"`
	AdditionalCost   float64 `json:"additional_cost"`
	FinalCost        float64 `json:"final_cost"`
	MoveRemainingQty float64 `json:"move_remaining_qty"`
}

type LandedCostResponse struct {
	ID                   int64                         `json:"id"`
	Name                 string                        `json:"name"`
	Date                 time.Time                     `json:"date"`
	State                string                        `json:"state"`
	PickingIDs           []int64                       `json:"picking_ids"`
	CostLines            []LandedCostLineResponse      `json:"cost_lines"`
	ValuationAdjustments []ValuationAdjustmentResponse `json:"valuation_adjustments"`
	Description          string                        `json:"description,omitempty"`
	AmountTotal          float64                       `json:"amount_total"`
	AccountMoveID        *int64                        `json:"account_move_id,omitempty"`
	JournalID            int64                         `json:"journal_id"`
	VendorBillID         *int64                        `json:"vendor_bill_id,omitempty"`
	CompanyID            int64                         `json:"company_id"`
	CreatedAt            time.Time                     `json:"created_at"`
	UpdatedAt            time.Time                     `json:"updated_at"`
}

func ToLandedCostResponse(lc *stock.LandedCost) LandedCostResponse {
	lines := make([]LandedCostLineResponse, len(lc.CostLines))
	for i, l := range lc.CostLines {
		lines[i] = LandedCostLineResponse{
			ID:          l.ID,
			Name:        l.Name,
			ProductID:   l.ProductID,
			AccountID:   l.AccountID,
			PriceUnit:   l.PriceUnit,
			SplitMethod: string(l.SplitMethod),
		}
	}
	adjusts := make([]ValuationAdjustmentResponse, len(lc.ValuationAdjustments))
	for i, a := range lc.ValuationAdjustments {
		adjusts[i] = ValuationAdjustmentResponse{
			ID:               a.ID,
			LandedCostID:     a.LandedCostID,
			CostLineID:       a.CostLineID,
			MoveID:           a.MoveID,
			ProductID:        a.ProductID,
			Quantity:         a.Quantity,
			Weight:           a.Weight,
			Volume:           a.Volume,
			FormerCost:       a.FormerCost,
			AdditionalCost:   a.AdditionalCost,
			FinalCost:        a.FinalCost,
			MoveRemainingQty: a.MoveRemainingQty,
		}
	}
	return LandedCostResponse{
		ID:                   lc.ID,
		Name:                 lc.Name,
		Date:                 lc.Date,
		State:                string(lc.State),
		PickingIDs:           lc.PickingIDs,
		CostLines:            lines,
		ValuationAdjustments: adjusts,
		Description:          lc.Description,
		AmountTotal:          lc.AmountTotal,
		AccountMoveID:        lc.AccountMoveID,
		JournalID:            lc.JournalID,
		VendorBillID:         lc.VendorBillID,
		CompanyID:            lc.CompanyID,
		CreatedAt:            lc.CreatedAt,
		UpdatedAt:            lc.UpdatedAt,
	}
}
