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

// RequisitionType is the DTO mirror of the purchase requisition type enum.
type RequisitionType string

const (
	RequisitionBlanketOrder RequisitionType = "blanket_order"
	RequisitionTemplate     RequisitionType = "purchase_template"
)

// RequisitionState is the DTO mirror of the purchase requisition lifecycle.
type RequisitionState string

const (
	RequisitionDraft     RequisitionState = "draft"
	RequisitionConfirmed RequisitionState = "confirmed"
	RequisitionDone      RequisitionState = "done"
	RequisitionCancel    RequisitionState = "cancel"
)

type CreateRequisitionLineRequest struct {
	ProductID    int64      `json:"product_id"`
	ProductQty   float64    `json:"product_qty"`
	ProductUOMID *int64     `json:"product_uom_id,omitempty"`
	PriceUnit    float64    `json:"price_unit"`
	ScheduleDate *time.Time `json:"schedule_date,omitempty"`
	SupplierID   *int64     `json:"supplier_id,omitempty"`
	Description  string     `json:"description,omitempty"`
}

func (r CreateRequisitionLineRequest) ToInput() purchaseusecase.CreateRequisitionLineInput {
	return purchaseusecase.CreateRequisitionLineInput{
		ProductID:    r.ProductID,
		ProductQty:   r.ProductQty,
		ProductUOMID: r.ProductUOMID,
		PriceUnit:    r.PriceUnit,
		ScheduleDate: r.ScheduleDate,
		SupplierID:   r.SupplierID,
		Description:  r.Description,
	}
}

type CreatePurchaseRequisitionRequest struct {
	Name        string                         `json:"name,omitempty"`
	Type        RequisitionType                `json:"type,omitempty"`
	VendorID    *int64                         `json:"vendor_id,omitempty"`
	UserID      int64                          `json:"user_id"`
	DateStart   *time.Time                     `json:"date_start,omitempty"`
	DateEnd     *time.Time                     `json:"date_end,omitempty"`
	CurrencyID  int64                          `json:"currency_id"`
	CompanyID   int64                          `json:"company_id"`
	Description string                         `json:"description,omitempty"`
	Lines       []CreateRequisitionLineRequest `json:"lines"`
}

type UpdatePurchaseRequisitionRequest = CreatePurchaseRequisitionRequest

func (r CreatePurchaseRequisitionRequest) ToInput() purchaseusecase.CreatePurchaseRequisitionInput {
	lines := make([]purchaseusecase.CreateRequisitionLineInput, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = l.ToInput()
	}
	return purchaseusecase.CreatePurchaseRequisitionInput{
		Name:        r.Name,
		Type:        purchase.RequisitionType(r.Type),
		VendorID:    r.VendorID,
		UserID:      r.UserID,
		DateStart:   r.DateStart,
		DateEnd:     r.DateEnd,
		CurrencyID:  r.CurrencyID,
		CompanyID:   r.CompanyID,
		Description: r.Description,
		Lines:       lines,
	}
}

type PurchaseRequisitionLineResponse struct {
	ID                         int64      `json:"id"`
	RequisitionID              int64      `json:"requisition_id"`
	ProductID                  int64      `json:"product_id"`
	ProductQty                 float64    `json:"product_qty"`
	ProductUOMID               *int64     `json:"product_uom_id,omitempty"`
	PriceUnit                  float64    `json:"price_unit"`
	QtyOrdered                 float64    `json:"qty_ordered"`
	ScheduleDate               *time.Time `json:"schedule_date,omitempty"`
	SupplierID                 *int64     `json:"supplier_id,omitempty"`
	SupplierInfoID             *int64     `json:"supplier_info_id,omitempty"`
	ProductDescriptionVariants string     `json:"product_description_variants,omitempty"`
	Description                string     `json:"description,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

type PurchaseRequisitionResponse struct {
	ID               int64                             `json:"id"`
	Name             string                            `json:"name"`
	Active           bool                              `json:"active"`
	Reference        string                            `json:"reference,omitempty"`
	Type             purchase.RequisitionType          `json:"type"`
	VendorID         *int64                            `json:"vendor_id,omitempty"`
	UserID           int64                             `json:"user_id"`
	DateStart        *time.Time                        `json:"date_start,omitempty"`
	DateEnd          *time.Time                        `json:"date_end,omitempty"`
	State            purchase.RequisitionState         `json:"state"`
	CurrencyID       int64                             `json:"currency_id"`
	CompanyID        int64                             `json:"company_id"`
	Description      string                            `json:"description,omitempty"`
	OrderCount       int                               `json:"order_count"`
	PurchaseOrderIDs []int64                           `json:"purchase_order_ids,omitempty"`
	Lines            []PurchaseRequisitionLineResponse `json:"lines"`
	CreatedAt        time.Time                         `json:"created_at"`
	UpdatedAt        time.Time                         `json:"updated_at"`
}

func ToPurchaseRequisitionLineResponse(l purchase.PurchaseRequisitionLine) PurchaseRequisitionLineResponse {
	return PurchaseRequisitionLineResponse{
		ID:                         l.ID,
		RequisitionID:              l.RequisitionID,
		ProductID:                  l.ProductID,
		ProductQty:                 l.ProductQty,
		ProductUOMID:               l.ProductUOMID,
		PriceUnit:                  l.PriceUnit,
		QtyOrdered:                 l.QtyOrdered,
		ScheduleDate:               l.ScheduleDate,
		SupplierID:                 l.SupplierID,
		SupplierInfoID:             l.SupplierInfoID,
		ProductDescriptionVariants: l.ProductDescriptionVariants,
		Description:                l.Description,
		CreatedAt:                  l.CreatedAt,
		UpdatedAt:                  l.UpdatedAt,
	}
}

func ToPurchaseRequisitionResponse(r *purchase.PurchaseRequisition) PurchaseRequisitionResponse {
	lines := make([]PurchaseRequisitionLineResponse, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = ToPurchaseRequisitionLineResponse(l)
	}
	return PurchaseRequisitionResponse{
		ID:               r.ID,
		Name:             r.Name,
		Active:           r.Active,
		Reference:        r.Reference,
		Type:             r.Type,
		VendorID:         r.VendorID,
		UserID:           r.UserID,
		DateStart:        r.DateStart,
		DateEnd:          r.DateEnd,
		State:            r.State,
		CurrencyID:       r.CurrencyID,
		CompanyID:        r.CompanyID,
		Description:      r.Description,
		OrderCount:       r.OrderCount,
		PurchaseOrderIDs: r.PurchaseOrderIDs,
		Lines:            lines,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func ToPurchaseRequisitionListResponse(items []purchase.PurchaseRequisition) []PurchaseRequisitionResponse {
	res := make([]PurchaseRequisitionResponse, len(items))
	for i := range items {
		res[i] = ToPurchaseRequisitionResponse(&items[i])
	}
	return res
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
