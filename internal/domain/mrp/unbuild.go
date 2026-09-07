package mrp

import (
	"cashflow_backend/internal/platform/audit"
)

// UnbuildOrder represents the reversal of a manufacturing process (mrp.unbuild in Odoo).
type UnbuildOrder struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	ProductID      int64   `json:"product_id"`
	BomID          int64   `json:"bom_id"`
	MOID           *int64  `json:"mo_id,omitempty"` // Reference to original MO
	Quantity       float64 `json:"quantity"`
	UoMID          int64   `json:"uom_id"`
	LocationID     int64   `json:"location_id"`
	DestLocationID int64   `json:"dest_location_id"`

	State     ProductionState `json:"state"` // reuse ProductionState: draft, done
	CompanyID int64           `json:"company_id"`
	Audit     audit.Fields    `json:"audit"`
}
