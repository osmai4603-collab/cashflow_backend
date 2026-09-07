package mrp

import (
	"time"

	"cashflow_backend/internal/platform/audit"
)

// ProductionState defines the lifecycle of a Manufacturing Order.
type ProductionState string

const (
	ProductionStateDraft     ProductionState = "draft"
	ProductionStateConfirmed ProductionState = "confirmed"
	ProductionStateProgress  ProductionState = "progress"
	ProductionStateToClose   ProductionState = "to_close"
	ProductionStateDone      ProductionState = "done"
	ProductionStateCancel    ProductionState = "cancel"
)

// ReservationState defines the readiness of components.
type ReservationState string

const (
	ReservationStateWaiting  ReservationState = "confirmed" // Waiting for materials
	ReservationStateReady    ReservationState = "assigned"  // Materials available
	ReservationStatePartial  ReservationState = "partial"
)

// ProductionOrder represents a Manufacturing Order (mrp.production in Odoo).
type ProductionOrder struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"` // Reference like MO/2026/0001
	Priority        int              `json:"priority"`
	BackorderSeq    int              `json:"backorder_sequence"`
	Origin          string           `json:"origin,omitempty"`

	ProductID       int64            `json:"product_id"`
	ProductQty      float64          `json:"product_qty"`
	UoMID           int64            `json:"uom_id"`
	QtyProducing    float64          `json:"qty_producing"`
	QtyProduced     float64          `json:"qty_produced"`

	BomID           int64            `json:"bom_id"`
	PickingTypeID   int64            `json:"picking_type_id"`
	LocationSrcID   int64            `json:"location_src_id"`
	LocationDestID  int64            `json:"location_dest_id"`

	DateDeadline    *time.Time       `json:"date_deadline,omitempty"`
	DateStart       time.Time        `json:"date_start"`
	DateFinished    *time.Time       `json:"date_finished,omitempty"`

	State           ProductionState  `json:"state"`
	ReservationState ReservationState `json:"reservation_state"`

	MoveRawIDs      []int64          `json:"move_raw_ids,omitempty"`      // StockMoves for components
	MoveFinishedIDs []int64          `json:"move_finished_ids,omitempty"` // StockMoves for finished products

	CompanyID       int64            `json:"company_id"`
	Audit           audit.Fields     `json:"audit"`
}

// Validate ensures ProductionOrder invariants.
func (p *ProductionOrder) Validate() error {
	if p.ProductID <= 0 {
		return interface{}(nil).(error) // Use standard error handling style if needed
	}
	if p.ProductQty <= 0 {
		p.ProductQty = 1.0
	}
	if p.State == "" {
		p.State = ProductionStateDraft
	}
	if p.DateStart.IsZero() {
		p.DateStart = time.Now()
	}
	return nil
}
