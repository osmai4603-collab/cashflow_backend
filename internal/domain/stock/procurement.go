package stock

import (
	"time"
)

// ProcurementGroup is used to group stock moves together, typically representing
// a single customer order or vendor order (procurement.group in Odoo).
type ProcurementGroup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"` // Reference name, e.g., SO number
	MoveIDs   []int64   `json:"move_ids,omitempty"`
	CompanyID int64     `json:"company_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
