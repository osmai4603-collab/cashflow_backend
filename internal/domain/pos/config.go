package pos

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type PosConfig struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	WarehouseID          int64     `json:"warehouse_id"`
	StockLocationID      int64     `json:"stock_location_id"`
	JournalID            int64     `json:"journal_id"`
	InvoiceJournalID     *int64    `json:"invoice_journal_id,omitempty"`
	PaymentMethodIDs     []int64   `json:"payment_method_ids"`
	ModulePosRestaurant  bool      `json:"module_pos_restaurant"`
	UpdateStockAtClosing bool      `json:"update_stock_at_closing"`
	AllowDiscount        bool      `json:"allow_discount"`
	ManualDiscountLimit  float64   `json:"manual_discount_limit"`
	Active               bool      `json:"active"`
	CompanyID            int64     `json:"company_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (config *PosConfig) Validate() error {
	config.Name = strings.TrimSpace(config.Name)
	if config.Name == "" {
		return platformerrors.Validation("point of sale name is required", map[string]string{"name": "cannot be empty"})
	}
	if config.WarehouseID <= 0 || config.StockLocationID <= 0 || config.JournalID <= 0 || config.CompanyID <= 0 {
		return platformerrors.Validation("point of sale references are required", map[string]string{"references": "warehouse, stock location, journal, and company must be valid"})
	}
	if config.ManualDiscountLimit < 0 || config.ManualDiscountLimit > 100 {
		return platformerrors.Validation("manual discount limit must be between 0 and 100", map[string]string{"manual_discount_limit": "must be in [0, 100]"})
	}
	return nil
}
