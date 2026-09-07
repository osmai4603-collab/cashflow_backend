package sale

import (
	"fmt"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// SaleOrderState represents the lifecycle status of a sales document (sale.order in Odoo).
type SaleOrderState string

const (
	OrderStateDraft  SaleOrderState = "draft"  // Quotation (عرض سعر)
	OrderStateSent   SaleOrderState = "sent"   // Quotation Sent (تم إرسال العرض للعميل)
	OrderStateSale   SaleOrderState = "sale"   // Sales Order Confirmed (أمر بيع معتمد)
	OrderStateDone   SaleOrderState = "done"   // Locked / Completed (مكتمل / مقفل)
	OrderStateCancel SaleOrderState = "cancel" // Cancelled (ملغي)
)

// InvoiceStatus indicates whether order quantities have been billed.
type InvoiceStatus string

const (
	InvoiceStatusNo        InvoiceStatus = "no"         // Nothing to Invoice
	InvoiceStatusToInvoice InvoiceStatus = "to_invoice" // Waiting to be invoiced
	InvoiceStatusInvoiced  InvoiceStatus = "invoiced"   // Fully Invoiced
)

// SaleOrderLine represents a single product line item in a sales order (sale.order.line in Odoo).
type SaleOrderLine struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	Sequence      int       `json:"sequence"`
	ProductID     int64     `json:"product_id"`
	Name          string    `json:"name"` // Description
	ProductUomQty float64   `json:"product_uom_qty"`
	ProductUom    *int64    `json:"product_uom,omitempty"`
	UnitPrice     float64   `json:"price_unit"`
	Discount      float64   `json:"discount"` // Percentage discount e.g. 10.0 for 10%
	TaxIDs        []int64   `json:"tax_ids,omitempty"`
	PriceSubtotal float64   `json:"price_subtotal"` // Untaxed amount after discount
	PriceTax      float64   `json:"price_tax"`      // Total tax amount
	PriceTotal    float64   `json:"price_total"`    // Subtotal + Tax
	QtyDelivered  float64   `json:"qty_delivered"`
	QtyInvoiced   float64   `json:"qty_invoiced"`
	RouteID       *int64    `json:"route_id,omitempty"` // ID of the stock route
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ComputeAmounts calculates the line subtotal, tax amount, and total price.
// taxRates contains the tax percentages (e.g. 15.0 for 15%).
func (l *SaleOrderLine) ComputeAmounts(taxRates []float64) {
	if l.ProductUomQty < 0 {
		l.ProductUomQty = 0
	}
	if l.UnitPrice < 0 {
		l.UnitPrice = 0
	}
	if l.Discount < 0 {
		l.Discount = 0
	} else if l.Discount > 100 {
		l.Discount = 100
	}

	discountFactor := 1.0 - (l.Discount / 100.0)
	untaxed := l.ProductUomQty * l.UnitPrice * discountFactor
	l.PriceSubtotal = roundTo4(untaxed)

	var totalTax float64
	for _, rate := range taxRates {
		if rate > 0 {
			totalTax += untaxed * (rate / 100.0)
		}
	}
	l.PriceTax = roundTo4(totalTax)
	l.PriceTotal = roundTo4(l.PriceSubtotal + l.PriceTax)
}

// SaleOrder represents a sales quotation or confirmed order (sale.order in Odoo).
type SaleOrder struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"` // Sequence number e.g. "SO/2026/00001" or "/"
	PartnerID     int64           `json:"partner_id"`
	DateOrder     time.Time       `json:"date_order"`
	ValidityDate  *time.Time      `json:"validity_date,omitempty"`
	State         SaleOrderState  `json:"state"`
	InvoiceStatus InvoiceStatus   `json:"invoice_status"`
	PricelistID   *int64          `json:"pricelist_id,omitempty"`
	PaymentTermID *int64          `json:"payment_term_id,omitempty"`
	UserID        *int64          `json:"user_id,omitempty"`
	CompanyID     *int64          `json:"company_id,omitempty"`
	Currency      string          `json:"currency"`
	Note          string          `json:"note,omitempty"`
	AmountUntaxed float64         `json:"amount_untaxed"`
	AmountTax     float64         `json:"amount_tax"`
	AmountTotal   float64         `json:"amount_total"`
	Lines         []SaleOrderLine `json:"lines,omitempty"`
	InvoiceIDs    []int64         `json:"invoice_ids,omitempty"`
	PickingIDs    []int64         `json:"picking_ids,omitempty"`      // Linked stock pickings (deliveries)
	DeliveryStatus string         `json:"delivery_status"`             // nothing, partial, full
	ProcurementGroupID *int64      `json:"procurement_group_id,omitempty"`
	Active        bool            `json:"active"`
	Audit         audit.Fields    `json:"audit"`
}

// RecomputeTotals aggregates amounts from all order lines.
func (o *SaleOrder) RecomputeTotals() {
	var untaxed, tax, total float64
	for _, l := range o.Lines {
		untaxed += l.PriceSubtotal
		tax += l.PriceTax
		total += l.PriceTotal
	}
	o.AmountUntaxed = roundTo4(untaxed)
	o.AmountTax = roundTo4(tax)
	o.AmountTotal = roundTo4(total)
}

