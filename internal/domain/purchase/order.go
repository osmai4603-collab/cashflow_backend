package purchase

import (
	"fmt"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// PurchaseOrderState represents the lifecycle status of a purchase document (purchase.order in Odoo).
type PurchaseOrderState string

const (
	OrderStateDraft    PurchaseOrderState = "draft"    // Request for Quotation (طلب عرض أسعار)
	OrderStateSent     PurchaseOrderState = "sent"     // RFQ Sent (تم إرسال طلب السعر للمورد)
	OrderStatePurchase PurchaseOrderState = "purchase" // Purchase Order Confirmed (أمر شراء معتمد)
	OrderStateDone     PurchaseOrderState = "done"     // Locked / Completed (مكتمل / مقفل)
	OrderStateCancel   PurchaseOrderState = "cancel"   // Cancelled (ملغي)
)

// InvoiceStatus indicates whether order quantities have been billed by the vendor.
type InvoiceStatus string

const (
	InvoiceStatusNo        InvoiceStatus = "no"         // Nothing to Bill
	InvoiceStatusToInvoice InvoiceStatus = "to_invoice" // Waiting to be billed
	InvoiceStatusInvoiced  InvoiceStatus = "invoiced"   // Fully Billed
)

// PurchaseOrderLine represents a single product line item in a purchase order (purchase.order.line in Odoo).
type PurchaseOrderLine struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	Sequence      int       `json:"sequence"`
	ProductID     int64     `json:"product_id"`
	Name          string    `json:"name"` // Description
	ProductQty    float64   `json:"product_qty"`
	ProductUom    *int64    `json:"product_uom,omitempty"`
	UnitPrice     float64   `json:"price_unit"` // Unit Cost
	Discount      float64   `json:"discount"`   // Percentage discount e.g. 10.0 for 10%
	TaxIDs        []int64   `json:"tax_ids,omitempty"`
	PriceSubtotal float64   `json:"price_subtotal"` // Untaxed amount after discount
	PriceTax      float64   `json:"price_tax"`      // Total purchase tax amount
	PriceTotal    float64   `json:"price_total"`    // Subtotal + Tax
	QtyReceived   float64   `json:"qty_received"`   // Quantity delivered from vendor
	QtyInvoiced   float64   `json:"qty_invoiced"`   // Quantity billed in vendor bill
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ComputeAmounts calculates the line subtotal, tax amount, and total price.
// taxRates contains the tax percentages (e.g. 15.0 for 15%).
func (l *PurchaseOrderLine) ComputeAmounts(taxRates []float64) {
	if l.ProductQty < 0 {
		l.ProductQty = 0
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
	untaxed := l.ProductQty * l.UnitPrice * discountFactor
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

// PurchaseOrder represents a purchase quotation or confirmed purchase order (purchase.order in Odoo).
type PurchaseOrder struct {
	ID                 int64               `json:"id"`
	Name               string              `json:"name"` // Sequence number e.g. "PO/2026/00001" or "/"
	PartnerID          int64               `json:"partner_id"`
	DateOrder          time.Time           `json:"date_order"`
	DatePlanned        *time.Time          `json:"date_planned,omitempty"`
	State              PurchaseOrderState  `json:"state"`
	InvoiceStatus      InvoiceStatus       `json:"invoice_status"`
	PaymentTermID      *int64              `json:"payment_term_id,omitempty"`
	UserID             *int64              `json:"user_id,omitempty"`
	CompanyID          *int64              `json:"company_id,omitempty"`
	Currency           string              `json:"currency"`
	Note               string              `json:"note,omitempty"`
	AmountUntaxed      float64             `json:"amount_untaxed"`
	AmountTax          float64             `json:"amount_tax"`
	AmountTotal        float64             `json:"amount_total"`
	Lines              []PurchaseOrderLine `json:"lines,omitempty"`
	BillIDs            []int64             `json:"bill_ids,omitempty"`
	PickingIDs         []int64             `json:"picking_ids,omitempty"` // Linked stock pickings (receipts)
	ReceiptStatus      string              `json:"receipt_status"`        // nothing, partial, full
	ProcurementGroupID *int64              `json:"procurement_group_id,omitempty"`
	RequisitionID      *int64              `json:"requisition_id,omitempty"`
	RequisitionType    *string             `json:"requisition_type,omitempty"`
	AlternativeGroupID *int64              `json:"alternative_group_id,omitempty"`
	AlternativePOIDs   []int64             `json:"alternative_po_ids,omitempty"`
	Active             bool                `json:"active"`
	Audit              audit.Fields        `json:"audit"`
}

// RecomputeTotals aggregates amounts from all order lines.
func (o *PurchaseOrder) RecomputeTotals() {
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
func (o *PurchaseOrder) Validate() error {
	if o.PartnerID <= 0 {
		return platformerrors.Validation("vendor is required", map[string]string{
			"partner_id": "must reference a valid partner/vendor",
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
		if o.State == OrderStatePurchase || o.State == OrderStateDone {
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
		if l.ProductQty < 0 || ((o.State == OrderStatePurchase || o.State == OrderStateDone) && l.ProductQty == 0) {
			return platformerrors.Validation("quantity cannot be negative or zero on a confirmed order", map[string]string{
				fmt.Sprintf("lines[%d].product_qty", i): "must be non-negative for an RFQ and greater than 0 when confirmed",
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

// ActionSend transitions the RFQ from draft to sent (quotation request sent to vendor).
func (o *PurchaseOrder) ActionSend() error {
	if o.State != OrderStateDraft {
		return platformerrors.Conflict(fmt.Sprintf("cannot send RFQ from state '%s'; must be draft", o.State))
	}
	o.State = OrderStateSent
	return nil
}

// ActionConfirm transitions RFQ (draft or sent) to confirmed purchase order (purchase).
func (o *PurchaseOrder) ActionConfirm(sequence string) error {
	if o.State != OrderStateDraft && o.State != OrderStateSent {
		return platformerrors.Conflict(fmt.Sprintf("cannot confirm order from state '%s'; must be draft or sent", o.State))
	}

	if len(o.Lines) == 0 {
		return platformerrors.Validation("cannot confirm empty purchase order", map[string]string{
			"lines": "purchase order must have at least one line item to be confirmed",
		})
	}
	for i, line := range o.Lines {
		if line.ProductQty <= 0 {
			return platformerrors.Validation("cannot confirm order with zero quantity", map[string]string{
				fmt.Sprintf("lines[%d].product_qty", i): "must be greater than 0 when confirmed",
			})
		}
	}

	o.State = OrderStatePurchase
	if sequence != "" {
		o.Name = sequence
	}
	o.UpdateBillStatus()
	return nil
}

// ActionCancel transitions order to cancelled state.
func (o *PurchaseOrder) ActionCancel() error {
	if o.State == OrderStateCancel {
		return nil
	}
	if o.State == OrderStateDone {
		return platformerrors.Conflict("cannot cancel completed/locked purchase order")
	}

	// Check if any lines have been billed
	for _, l := range o.Lines {
		if l.QtyInvoiced > 0 {
			return platformerrors.Conflict("cannot cancel purchase order with billed quantities; vendor credit notes must be issued first")
		}
	}

	o.State = OrderStateCancel
	o.InvoiceStatus = InvoiceStatusNo
	return nil
}

// ActionDone locks the purchase order once deliveries and vendor bills are finalized.
func (o *PurchaseOrder) ActionDone() error {
	if o.State != OrderStatePurchase {
		return platformerrors.Conflict(fmt.Sprintf("cannot lock order from state '%s'; must be in purchase state", o.State))
	}
	o.State = OrderStateDone
	return nil
}

// ActionDraft resets a cancelled RFQ back to draft for revision and renegotiation.
func (o *PurchaseOrder) ActionDraft() error {
	if o.State != OrderStateCancel {
		return platformerrors.Conflict(fmt.Sprintf("cannot reset order from state '%s'; must be cancel", o.State))
	}
	o.State = OrderStateDraft
	o.InvoiceStatus = InvoiceStatusNo
	return nil
}

// UpdateBillStatus re-evaluates the invoice status based on line quantities.
func (o *PurchaseOrder) UpdateBillStatus() {
	if o.State != OrderStatePurchase && o.State != OrderStateDone {
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
		if l.QtyInvoiced < l.ProductQty {
			allInvoiced = false
		}
		if l.QtyInvoiced > 0 {
			hasInvoiced = true
		}
	}

	if allInvoiced {
		o.InvoiceStatus = InvoiceStatusInvoiced
	} else if hasInvoiced || o.State == OrderStatePurchase {
		o.InvoiceStatus = InvoiceStatusToInvoice
	} else {
		o.InvoiceStatus = InvoiceStatusNo
	}
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
