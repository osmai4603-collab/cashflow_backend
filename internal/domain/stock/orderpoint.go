package stock

import (
	"math"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// OrderpointSource is the procurement route used to satisfy a reorder rule
// (Odoo stock_orderpoint source_of_procurement). Only 'buy' is supported for now.
type OrderpointSource string

const (
	OrderpointSourceBuy OrderpointSource = "buy"
)

// OrderpointTrigger controls whether the rule orders automatically or only on demand.
type OrderpointTrigger string

const (
	OrderpointTriggerAuto   OrderpointTrigger = "auto"
	OrderpointTriggerManual OrderpointTrigger = "manual"
)

// Orderpoint is a reorder rule (Odoo stock.warehouse.orderpoint) that watches the
// forecast quantity of a product in a location and raises purchase proposals when
// it falls below the minimum.
type Orderpoint struct {
	ID           int64             `json:"id"`
	Name         string            `json:"name"` // "ROP/2026/00001"
	ProductID    int64             `json:"product_id"`
	WarehouseID  int64             `json:"warehouse_id"`
	LocationID   int64             `json:"location_id"`
	VendorID     *int64            `json:"vendor_id,omitempty"` // preferred supplier when creating the PO
	MinQty       float64           `json:"min_qty"`
	MaxQty       float64           `json:"max_qty"`
	QtyMultiple  float64           `json:"qty_multiple"`
	LeadDays     int               `json:"lead_days"`
	Source       OrderpointSource  `json:"source"`
	Trigger      OrderpointTrigger `json:"trigger"`
	SnoozedUntil *time.Time        `json:"snoozed_until,omitempty"`

	// Computed quantities (mirror Odoo qty_on_hand / qty_forecast).
	QtyOnHand   float64 `json:"qty_on_hand"`
	QtyForecast float64 `json:"qty_forecast"`
	// QtyToOrder is the computed suggestion; QtyToOrderManual is the user override.
	QtyToOrder       float64    `json:"qty_to_order"`
	QtyToOrderManual float64    `json:"qty_to_order_manual"`
	DeadlineDate     *time.Time `json:"deadline_date,omitempty"`

	Active    bool      `json:"active"`
	CompanyID int64     `json:"company_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate enforces reorder rule invariants (Odoo _check_min_max).
func (o *Orderpoint) Validate() error {
	if o.ProductID <= 0 {
		return platformerrors.Validation("orderpoint needs a product", map[string]string{
			"product_id": "must reference a valid product",
		})
	}
	if o.LocationID <= 0 {
		return platformerrors.Validation("orderpoint needs a location", map[string]string{
			"location_id": "must reference a valid location",
		})
	}
	if o.WarehouseID <= 0 {
		return platformerrors.Validation("orderpoint needs a warehouse", map[string]string{
			"warehouse_id": "must reference a valid warehouse",
		})
	}
	if o.MinQty < 0 {
		return platformerrors.Validation("minimum quantity cannot be negative", map[string]string{
			"min_qty": "must be greater than or equal to 0",
		})
	}
	if o.MaxQty > 0 && o.MaxQty < o.MinQty {
		return platformerrors.Validation("maximum quantity must be greater than or equal to the minimum", map[string]string{
			"max_qty": "cannot be less than min_qty",
		})
	}
	if o.QtyMultiple <= 0 {
		o.QtyMultiple = 1
	}
	if o.LeadDays < 0 {
		return platformerrors.Validation("lead days cannot be negative", map[string]string{
			"lead_days": "must be greater than or equal to 0",
		})
	}
	if o.Source == "" {
		o.Source = OrderpointSourceBuy
	}
	if o.Source != OrderpointSourceBuy {
		return platformerrors.Validation("unsupported procurement route", map[string]string{
			"source": "only 'buy' is supported",
		})
	}
	if o.Trigger == "" {
		o.Trigger = OrderpointTriggerManual
	}
	if o.Trigger != OrderpointTriggerAuto && o.Trigger != OrderpointTriggerManual {
		return platformerrors.Validation("invalid trigger", map[string]string{
			"trigger": "must be 'auto' or 'manual'",
		})
	}
	// Snoozing is only meaningful for manual rules (Odoo sla mechanism).
	if o.SnoozedUntil != nil && o.Trigger != OrderpointTriggerManual {
		return platformerrors.Validation("snoozing is only allowed on manual rules", map[string]string{
			"snoozed_until": "cannot be set on an auto rule",
		})
	}
	o.Name = strings.TrimSpace(o.Name)
	if o.QtyToOrderManual < 0 {
		return platformerrors.Validation("manual order quantity cannot be negative", map[string]string{
			"qty_to_order_manual": "must be greater than or equal to 0",
		})
	}
	return nil
}

// IsSnoozed reports whether a manual rule is currently silenced.
func (o *Orderpoint) IsSnoozed(now time.Time) bool {
	return o.Trigger == OrderpointTriggerManual &&
		o.SnoozedUntil != nil && now.Before(*o.SnoozedUntil)
}

// NeedsOrder reports whether the rule currently requires a purchase proposal
// (forecast below minimum and not snoozed).
func (o *Orderpoint) NeedsOrder(now time.Time) bool {
	if o.IsSnoozed(now) {
		return false
	}
	return o.QtyForecast < o.MinQty
}

// EffectiveQtyToOrder returns the user override when set, otherwise the computed suggestion.
func (o *Orderpoint) EffectiveQtyToOrder() float64 {
	if o.QtyToOrderManual > 0 {
		return o.QtyToOrderManual
	}
	return o.QtyToOrder
}

// ClampToMultiple rounds qty up to the ordering multiple (Odoo _round_to_multiple).
func (o *Orderpoint) ClampToMultiple(qty float64) float64 {
	return roundUpToMultiple(qty, o.QtyMultiple)
}

// RoundUpToMultiple rounds qty up to the next multiple (ceil), preserving 4-decimal precision.
func roundUpToMultiple(qty, multiple float64) float64 {
	if qty <= 0 {
		return 0
	}
	if multiple <= 0 {
		return roundTo4(qty)
	}
	return roundTo4(math.Ceil(qty/multiple) * multiple)
}

// ComputeQtyToOrder derives the order suggestion from forecast, min and max
// (Odoo stock_orderpoint _compute_qty_to_order: (max - forecast) rounded up).
func (o *Orderpoint) ComputeQtyToOrder() {
	o.QtyToOrder = 0
	if o.MinQty == 0 && o.MaxQty == 0 {
		return
	}
	target := o.MaxQty
	if target == 0 {
		target = o.MinQty
	}
	shortage := target - o.QtyForecast
	if shortage <= 0 {
		return
	}
	o.QtyToOrder = o.ClampToMultiple(shortage)
}
