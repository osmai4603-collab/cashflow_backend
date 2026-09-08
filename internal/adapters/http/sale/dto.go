package salehttp

import (
	"time"

	"cashflow_backend/internal/domain/sale"
	saleusecase "cashflow_backend/internal/usecase/sale"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateSaleOrderLineRequest struct {
	ProductID     int64    `json:"product_id"`
	Name          string   `json:"name"`
	ProductUomQty float64  `json:"product_uom_qty"`
	ProductUom    *int64   `json:"product_uom,omitempty"`
	UnitPrice     *float64 `json:"price_unit,omitempty"`
	Discount      float64  `json:"discount"`
	TaxIDs        []int64  `json:"tax_ids,omitempty"`
}

type CreateSaleOrderRequest struct {
	Name          string                       `json:"name,omitempty"`
	PartnerID     int64                        `json:"partner_id"`
	DateOrder     *time.Time                   `json:"date_order,omitempty"`
	ValidityDate  *time.Time                   `json:"validity_date,omitempty"`
	PricelistID   *int64                       `json:"pricelist_id,omitempty"`
	PaymentTermID *int64                       `json:"payment_term_id,omitempty"`
	UserID        *int64                       `json:"user_id,omitempty"`
	CompanyID     *int64                       `json:"company_id,omitempty"`
	Currency      string                       `json:"currency,omitempty"`
	Note          string                       `json:"note,omitempty"`
	Lines         []CreateSaleOrderLineRequest `json:"lines"`
}

func (r CreateSaleOrderRequest) ToInput() saleusecase.CreateSaleOrderInput {
	var dateOrder time.Time
	if r.DateOrder != nil {
		dateOrder = *r.DateOrder
	}

	lines := make([]saleusecase.CreateSaleOrderLineInput, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = saleusecase.CreateSaleOrderLineInput{
			ProductID:     l.ProductID,
			Name:          string(l.Name),
			ProductUomQty: l.ProductUomQty,
			ProductUom:    l.ProductUom,
			UnitPrice:     l.UnitPrice,
			Discount:      l.Discount,
			TaxIDs:        l.TaxIDs,
		}
	}

	return saleusecase.CreateSaleOrderInput{
		Name:          r.Name,
		PartnerID:     r.PartnerID,
		DateOrder:     dateOrder,
		ValidityDate:  r.ValidityDate,
		PricelistID:   r.PricelistID,
		PaymentTermID: r.PaymentTermID,
		UserID:        r.UserID,
		CompanyID:     r.CompanyID,
		Currency:      r.Currency,
		Note:          r.Note,
		Lines:         lines,
	}
}

type UpdateSaleOrderRequest struct {
	PartnerID     *int64                       `json:"partner_id,omitempty"`
	DateOrder     *time.Time                   `json:"date_order,omitempty"`
	ValidityDate  *time.Time                   `json:"validity_date,omitempty"`
	PricelistID   *int64                       `json:"pricelist_id,omitempty"`
	PaymentTermID *int64                       `json:"payment_term_id,omitempty"`
	UserID        *int64                       `json:"user_id,omitempty"`
	Currency      *string                      `json:"currency,omitempty"`
	Note          *string                      `json:"note,omitempty"`
	Lines         []CreateSaleOrderLineRequest `json:"lines,omitempty"`
}

func (r UpdateSaleOrderRequest) ToInput() saleusecase.UpdateSaleOrderInput {
	var lines []saleusecase.CreateSaleOrderLineInput
	if len(r.Lines) > 0 {
		lines = make([]saleusecase.CreateSaleOrderLineInput, len(r.Lines))
		for i, l := range r.Lines {
			lines[i] = saleusecase.CreateSaleOrderLineInput{
				ProductID:     l.ProductID,
				Name:          l.Name,
				ProductUomQty: l.ProductUomQty,
				ProductUom:    l.ProductUom,
				UnitPrice:     l.UnitPrice,
				Discount:      l.Discount,
				TaxIDs:        l.TaxIDs,
			}
		}
	}

	return saleusecase.UpdateSaleOrderInput{
		PartnerID:     r.PartnerID,
		DateOrder:     r.DateOrder,
		ValidityDate:  r.ValidityDate,
		PricelistID:   r.PricelistID,
		PaymentTermID: r.PaymentTermID,
		UserID:        r.UserID,
		Currency:      r.Currency,
		Note:          r.Note,
		Lines:         lines,
	}
}

type LineInvoiceQuantityRequest struct {
	LineID   int64   `json:"line_id"`
	Quantity float64 `json:"quantity"`
}

type CreateInvoiceFromOrderRequest struct {
	Date      *time.Time                   `json:"date,omitempty"`
	JournalID *int64                       `json:"journal_id,omitempty"`
	Lines     []LineInvoiceQuantityRequest `json:"lines,omitempty"`
}

func (r CreateInvoiceFromOrderRequest) ToInput() saleusecase.CreateInvoiceFromOrderInput {
	var date time.Time
	if r.Date != nil {
		date = *r.Date
	}
	var lines []saleusecase.LineInvoiceQuantityInput
	for _, l := range r.Lines {
		lines = append(lines, saleusecase.LineInvoiceQuantityInput{
			LineID:   l.LineID,
			Quantity: l.Quantity,
		})
	}
	return saleusecase.CreateInvoiceFromOrderInput{
		Date:      date,
		JournalID: r.JournalID,
		Lines:     lines,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Response DTOs
// ─────────────────────────────────────────────────────────────────────────────

type SaleOrderLineResponse struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	Sequence      int       `json:"sequence"`
	ProductID     int64     `json:"product_id"`
	Name          string    `json:"name"`
	ProductUomQty float64   `json:"product_uom_qty"`
	ProductUom    *int64    `json:"product_uom,omitempty"`
	UnitPrice     float64   `json:"price_unit"`
	Discount      float64   `json:"discount"`
	TaxIDs        []int64   `json:"tax_ids"`
	PriceSubtotal float64   `json:"price_subtotal"`
	PriceTax      float64   `json:"price_tax"`
	PriceTotal    float64   `json:"price_total"`
	QtyDelivered  float64   `json:"qty_delivered"`
	QtyInvoiced   float64   `json:"qty_invoiced"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SaleOrderResponse struct {
	ID            int64                   `json:"id"`
	Name          string                  `json:"name"`
	PartnerID     int64                   `json:"partner_id"`
	DateOrder     time.Time               `json:"date_order"`
	ValidityDate  *time.Time              `json:"validity_date,omitempty"`
	State         sale.SaleOrderState     `json:"state"`
	InvoiceStatus sale.InvoiceStatus      `json:"invoice_status"`
	PricelistID   *int64                  `json:"pricelist_id,omitempty"`
	PaymentTermID *int64                  `json:"payment_term_id,omitempty"`
	UserID        *int64                  `json:"user_id,omitempty"`
	CompanyID     *int64                  `json:"company_id,omitempty"`
	Currency      string                  `json:"currency"`
	Note          string                  `json:"note,omitempty"`
	AmountUntaxed float64                 `json:"amount_untaxed"`
	AmountTax     float64                 `json:"amount_tax"`
	AmountTotal   float64                 `json:"amount_total"`
	Lines         []SaleOrderLineResponse `json:"lines"`
	InvoiceIDs    []int64                 `json:"invoice_ids"`
	Active        bool                    `json:"active"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
}

func ToSaleOrderResponse(o *sale.SaleOrder) SaleOrderResponse {
	lines := make([]SaleOrderLineResponse, len(o.Lines))
	for i, l := range o.Lines {
		taxIDs := l.TaxIDs
		if taxIDs == nil {
			taxIDs = []int64{}
		}
		lines[i] = SaleOrderLineResponse{
			ID:            l.ID,
			OrderID:       l.OrderID,
			Sequence:      l.Sequence,
			ProductID:     l.ProductID,
			Name:          string(l.Name),
			ProductUomQty: l.ProductUomQty,
			ProductUom:    l.ProductUom,
			UnitPrice:     l.UnitPrice,
			Discount:      l.Discount,
			TaxIDs:        taxIDs,
			PriceSubtotal: l.PriceSubtotal,
			PriceTax:      l.PriceTax,
			PriceTotal:    l.PriceTotal,
			QtyDelivered:  l.QtyDelivered,
			QtyInvoiced:   l.QtyInvoiced,
			CreatedAt:     l.CreatedAt,
			UpdatedAt:     l.UpdatedAt,
		}
	}

	invoiceIDs := o.InvoiceIDs
	if invoiceIDs == nil {
		invoiceIDs = []int64{}
	}

	return SaleOrderResponse{
		ID:            o.ID,
		Name:          o.Name,
		PartnerID:     o.PartnerID,
		DateOrder:     o.DateOrder,
		ValidityDate:  o.ValidityDate,
		State:         o.State,
		InvoiceStatus: o.InvoiceStatus,
		PricelistID:   o.PricelistID,
		PaymentTermID: o.PaymentTermID,
		UserID:        o.UserID,
		CompanyID:     o.CompanyID,
		Currency:      o.Currency,
		Note:          o.Note,
		AmountUntaxed: o.AmountUntaxed,
		AmountTax:     o.AmountTax,
		AmountTotal:   o.AmountTotal,
		Lines:         lines,
		InvoiceIDs:    invoiceIDs,
		Active:        o.Active,
		CreatedAt:     o.Audit.CreatedAt,
		UpdatedAt:     o.Audit.UpdatedAt,
	}
}

func ToSaleOrderListResponse(orders []sale.SaleOrder) []SaleOrderResponse {
	res := make([]SaleOrderResponse, len(orders))
	for i := range orders {
		res[i] = ToSaleOrderResponse(&orders[i])
	}
	return res
}
