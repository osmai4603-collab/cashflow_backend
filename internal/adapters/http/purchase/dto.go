package purchasehttp

import (
	"time"

	"cashflow_backend/internal/domain/purchase"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreatePurchaseOrderLineRequest struct {
	ProductID  int64    `json:"product_id"`
	Name       string   `json:"name"`
	ProductQty float64  `json:"product_qty"`
	ProductUom *int64   `json:"product_uom,omitempty"`
	UnitPrice  *float64 `json:"price_unit,omitempty"`
	Discount   float64  `json:"discount"`
	TaxIDs     []int64  `json:"tax_ids,omitempty"`
}

type CreatePurchaseOrderRequest struct {
	Name          string                           `json:"name,omitempty"`
	PartnerID     int64                            `json:"partner_id"`
	DateOrder     *time.Time                       `json:"date_order,omitempty"`
	DatePlanned   *time.Time                       `json:"date_planned,omitempty"`
	PaymentTermID *int64                           `json:"payment_term_id,omitempty"`
	UserID        *int64                           `json:"user_id,omitempty"`
	CompanyID     *int64                           `json:"company_id,omitempty"`
	Currency      string                           `json:"currency,omitempty"`
	Note          string                           `json:"note,omitempty"`
	Lines         []CreatePurchaseOrderLineRequest `json:"lines"`
}

func (r CreatePurchaseOrderRequest) ToInput() purchaseusecase.CreatePurchaseOrderInput {
	var dateOrder time.Time
	if r.DateOrder != nil {
		dateOrder = *r.DateOrder
	}

	lines := make([]purchaseusecase.CreatePurchaseOrderLineInput, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = purchaseusecase.CreatePurchaseOrderLineInput{
			ProductID:  l.ProductID,
			Name:       l.Name,
			ProductQty: l.ProductQty,
			ProductUom: l.ProductUom,
			UnitPrice:  l.UnitPrice,
			Discount:   l.Discount,
			TaxIDs:     l.TaxIDs,
		}
	}

	return purchaseusecase.CreatePurchaseOrderInput{
		Name:          r.Name,
		PartnerID:     r.PartnerID,
		DateOrder:     dateOrder,
		DatePlanned:   r.DatePlanned,
		PaymentTermID: r.PaymentTermID,
		UserID:        r.UserID,
		CompanyID:     r.CompanyID,
		Currency:      r.Currency,
		Note:          r.Note,
		Lines:         lines,
	}
}

type UpdatePurchaseOrderRequest struct {
	PartnerID     *int64                           `json:"partner_id,omitempty"`
	DateOrder     *time.Time                       `json:"date_order,omitempty"`
	DatePlanned   *time.Time                       `json:"date_planned,omitempty"`
	PaymentTermID *int64                           `json:"payment_term_id,omitempty"`
	UserID        *int64                           `json:"user_id,omitempty"`
	Currency      *string                          `json:"currency,omitempty"`
	Note          *string                          `json:"note,omitempty"`
	Lines         []CreatePurchaseOrderLineRequest `json:"lines,omitempty"`
}

func (r UpdatePurchaseOrderRequest) ToInput() purchaseusecase.UpdatePurchaseOrderInput {
	var lines []purchaseusecase.CreatePurchaseOrderLineInput
	if len(r.Lines) > 0 {
		lines = make([]purchaseusecase.CreatePurchaseOrderLineInput, len(r.Lines))
		for i, l := range r.Lines {
			lines[i] = purchaseusecase.CreatePurchaseOrderLineInput{
				ProductID:  l.ProductID,
				Name:       l.Name,
				ProductQty: l.ProductQty,
				ProductUom: l.ProductUom,
				UnitPrice:  l.UnitPrice,
				Discount:   l.Discount,
				TaxIDs:     l.TaxIDs,
			}
		}
	}

	return purchaseusecase.UpdatePurchaseOrderInput{
		PartnerID:     r.PartnerID,
		DateOrder:     r.DateOrder,
		DatePlanned:   r.DatePlanned,
		PaymentTermID: r.PaymentTermID,
		UserID:        r.UserID,
		Currency:      r.Currency,
		Note:          r.Note,
		Lines:         lines,
	}
}

type LineBillQuantityRequest struct {
	LineID   int64   `json:"line_id"`
	Quantity float64 `json:"quantity"`
}

type CreateBillFromOrderRequest struct {
	Date      *time.Time                `json:"date,omitempty"`
	JournalID *int64                    `json:"journal_id,omitempty"`
	Lines     []LineBillQuantityRequest `json:"lines,omitempty"`
}

func (r CreateBillFromOrderRequest) ToInput() purchaseusecase.CreateBillFromOrderInput {
	var date time.Time
	if r.Date != nil {
		date = *r.Date
	}
	var lines []purchaseusecase.LineBillQuantityInput
	for _, l := range r.Lines {
		lines = append(lines, purchaseusecase.LineBillQuantityInput{
			LineID:   l.LineID,
			Quantity: l.Quantity,
		})
	}
	return purchaseusecase.CreateBillFromOrderInput{
		Date:      date,
		JournalID: r.JournalID,
		Lines:     lines,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Response DTOs
// ─────────────────────────────────────────────────────────────────────────────

type PurchaseOrderLineResponse struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	Sequence      int       `json:"sequence"`
	ProductID     int64     `json:"product_id"`
	Name          string    `json:"name"`
	ProductQty    float64   `json:"product_qty"`
	ProductUom    *int64    `json:"product_uom,omitempty"`
	UnitPrice     float64   `json:"price_unit"`
	Discount      float64   `json:"discount"`
	TaxIDs        []int64   `json:"tax_ids"`
	PriceSubtotal float64   `json:"price_subtotal"`
	PriceTax      float64   `json:"price_tax"`
	PriceTotal    float64   `json:"price_total"`
	QtyReceived   float64   `json:"qty_received"`
	QtyInvoiced   float64   `json:"qty_invoiced"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PurchaseOrderResponse struct {
	ID            int64                       `json:"id"`
	Name          string                      `json:"name"`
	PartnerID     int64                       `json:"partner_id"`
	DateOrder     time.Time                   `json:"date_order"`
	DatePlanned   *time.Time                  `json:"date_planned,omitempty"`
	State         purchase.PurchaseOrderState `json:"state"`
	InvoiceStatus purchase.InvoiceStatus      `json:"invoice_status"`
	PaymentTermID *int64                      `json:"payment_term_id,omitempty"`
	UserID        *int64                      `json:"user_id,omitempty"`
	CompanyID     *int64                      `json:"company_id,omitempty"`
	Currency      string                      `json:"currency"`
	Note          string                      `json:"note,omitempty"`
	AmountUntaxed float64                     `json:"amount_untaxed"`
	AmountTax     float64                     `json:"amount_tax"`
	AmountTotal   float64                     `json:"amount_total"`
	Lines         []PurchaseOrderLineResponse `json:"lines"`
	BillIDs       []int64                     `json:"bill_ids"`
	Active        bool                        `json:"active"`
	CreatedAt     time.Time                   `json:"created_at"`
	UpdatedAt     time.Time                   `json:"updated_at"`
}

func ToPurchaseOrderLineResponse(l purchase.PurchaseOrderLine) PurchaseOrderLineResponse {
	taxIDs := l.TaxIDs
	if taxIDs == nil {
		taxIDs = []int64{}
	}
	return PurchaseOrderLineResponse{
		ID:            l.ID,
		OrderID:       l.OrderID,
		Sequence:      l.Sequence,
		ProductID:     l.ProductID,
		Name:          l.Name,
		ProductQty:    l.ProductQty,
		ProductUom:    l.ProductUom,
		UnitPrice:     l.UnitPrice,
		Discount:      l.Discount,
		TaxIDs:        taxIDs,
		PriceSubtotal: l.PriceSubtotal,
		PriceTax:      l.PriceTax,
		PriceTotal:    l.PriceTotal,
		QtyReceived:   l.QtyReceived,
		QtyInvoiced:   l.QtyInvoiced,
		CreatedAt:     l.CreatedAt,
		UpdatedAt:     l.UpdatedAt,
	}
}

func ToPurchaseOrderResponse(o *purchase.PurchaseOrder) PurchaseOrderResponse {
	lines := make([]PurchaseOrderLineResponse, len(o.Lines))
	for i, l := range o.Lines {
		lines[i] = ToPurchaseOrderLineResponse(l)
	}
	billIDs := o.BillIDs
	if billIDs == nil {
		billIDs = []int64{}
	}

	return PurchaseOrderResponse{
		ID:            o.ID,
		Name:          o.Name,
		PartnerID:     o.PartnerID,
		DateOrder:     o.DateOrder,
		DatePlanned:   o.DatePlanned,
		State:         o.State,
		InvoiceStatus: o.InvoiceStatus,
		PaymentTermID: o.PaymentTermID,
		UserID:        o.UserID,
		CompanyID:     o.CompanyID,
		Currency:      o.Currency,
		Note:          o.Note,
		AmountUntaxed: o.AmountUntaxed,
		AmountTax:     o.AmountTax,
		AmountTotal:   o.AmountTotal,
		Lines:         lines,
		BillIDs:       billIDs,
		Active:        o.Active,
		CreatedAt:     o.Audit.CreatedAt,
		UpdatedAt:     o.Audit.UpdatedAt,
	}
}

func ToPurchaseOrderListResponse(orders []purchase.PurchaseOrder) []PurchaseOrderResponse {
	res := make([]PurchaseOrderResponse, len(orders))
	for i := range orders {
		res[i] = ToPurchaseOrderResponse(&orders[i])
	}
	return res
}
