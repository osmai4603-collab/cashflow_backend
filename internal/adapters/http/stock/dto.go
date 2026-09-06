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
		Name:           l.Name,
		CompleteName:   l.CompleteName,
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
		Name:           w.Name,
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
