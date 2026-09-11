package stock

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// StockAccountConfig holds the accounting configuration for inventory valuation.
// Mirrors Odoo 19 stock_account property accounts on product categories/companies.
type StockAccountConfig struct {
	ID                      int64     `json:"id"`
	CompanyID               int64     `json:"company_id"`
	ProductCategoryID       *int64    `json:"product_category_id,omitempty"`
	StockValuationAccountID int64     `json:"stock_valuation_account_id"`      // Inventory asset account (e.g. 140000 Inventory)
	StockInputAccountID     int64     `json:"stock_input_account_id"`          // Goods Received interim account (e.g. 140100 Stock Interim Received)
	StockOutputAccountID    int64     `json:"stock_output_account_id"`         // Goods Delivered interim / COGS account (e.g. 500000 COGS)
	StockJournalID          int64     `json:"stock_journal_id"`               // Stock operations journal (e.g. STJ)
	PriceDiffAccountID      *int64    `json:"price_diff_account_id,omitempty"` // Price difference account
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// Validate checks business invariants for StockAccountConfig.
func (c *StockAccountConfig) Validate() error {
	if c.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", map[string]string{
			"company_id": "must reference a valid company",
		})
	}
	if c.StockValuationAccountID <= 0 {
		return platformerrors.Validation("stock_valuation_account_id is required", map[string]string{
			"stock_valuation_account_id": "must reference a valid valuation account",
		})
	}
	if c.StockInputAccountID <= 0 {
		return platformerrors.Validation("stock_input_account_id is required", map[string]string{
			"stock_input_account_id": "must reference a valid interim input account",
		})
	}
	if c.StockOutputAccountID <= 0 {
		return platformerrors.Validation("stock_output_account_id is required", map[string]string{
			"stock_output_account_id": "must reference a valid interim output account",
		})
	}
	if c.StockJournalID <= 0 {
		return platformerrors.Validation("stock_journal_id is required", map[string]string{
			"stock_journal_id": "must reference a valid stock journal",
		})
	}
	return nil
}
