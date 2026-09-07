package stock

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// LandedCostState is the lifecycle state of a landed cost (stock.landed.cost state).
type LandedCostState string

const (
	LandedCostDraft  LandedCostState = "draft"
	LandedCostDone   LandedCostState = "done"
	LandedCostCancel LandedCostState = "cancel"
)

// SplitMethod allocates a cost line across the valuation adjustments
// (Odoo stock.landed.cost.lines split_method).
type SplitMethod string

const (
	SplitEqual      SplitMethod = "equal"
	SplitByQuantity SplitMethod = "by_quantity"
	SplitByCost     SplitMethod = "by_current_cost_price"
	SplitByWeight   SplitMethod = "by_weight"
	SplitByVolume   SplitMethod = "by_volume"
)

// ValidSplitMethods lists every supported allocation strategy.
var ValidSplitMethods = []SplitMethod{SplitEqual, SplitByQuantity, SplitByCost, SplitByWeight, SplitByVolume}

// IsValidSplitMethod reports whether method is a supported allocation strategy.
func IsValidSplitMethod(method SplitMethod) bool {
	for _, m := range ValidSplitMethods {
		if m == method {
			return true
		}
	}
	return false
}

// LandedCost groups extra expenses (freight, customs, insurance...) attached to the
// received stock moves and revalues the inventory (Odoo stock.landed.cost).
type LandedCost struct {
	ID                   int64                 `json:"id"`
	Name                 string                `json:"name"` // "LC/2026/00001"
	Date                 time.Time             `json:"date"`
	State                LandedCostState       `json:"state"`
	PickingIDs           []int64               `json:"picking_ids"`
	CostLines            []LandedCostLine      `json:"cost_lines,omitempty"`
	ValuationAdjustments []ValuationAdjustment `json:"valuation_adjustments,omitempty"`
	Description          string                `json:"description,omitempty"`
	AmountTotal          float64               `json:"amount_total"`
	AccountMoveID        *int64                `json:"account_move_id,omitempty"`
	JournalID            int64                 `json:"journal_id"`
	VendorBillID         *int64                `json:"vendor_bill_id,omitempty"`
	CompanyID            int64                 `json:"company_id"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

// Validate enforces landed cost invariants.
func (c *LandedCost) Validate() error {
	if len(c.PickingIDs) == 0 {
		return platformerrors.Validation("at least one receipt must be selected", map[string]string{
			"picking_ids": "cannot be empty",
		})
	}
	if len(c.CostLines) == 0 {
		return platformerrors.Validation("at least one cost line is required", map[string]string{
			"cost_lines": "cannot be empty",
		})
	}
	if c.Date.IsZero() {
		c.Date = time.Now().UTC()
	}
	if c.State == "" {
		c.State = LandedCostDraft
	}
	if c.State != LandedCostDraft && c.State != LandedCostDone && c.State != LandedCostCancel {
		return platformerrors.Validation("invalid landed cost state", map[string]string{
			"state": "must be 'draft', 'done' or 'cancel'",
		})
	}
	if c.JournalID <= 0 {
		return platformerrors.Validation("a journal is required", map[string]string{
			"journal_id": "must reference a valid journal",
		})
	}
	for i := range c.CostLines {
		c.CostLines[i].Name = strings.TrimSpace(c.CostLines[i].Name)
		if c.CostLines[i].AccountID <= 0 {
			return platformerrors.Validation("every cost line needs an expense account", map[string]string{
				"account_id": "cannot be empty",
			})
		}
		if c.CostLines[i].PriceUnit < 0 {
			return platformerrors.Validation("cost line amount cannot be negative", map[string]string{
				"price_unit": "must be greater than or equal to 0",
			})
		}
		if c.CostLines[i].SplitMethod == "" {
			c.CostLines[i].SplitMethod = SplitEqual
		}
		if !IsValidSplitMethod(c.CostLines[i].SplitMethod) {
			return platformerrors.Validation("invalid split method", map[string]string{
				"split_method": "must be equal, by_quantity, by_current_cost_price, by_weight or by_volume",
			})
		}
	}
	return nil
}

// ComputeTotalAmount recomputes amount_total from the cost lines (Odoo amount_total field).
func (c *LandedCost) ComputeTotalAmount() {
	total := 0.0
	for _, l := range c.CostLines {
		total += l.PriceUnit
	}
	c.AmountTotal = roundTo4(total)
}

// LandedCostLine is a single expense attached to a landed cost (stock.landed.cost.lines).
type LandedCostLine struct {
	ID           int64       `json:"id"`
	LandedCostID int64       `json:"landed_cost_id"`
	Name         string      `json:"name"`
	ProductID    int64       `json:"product_id"`
	AccountID    int64       `json:"account_id"`
	PriceUnit    float64     `json:"price_unit"`
	SplitMethod  SplitMethod `json:"split_method"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// ValuationAdjustment allocates one cost line across a received move
// (Odoo stock.valuation.adjustment.lines).
type ValuationAdjustment struct {
	ID               int64     `json:"id"`
	LandedCostID     int64     `json:"landed_cost_id"`
	CostLineID       int64     `json:"cost_line_id"`
	MoveID           int64     `json:"move_id"`
	ProductID        int64     `json:"product_id"`
	Quantity         float64   `json:"quantity"`
	Weight           float64   `json:"weight"`
	Volume           float64   `json:"volume"`
	FormerCost       float64   `json:"former_cost"`     // unit cost before the landed cost
	AdditionalCost   float64   `json:"additional_cost"` // added unit cost allocated by this line
	FinalCost        float64   `json:"final_cost"`
	MoveRemainingQty float64   `json:"move_remaining_qty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ComputeAdditionalCost derives the extra unit cost for this adjustment and returns it.
func (v *ValuationAdjustment) ComputeAdditionalCost() float64 {
	return v.AdditionalCost
}

// ComputeFinalCost sets final-cost = former + additional (Odoo final_cost field).
func (v *ValuationAdjustment) ComputeFinalCost() {
	v.FinalCost = roundTo4(v.FormerCost + v.AdditionalCost)
}

// ComputeSplitValue allocates priceUnit across one adjustment according to split
// using the global totals of the matched set (Odoo stock_landed_cost.compute_landed_cost).
// When split is "equal" the caller passes totalLines = number of matched adjustments.
func ComputeSplitValue(split SplitMethod, priceUnit, totalQty, totalWeight, totalVolume, totalCost, totalLines, qty, weight, volume, formerCost float64) float64 {
	switch split {
	case SplitByQuantity:
		if totalQty != 0 {
			return (priceUnit / totalQty) * qty
		}
	case SplitByCost:
		if totalCost != 0 {
			return (priceUnit / totalCost) * formerCost
		}
	case SplitByWeight:
		if totalWeight != 0 {
			return (priceUnit / totalWeight) * weight
		}
	case SplitByVolume:
		if totalVolume != 0 {
			return (priceUnit / totalVolume) * volume
		}
	}
	if totalLines != 0 {
		return priceUnit / totalLines
	}
	return 0
}

// LandedCostJournalAmount computes the booked amount for one adjustment
// (Odoo stock_landed_cost.create_account_move: diff = additional_landed_cost * remaining/quantity).
// Anything fully consumed by deliveries (remaining == 0) is not bookable.
func LandedCostJournalAmount(additionalCost, quantity, moveRemainingQty float64) float64 {
	if moveRemainingQty == 0 || quantity == 0 {
		return 0
	}
	return roundTo4(additionalCost * (moveRemainingQty / quantity))
}
