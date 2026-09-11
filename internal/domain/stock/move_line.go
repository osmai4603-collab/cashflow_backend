package stock

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// StockMoveLine is the detailed reservation and execution line of a stock move.
// It mirrors Odoo stock.move.line and can optionally identify a lot or serial.
type StockMoveLine struct {
	ID               int64        `json:"id"`
	MoveID           int64        `json:"move_id"`
	ProductID        int64        `json:"product_id"`
	TrackingMode     TrackingMode `json:"tracking_mode"`
	ProductUom       *int64       `json:"product_uom,omitempty"`
	LotID            *int64       `json:"lot_id,omitempty"`
	PackageID        *int64       `json:"package_id,omitempty"`
	ResultPackageID  *int64       `json:"result_package_id,omitempty"`
	OwnerID          *int64       `json:"owner_id,omitempty"`
	LocationID       int64        `json:"location_id"`
	LocationDestID   int64        `json:"location_dest_id"`
	ReservedQuantity float64      `json:"reserved_quantity"`
	QuantityDone     float64      `json:"quantity_done"`
	Date             time.Time    `json:"date"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// Validate checks detailed stock reservation and execution invariants.
func (l *StockMoveLine) Validate() error {
	if l.MoveID <= 0 {
		return platformerrors.Validation("move is required", map[string]string{"move_id": "must be positive"})
	}
	if l.ProductID <= 0 {
		return platformerrors.Validation("product is required", map[string]string{"product_id": "must be positive"})
	}
	if l.TrackingMode == "" {
		l.TrackingMode = TrackingNone
	}
	if l.TrackingMode != TrackingNone && l.TrackingMode != TrackingLot && l.TrackingMode != TrackingSerial {
		return platformerrors.Validation("invalid move line tracking mode", map[string]string{"tracking_mode": "must be none, lot or serial"})
	}
	if l.TrackingMode != TrackingNone && l.LotID == nil {
		return platformerrors.Validation("tracked move line requires a lot or serial", map[string]string{"lot_id": "is required"})
	}
	if l.LocationID <= 0 || l.LocationDestID <= 0 {
		return platformerrors.Validation("move line locations are required", map[string]string{"location_id": "must be positive"})
	}
	if l.LocationID == l.LocationDestID {
		return platformerrors.Validation("move line locations cannot be identical", map[string]string{"location_dest_id": "must differ from location_id"})
	}
	if l.ReservedQuantity < 0 || l.QuantityDone < 0 {
		return platformerrors.Validation("move line quantities cannot be negative", nil)
	}
	return nil
}
