package stock

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// TrackingMode defines how a product lot is identified in stock operations.
type TrackingMode string

const (
	TrackingNone   TrackingMode = "none"
	TrackingLot    TrackingMode = "lot"
	TrackingSerial TrackingMode = "serial"
)

// StockLot identifies a lot or serial number for one product.
type StockLot struct {
	ID           int64        `json:"id"`
	ProductID    int64        `json:"product_id"`
	Name         string       `json:"name"`
	TrackingMode TrackingMode `json:"tracking_mode"`
	CompanyID    *int64       `json:"company_id,omitempty"`
	ExpirationAt *time.Time   `json:"expiration_at,omitempty"`
	Active       bool         `json:"active"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func (l *StockLot) Validate() error {
	l.Name = strings.TrimSpace(l.Name)
	if l.ProductID <= 0 {
		return platformerrors.Validation("lot product is required", map[string]string{"product_id": "must be positive"})
	}
	if l.Name == "" {
		return platformerrors.Validation("lot name is required", map[string]string{"name": "cannot be empty"})
	}
	if l.TrackingMode == "" {
		l.TrackingMode = TrackingLot
	}
	switch l.TrackingMode {
	case TrackingLot, TrackingSerial:
	default:
		return platformerrors.Validation("invalid lot tracking mode", map[string]string{"tracking_mode": "must be lot or serial"})
	}
	return nil
}
