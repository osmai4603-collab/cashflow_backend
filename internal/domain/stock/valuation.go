package stock

import (
	"fmt"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// CostMethod defines how a product's stock moves are valued (stock.move/company cost_method in Odoo 19).
type CostMethod string

const (
	CostStandard CostMethod = "standard" // Standard Price — fixed valuation price
	CostFIFO     CostMethod = "fifo"     // First In First Out — consume oldest incoming layers first
	CostAverage  CostMethod = "average"  // Average Cost (AVCO) — running weighted average
)

// Validate reports whether the cost method is supported.
func (m CostMethod) Validate() error {
	switch m {
	case CostStandard, CostFIFO, CostAverage:
		return nil
	default:
		return platformerrors.Validation("invalid cost method", map[string]string{
			"cost_method": fmt.Sprintf("must be one of [standard, fifo, average], got '%s'", m),
		})
	}
}

// ValuationMode defines when accounting entries for stock moves are generated.
type ValuationMode string

const (
	ValuationRealTime ValuationMode = "real_time" // Perpetual — post entries immediately upon validation
	ValuationPeriodic ValuationMode = "periodic"  // Periodic — post entries at period closing
)

// Validate reports whether the valuation mode is supported.
func (m ValuationMode) Validate() error {
	switch m {
	case ValuationRealTime, ValuationPeriodic:
		return nil
	default:
		return platformerrors.Validation("invalid valuation mode", map[string]string{
			"valuation": fmt.Sprintf("must be one of [real_time, periodic], got '%s'", m),
		})
	}
}

// FIFOEntry is a single layer in the FIFO stack (one per incoming move).
type FIFOEntry struct {
	MoveID int64   `json:"move_id"`
	Qty    float64 `json:"qty"`    // remaining valued quantity available for consumption
	Value  float64 `json:"value"`  // total remaining value of this layer
}

// FIFOStack is an ordered list of incoming layers, oldest first.
type FIFOStack []FIFOEntry

// TotalQty returns the total remaining quantity across all layers.
func (s FIFOStack) TotalQty() float64 {
	var total float64
	for _, e := range s {
		total += e.Qty
	}
	return total
}

// TotalValue returns the total remaining value across all layers.
func (s FIFOStack) TotalValue() float64 {
	var total float64
	for _, e := range s {
		total += e.Value
	}
	return total
}

// UnitPrice returns the weighted average price of the current stack (0 if empty).
func (s FIFOStack) UnitPrice() float64 {
	totalQty := s.TotalQty()
	if totalQty <= 0 {
		return 0
	}
	return s.TotalValue() / totalQty
}

// ProductValue is the primary history record of a manual value update
// (equivalent of product.value in Odoo 19 — the renamed stock.valuation.layer history).
type ProductValue struct {
	ID          int64     `json:"id"`
	ProductID   int64     `json:"product_id"`
	MoveID      *int64    `json:"move_id,omitempty"`
	LotID       *int64    `json:"lot_id,omitempty"`
	Value       float64   `json:"value"`
	CompanyID   int64     `json:"company_id"`
	Date        time.Time `json:"date"`
	UserID      int64     `json:"user_id"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Validate checks ProductValue invariants.
func (pv *ProductValue) Validate() error {
	if pv.ProductID <= 0 {
		return platformerrors.Validation("product is required", map[string]string{
			"product_id": "must reference a valid product",
		})
	}
	if pv.MoveID == nil && pv.LotID == nil {
		return platformerrors.Validation("reference is required", map[string]string{
			"move_id": "a product value record must reference a move or a lot",
		})
	}
	return nil
}

// AccountingPeriod represents a period for periodic (closing) stock valuation.
type AccountingPeriod struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	DateFrom      time.Time `json:"date_from"`
	DateTo        time.Time `json:"date_to"`
	State         string    `json:"state"` // open / closed
	JournalID     int64     `json:"journal_id"`
	AccountMoveID *int64    `json:"account_move_id,omitempty"`
	CompanyID     int64     `json:"company_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Validate checks AccountingPeriod invariants.
func (p *AccountingPeriod) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = fmt.Sprintf("Stock Valuation %s — %s", p.DateFrom.Format("2006-01-02"), p.DateTo.Format("2006-01-02"))
	}
	if p.DateFrom.IsZero() || p.DateTo.IsZero() {
		return platformerrors.Validation("period dates are required", map[string]string{
			"date_from": "cannot be empty",
			"date_to":   "cannot be empty",
		})
	}
	if !p.DateTo.After(p.DateFrom) {
		return platformerrors.Validation("invalid period range", map[string]string{
			"date_to": "must be after date_from",
		})
	}
	if p.JournalID <= 0 {
		return platformerrors.Validation("journal is required", map[string]string{
			"journal_id": "must reference a valid journal",
		})
	}
	if p.State == "" {
		p.State = "open"
	}
	switch p.State {
	case "open", "closed":
	default:
		return platformerrors.Validation("invalid period state", map[string]string{
			"state": fmt.Sprintf("must be one of [open, closed], got '%s'", p.State),
		})
	}
	return nil
}

// ProductValuationConfig is the fully-resolved valuation settings used by the engine
// (resolved from product → category → company).
type ProductValuationConfig struct {
	CostMethod               CostMethod
	Valuation                ValuationMode
	LotValuated              bool
	StockValuationAccountID  *int64
	PriceDifferenceAccountID *int64
	StockJournalID           *int64
}

// ValuationSummary is a snapshot of stock valuation for reports and endpoint responses.
type ValuationSummary struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	LocationID  *int64  `json:"location_id,omitempty"`
	Location    string  `json:"location,omitempty"`
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	Value       float64 `json:"value"`
}