// Validate verifies domain integrity and required fields.
func (o *SaleOrder) Validate() error {
	if o.PartnerID <= 0 {
		return platformerrors.Validation("customer is required", map[string]string{
			"partner_id": "must reference a valid partner/customer",
		})
	}

	if o.DateOrder.IsZero() {
		return platformerrors.Validation("order date is required", map[string]string{
			"date_order": "cannot be empty",
		})
	}

	if o.Currency == "" {
		o.Currency = "USD"
	}

	if o.State == "" {
		o.State = OrderStateDraft
	}

	if o.InvoiceStatus == "" {
		if o.State == OrderStateSale || o.State == OrderStateDone {
			o.InvoiceStatus = InvoiceStatusToInvoice
		} else {
			o.InvoiceStatus = InvoiceStatusNo
		}
	}

	for i, l := range o.Lines {
		if l.ProductID <= 0 {
			return platformerrors.Validation("product is required for order line", map[string]string{
				fmt.Sprintf("lines[%d].product_id", i): "must reference a valid product",
			})
		}
		if strings.TrimSpace(l.Name) == "" {
			return platformerrors.Validation("description is required for order line", map[string]string{
				fmt.Sprintf("lines[%d].name", i): "cannot be empty",
			})
		}
		if l.ProductUomQty <= 0 {
			return platformerrors.Validation("quantity must be strictly positive", map[string]string{
				fmt.Sprintf("lines[%d].product_uom_qty", i): "must be greater than 0",
			})
		}
		if l.UnitPrice < 0 {
			return platformerrors.Validation("unit price cannot be negative", map[string]string{
				fmt.Sprintf("lines[%d].price_unit", i): "must be non-negative",
			})
		}
		if l.Discount < 0 || l.Discount > 100 {
			return platformerrors.Validation("discount must be between 0 and 100", map[string]string{
				fmt.Sprintf("lines[%d].discount", i): "must be in [0, 100]",
			})
		}
	}

	return nil
}

// ActionSend transitions the order from draft to sent (quotation sent to customer).
func (o *SaleOrder) ActionSend() error {
	if o.State != OrderStateDraft {
		return platformerrors.Conflict(fmt.Sprintf("cannot send quotation from state '%s'; must be draft", o.State))
	}
	o.State = OrderStateSent
	return nil
}

// ActionConfirm transitions quotation (draft or sent) to confirmed sales order (sale).
func (o *SaleOrder) ActionConfirm(sequence string) error {
	if o.State != OrderStateDraft && o.State != OrderStateSent {
		return platformerrors.Conflict(fmt.Sprintf("cannot confirm order from state '%s'; must be draft or sent", o.State))
	}

	if len(o.Lines) == 0 {
		return platformerrors.Validation("cannot confirm empty order", map[string]string{
			"lines": "order must have at least one line item to be confirmed",
		})
	}

	o.State = OrderStateSale
	if sequence != "" {
		o.Name = sequence
	}
	o.UpdateInvoiceStatus()
	return nil
}

// ActionCancel transitions order to cancelled state.
func (o *SaleOrder) ActionCancel() error {
	if o.State == OrderStateCancel {
		return nil
	}
	if o.State == OrderStateDone {
		return platformerrors.Conflict("cannot cancel completed/locked order")
	}

	// Check if any lines have been invoiced
	for _, l := range o.Lines {
		if l.QtyInvoiced > 0 {
			return platformerrors.Conflict("cannot cancel order with invoiced quantities; credit notes must be issued first")
		}
	}

	o.State = OrderStateCancel
	o.InvoiceStatus = InvoiceStatusNo
	return nil
}

// ActionDone locks the order once deliveries and invoicing are finalized.
func (o *SaleOrder) ActionDone() error {
	if o.State != OrderStateSale {
		return platformerrors.Conflict(fmt.Sprintf("cannot lock order from state '%s'; must be in sale state", o.State))
	}
	o.State = OrderStateDone
	return nil
}

// ActionDraft resets a cancelled quotation back to draft for revision.
func (o *SaleOrder) ActionDraft() error {
	if o.State != OrderStateCancel {
		return platformerrors.Conflict(fmt.Sprintf("cannot reset order from state '%s'; must be cancel", o.State))
	}
	o.State = OrderStateDraft
	o.InvoiceStatus = InvoiceStatusNo
	return nil
}

// UpdateInvoiceStatus re-evaluates the invoice status based on line quantities.
func (o *SaleOrder) UpdateInvoiceStatus() {
	if o.State != OrderStateSale && o.State != OrderStateDone {
		o.InvoiceStatus = InvoiceStatusNo
		return
	}

	if len(o.Lines) == 0 {
		o.InvoiceStatus = InvoiceStatusNo
		return
	}

	allInvoiced := true
	hasInvoiced := false

	for _, l := range o.Lines {
		if l.QtyInvoiced < l.ProductUomQty {
			allInvoiced = false
		}
		if l.QtyInvoiced > 0 {
			hasInvoiced = true
		}
	}

	if allInvoiced {
		o.InvoiceStatus = InvoiceStatusInvoiced
	} else if hasInvoiced || o.State == OrderStateSale {
		o.InvoiceStatus = InvoiceStatusToInvoice
	} else {
		o.InvoiceStatus = InvoiceStatusNo
	}
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
