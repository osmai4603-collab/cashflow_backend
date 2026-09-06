package stock

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// StockQuant represents the physical stock balance of a product in a specific location (stock.quant in Odoo).
type StockQuant struct {
	ID               int64     `json:"id"`
	ProductID        int64     `json:"product_id"`
	LocationID       int64     `json:"location_id"`
	Quantity         float64   `json:"quantity"`
	ReservedQuantity float64   `json:"reserved_quantity"`
	CompanyID        *int64    `json:"company_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AvailableQuantity returns the unreserved on-hand quantity.
func (q *StockQuant) AvailableQuantity() float64 {
	avail := q.Quantity - q.ReservedQuantity
	if avail < 0 {
		return 0
	}
	return avail
}

// Validate checks StockQuant invariants.
func (q *StockQuant) Validate() error {
	if q.ProductID <= 0 {
		return platformerrors.Validation("product is required", map[string]string{
			"product_id": "must reference a valid product",
		})
	}
	if q.LocationID <= 0 {
		return platformerrors.Validation("location is required", map[string]string{
			"location_id": "must reference a valid location",
		})
	}
	if q.ReservedQuantity < 0 {
		return platformerrors.Validation("reserved quantity cannot be negative", map[string]string{
			"reserved_quantity": "must be >= 0",
		})
	}
	return nil
}

// StockOnHandItem represents an enriched record of stock on hand for reports and API responses.
type StockOnHandItem struct {
	ProductID         int64   `json:"product_id"`
	ProductName       string  `json:"product_name"`
	ProductSKU        string  `json:"product_sku,omitempty"`
	ProductType       string  `json:"product_type"`
	LocationID        int64   `json:"location_id"`
	LocationName      string  `json:"location_name"`
	WarehouseID       *int64  `json:"warehouse_id,omitempty"`
	WarehouseName     string  `json:"warehouse_name,omitempty"`
	Quantity          float64 `json:"quantity"`
	ReservedQuantity  float64 `json:"reserved_quantity"`
	AvailableQuantity float64 `json:"available_quantity"`
	UoMName           string  `json:"uom_name,omitempty"`
}
