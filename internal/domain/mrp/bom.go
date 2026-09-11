package mrp

import (
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// BomType defines the nature of the Bill of Materials.
type BomType string

const (
	BomTypeNormal      BomType = "normal"      // Regular manufacturing
	BomTypePhantom     BomType = "phantom"     // Kit/Bundle, components delivered instead of finished product
	BomTypeSubcontract BomType = "subcontract" // Manufacturing performed by an external partner
)

// BillOfMaterials (BoM) defines components and operations for a product (mrp.bom).
type BillOfMaterials struct {
	ID             int64   `json:"id"`
	Code           string  `json:"code,omitempty"`
	ProductID      int64   `json:"product_id"`
	ProductQty     float64 `json:"product_qty"`
	UoMID          int64   `json:"uom_id"`
	Type           BomType `json:"type"`
	ReadyToProduce string  `json:"ready_to_produce"` // all_available, asap
	Consumption    string  `json:"consumption"`      // flexible, warning, strict
	Active         bool    `json:"active"`
	CompanyID      int64   `json:"company_id"`

	Lines      []BomLine          `json:"lines,omitempty"`
	ByProducts []BomByProduct     `json:"by_products,omitempty"`
	Operations []RoutingOperation `json:"operations,omitempty"`

	Audit audit.Fields `json:"audit"`
}

// BomByProduct represents a byproduct of a manufacturing process (mrp.bom.byproduct).
type BomByProduct struct {
	ID        int64   `json:"id"`
	BomID     int64   `json:"bom_id"`
	ProductID int64   `json:"product_id"`
	Quantity  float64 `json:"product_qty"`
	UoMID     int64   `json:"uom_id"`
}

// Validate ensures BoM invariants.
func (b *BillOfMaterials) Validate() error {
	if b.ProductID <= 0 {
		return platformerrors.Validation("product is required", map[string]string{
			"product_id": "must be valid",
		})
	}
	if b.ProductQty <= 0 {
		return platformerrors.Validation("quantity must be positive", map[string]string{
			"product_qty": "must be > 0",
		})
	}
	if b.Type == "" {
		b.Type = BomTypeNormal
	}
	if b.ReadyToProduce == "" {
		b.ReadyToProduce = "all_available"
	}
	if b.Consumption == "" {
		b.Consumption = "flexible"
	}
	return nil
}

// BomLine represents a component in a BoM (mrp.bom.line).
type BomLine struct {
	ID          int64   `json:"id"`
	BomID       int64   `json:"bom_id"`
	ProductID   int64   `json:"product_id"`
	Quantity    float64 `json:"quantity"`
	UoMID       int64   `json:"uom_id"`
	OperationID *int64  `json:"operation_id,omitempty"` // The operation where this component is consumed
	Sequence    int     `json:"sequence"`
}

// RoutingOperation represents a step in the manufacturing process (mrp.routing.workcenter).
type RoutingOperation struct {
	ID              int64   `json:"id"`
	BomID           int64   `json:"bom_id"`
	WorkcenterID    int64   `json:"workcenter_id"`
	Name            string  `json:"name"`
	Sequence        int     `json:"sequence"`
	TimeMode        string  `json:"time_mode"`         // manual, computed
	TimeCycleManual float64 `json:"time_cycle_manual"` // expected duration in minutes
}

// Validate ensures RoutingOperation invariants.
func (o *RoutingOperation) Validate() error {
	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" {
		return platformerrors.Validation("operation name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if o.WorkcenterID <= 0 {
		return platformerrors.Validation("workcenter is required", map[string]string{
			"workcenter_id": "must be valid",
		})
	}
	return nil
}
