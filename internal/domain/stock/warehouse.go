package stock

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Warehouse represents a physical logistics facility or building (stock.warehouse in Odoo).
type Warehouse struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	CompanyID      *int64    `json:"company_id,omitempty"`
	PartnerID      *int64    `json:"partner_id,omitempty"`
	ViewLocationID *int64    `json:"view_location_id,omitempty"`
	LotStockID     int64     `json:"lot_stock_id"` // Default internal stock location
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      *int64    `json:"created_by,omitempty"`
	UpdatedBy      *int64    `json:"updated_by,omitempty"`
}

// Validate checks business invariants for Warehouse.
func (w *Warehouse) Validate() error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return platformerrors.Validation("warehouse name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if len(w.Name) > 128 {
		return platformerrors.Validation("warehouse name exceeds maximum length", map[string]string{
			"name": "must not exceed 128 characters",
		})
	}

	w.Code = strings.ToUpper(strings.TrimSpace(w.Code))
	if w.Code == "" {
		return platformerrors.Validation("warehouse code is required", map[string]string{
			"code": "cannot be empty",
		})
	}
	if len(w.Code) > 16 {
		return platformerrors.Validation("warehouse code exceeds maximum length", map[string]string{
			"code": "must not exceed 16 characters",
		})
	}

	if w.LotStockID <= 0 {
		return platformerrors.Validation("default stock location is required", map[string]string{
			"lot_stock_id": "must be a valid stock location ID",
		})
	}

	return nil
}
