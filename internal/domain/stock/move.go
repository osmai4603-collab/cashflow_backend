package stock

import (
	"fmt"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// MoveState represents the lifecycle status of an individual stock movement (stock.move in Odoo).
type MoveState string

const (
	MoveStateDraft     MoveState = "draft"     // New movement line
	MoveStateWaiting   MoveState = "waiting"   // Waiting for another move
	MoveStateConfirmed MoveState = "confirmed" // Confirmed / Planned
	MoveStateAssigned  MoveState = "assigned"  // Stock reserved & ready
	MoveStateDone      MoveState = "done"      // Successfully executed / quants updated
	MoveStateCancel    MoveState = "cancel"    // Movement cancelled
)

// StockMove represents a movement of a specified quantity of a product between two locations.
type StockMove struct {
	ID                   int64     `json:"id"`
	PickingID            *int64    `json:"picking_id,omitempty"`
	Sequence             int       `json:"sequence"`
	Name                 string    `json:"name"`
	ProductID            int64     `json:"product_id"`
	ProductUom           *int64    `json:"product_uom,omitempty"`
	ProductQty           float64   `json:"product_qty"`
	ReservedQuantity     float64   `json:"reserved_quantity"`
	QuantityDone         float64   `json:"quantity_done"`
	LocationID           int64     `json:"location_id"`
	LocationDestID       int64     `json:"location_dest_id"`
	State                MoveState `json:"state"`
	SaleLineID           *int64    `json:"sale_line_id,omitempty"`
	PurchaseLineID       *int64    `json:"purchase_line_id,omitempty"`
	ProductionID         *int64    `json:"production_id,omitempty"`          // Raw material for this MO
	ProductionFinishedID *int64    `json:"production_finished_id,omitempty"` // Finished product for this MO
	ProcurementGroupID   *int64    `json:"procurement_group_id,omitempty"`
	Date                 time.Time `json:"date"`
	// Valuation fields (Phase 12 — stock_account integration).
	Value          float64         `json:"value"`                     // current valuation value of the move (0 when not valued)
	ValueManual    *float64        `json:"value_manual,omitempty"`    // manual override → triggers a ProductValue history record
	StandardPrice  float64         `json:"standard_price"`            // unit cost captured at valuation time
	IsIn           bool            `json:"is_in"`                     // valued incoming move
	IsOut          bool            `json:"is_out"`                    // valued outgoing move
	IsDropship     bool            `json:"is_dropship"`               // supplier→customer direct move (valued)
	RemainingQty   float64         `json:"remaining_qty"`             // FIFO stack: remaining qty still in stock
	RemainingValue float64         `json:"remaining_value"`           // FIFO stack: remaining value
	AccountMoveID  *int64          `json:"account_move_id,omitempty"` // journal entry created for this move
	MoveLines      []StockMoveLine `json:"move_lines,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Validate checks business invariants for StockMove.
func (m *StockMove) Validate() error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return platformerrors.Validation("move name or description is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if m.ProductID <= 0 {
		return platformerrors.Validation("product is required", map[string]string{
			"product_id": "must reference a valid product",
		})
	}

	if m.ProductQty <= 0 {
		return platformerrors.Validation("product quantity must be positive", map[string]string{
			"product_qty": "must be greater than 0",
		})
	}
	if m.ReservedQuantity < 0 || m.ReservedQuantity > m.ProductQty {
		return platformerrors.Validation("reserved quantity is outside the requested quantity", map[string]string{
			"reserved_quantity": "must be between 0 and product_qty",
		})
	}

	if m.LocationID <= 0 {
		return platformerrors.Validation("source location is required", map[string]string{
			"location_id": "must reference a valid source location",
		})
	}

	if m.LocationDestID <= 0 {
		return platformerrors.Validation("destination location is required", map[string]string{
			"location_dest_id": "must reference a valid destination location",
		})
	}

	if m.LocationID == m.LocationDestID {
		return platformerrors.Validation("source and destination locations cannot be identical", map[string]string{
			"location_dest_id": "must be different from source location",
		})
	}

	if m.State == "" {
		m.State = MoveStateDraft
	}

	switch m.State {
	case MoveStateDraft, MoveStateWaiting, MoveStateConfirmed, MoveStateAssigned, MoveStateDone, MoveStateCancel:
		// Valid state
	default:
		return platformerrors.Validation("invalid move state", map[string]string{
			"state": fmt.Sprintf("invalid move state '%s'", m.State),
		})
	}

	return nil
}

// ActionConfirm advances the move to confirmed state.
func (m *StockMove) ActionConfirm() error {
	if m.State != MoveStateDraft {
		return platformerrors.Conflict(fmt.Sprintf("cannot confirm stock move in state '%s'; must be draft", m.State))
	}
	m.State = MoveStateConfirmed
	return nil
}

// ActionDone completes the move with the validated quantity.
func (m *StockMove) ActionDone(qtyDone float64) error {
	if m.State == MoveStateDone {
		return platformerrors.Conflict("stock move is already completed")
	}
	if m.State == MoveStateCancel {
		return platformerrors.Conflict("cannot execute cancelled stock move")
	}
	if qtyDone < 0 {
		return platformerrors.Validation("done quantity cannot be negative", map[string]string{
			"quantity_done": "must be >= 0",
		})
	}
	if qtyDone == 0 {
		qtyDone = m.ProductQty
	}
	if qtyDone > m.ProductQty {
		return platformerrors.Validation("done quantity cannot exceed requested quantity", map[string]string{
			"quantity_done": "must be less than or equal to product_qty",
		})
	}
	m.QuantityDone = qtyDone
	m.State = MoveStateDone
	return nil
}

// ActionCancel cancels the stock move.
func (m *StockMove) ActionCancel() error {
	if m.State == MoveStateDone {
		return platformerrors.Conflict("cannot cancel a completed stock move")
	}
	m.State = MoveStateCancel
	return nil
}
