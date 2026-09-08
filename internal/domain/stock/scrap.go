package stock

import (
	"time"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type ScrapState string

const (
	ScrapDraft ScrapState = "draft"
	ScrapDone  ScrapState = "done"
)

type StockScrap struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	ProductID       int64      `json:"product_id"`
	Quantity        float64    `json:"quantity"`
	ProductUom      *int64     `json:"product_uom,omitempty"`
	LocationID      int64      `json:"location_id"`       // Source
	ScrapLocationID int64      `json:"scrap_location_id"` // Destination (usage=scrap)
	LotID           *int64     `json:"lot_id,omitempty"`
	State           ScrapState `json:"state"`
	Date            time.Time  `json:"date"`
	MoveID          *int64     `json:"move_id,omitempty"`
	CompanyID       int64      `json:"company_id"`
}

func (s *StockScrap) Validate() error {
	if s.ProductID <= 0 {
		return platformerrors.Validation("product is required", nil)
	}
	if s.Quantity <= 0 {
		return platformerrors.Validation("quantity must be positive", nil)
	}
	if s.LocationID <= 0 || s.ScrapLocationID <= 0 {
		return platformerrors.Validation("source and scrap locations are required", nil)
	}
	if s.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	return nil
}
