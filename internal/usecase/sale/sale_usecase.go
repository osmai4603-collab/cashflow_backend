package saleusecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// AccountingService abstracts the accounting usecase operations needed by sales.
type AccountingService interface {
	CreateInvoice(ctx context.Context, in accountingusecase.CreateInvoiceInput) (*accounting.AccountMove, error)
	GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
}

// LoyaltyService abstracts the loyalty & rewards integration needed by sales.
// It is defined here (not importing the loyalty usecase) to keep the dependency
// direction acyclic: sale → (interface) loyalty → sale.
type LoyaltyService interface {
	SettleOrder(ctx context.Context, order *sale.SaleOrder) error
	ReverseOrder(ctx context.Context, order *sale.SaleOrder) error
	ReleaseOrder(ctx context.Context, order *sale.SaleOrder) error
}

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateSaleOrderLineInput struct {
	ProductID     int64    `json:"product_id"`
	Name          string   `json:"name"`
	ProductUomQty float64  `json:"product_uom_qty"`
	ProductUom    *int64   `json:"product_uom"`
	UnitPrice     *float64 `json:"price_unit"` // Optional; if omitted, fetched from product/pricelist
	Discount      float64  `json:"discount"`   // Percentage discount e.g. 10.0 for 10%
	TaxIDs        []int64  `json:"tax_ids"`    // Applicable taxes
}

type CreateSaleOrderInput struct {
	Name          string                     `json:"name"` // Optional sequence name
	PartnerID     int64                      `json:"partner_id"`
	DateOrder     time.Time                  `json:"date_order"`
	ValidityDate  *time.Time                 `json:"validity_date"`
	PricelistID   *int64                     `json:"pricelist_id"`
	PaymentTermID *int64                     `json:"payment_term_id"`
	UserID        *int64                     `json:"user_id"`
	CompanyID     *int64                     `json:"company_id"`
	Currency      string                     `json:"currency"`
	Note          string                     `json:"note"`
	Lines         []CreateSaleOrderLineInput `json:"lines"`
}

type UpdateSaleOrderInput struct {
	PartnerID     *int64                     `json:"partner_id"`
	DateOrder     *time.Time                 `json:"date_order"`
	ValidityDate  *time.Time                 `json:"validity_date"`
	PricelistID   *int64                     `json:"pricelist_id"`
	PaymentTermID *int64                     `json:"payment_term_id"`
	UserID        *int64                     `json:"user_id"`
	Currency      *string                    `json:"currency"`
	Note          *string                    `json:"note"`
	Lines         []CreateSaleOrderLineInput `json:"lines"`
}

type LineInvoiceQuantityInput struct {
	LineID   int64   `json:"line_id"`
	Quantity float64 `json:"quantity"`
}

type CreateInvoiceFromOrderInput struct {
	Date      time.Time                  `json:"date"`
	JournalID *int64                     `json:"journal_id"` // Optional; defaults to Sales Journal
	Lines     []LineInvoiceQuantityInput `json:"lines"`      // Optional; empty means invoice all remaining qty
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Implementation
// ─────────────────────────────────────────────────────────────────────────────

type UseCase struct {
	repo           sale.Repository
	partnerRepo    partner.Repository
	productRepo    product.Repository
	accountingRepo accounting.Repository
	accountingUC   AccountingService
	stockRepo      stock.Repository
	stockUC        *stockusecase.UseCase
	loyaltyUC      LoyaltyService
	chatterUC      *activityusecase.UseCase
	logger         *slog.Logger
}

// New creates a new sales UseCase with injected dependencies.
func New(
	repo sale.Repository,
	partnerRepo partner.Repository,
	productRepo product.Repository,
	accountingRepo accounting.Repository,
	accountingUC AccountingService,
	logger *slog.Logger,
	optional ...any,
) *UseCase {
	var stockRepo stock.Repository
	var stockUC *stockusecase.UseCase
	var loyaltyUC LoyaltyService
	var chatterUC *activityusecase.UseCase
	for _, opt := range optional {
		switch v := opt.(type) {
		case stock.Repository:
			stockRepo = v
		case *stockusecase.UseCase:
			stockUC = v
		case LoyaltyService:
			loyaltyUC = v
		case *activityusecase.UseCase:
			chatterUC = v
		}
	}

	return &UseCase{
		repo:           repo,
		partnerRepo:    partnerRepo,
		productRepo:    productRepo,
		accountingRepo: accountingRepo,
		accountingUC:   accountingUC,
		stockRepo:      stockRepo,
		stockUC:        stockUC,
		loyaltyUC:      loyaltyUC,
		chatterUC:      chatterUC,
		logger:         logger,
	}
}

// CreateOrder validates customer and products, calculates pricing and taxes, and saves a quotation.
func (uc *UseCase) CreateOrder(ctx context.Context, in CreateSaleOrderInput) (*sale.SaleOrder, error) {
	if in.PartnerID <= 0 {
		return nil, platformerrors.Validation("customer is required", map[string]string{
			"partner_id": "must reference a valid partner",
		})
	}

	// Validate partner exists and is active
	if uc.partnerRepo != nil {
		p, err := uc.partnerRepo.GetByID(ctx, in.PartnerID)
		if err != nil {
			return nil, err
		}
		if !p.Active {
			return nil, platformerrors.Conflict("partner is inactive")
		}
	}

	if in.DateOrder.IsZero() {
		in.DateOrder = time.Now().UTC()
	}
	if in.Currency == "" {
		in.Currency = "USD"
	}

	name := strings.TrimSpace(in.Name)
	if name == "" || name == "/" {
		seq, err := uc.repo.NextSequence(ctx, in.DateOrder.Year())
		if err == nil {
			name = seq
		} else {
			name = "/"
		}
	}

	order := &sale.SaleOrder{
		Name:          name,
		PartnerID:     in.PartnerID,
		DateOrder:     in.DateOrder,
		ValidityDate:  in.ValidityDate,
		State:         sale.OrderStateDraft,
		InvoiceStatus: sale.InvoiceStatusNo,
		PricelistID:   in.PricelistID,
		PaymentTermID: in.PaymentTermID,
		UserID:        in.UserID,
		CompanyID:     in.CompanyID,
		Currency:      in.Currency,
		Note:          in.Note,
		Active:        true,
	}

	lines, err := uc.prepareLines(ctx, in.Lines, in.PricelistID)
	if err != nil {
		return nil, err
	}
	order.Lines = lines
	order.RecomputeTotals()

	if err := order.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	if uc.chatterUC != nil {
		_ = uc.chatterUC.MessagePost(ctx, &activity.Message{
			Body:        "Quotation created",
			MessageType: activity.MessageTypeNotification,
			ResModel:    "sale.order",
			ResID:       &order.ID,
			AuthorID:    order.UserID,
			CompanyID:   *order.CompanyID,
		})
	}

	uc.logger.InfoContext(ctx, "sale quotation created", "id", order.ID, "name", order.Name, "partner_id", order.PartnerID)
	return order, nil
}

// GetOrderByID retrieves an order by its ID.
func (uc *UseCase) GetOrderByID(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	return uc.repo.GetOrderByID(ctx, id)
}

// ListOrders retrieves paginated sale orders matching filters.
func (uc *UseCase) ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[sale.SaleOrder], error) {
	return uc.repo.ListOrders(ctx, f, page)
}

// UpdateOrder updates quotation fields and lines. Only permitted in draft or sent state.
func (uc *UseCase) UpdateOrder(ctx context.Context, id int64, in UpdateSaleOrderInput) (*sale.SaleOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.State != sale.OrderStateDraft && order.State != sale.OrderStateSent {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot edit order in state '%s'; only draft and sent quotations can be modified", order.State))
	}

	oldOrder := *order

	if in.PartnerID != nil && *in.PartnerID > 0 {
		if uc.partnerRepo != nil {
			if _, err := uc.partnerRepo.GetByID(ctx, *in.PartnerID); err != nil {
				return nil, err
			}
		}
		order.PartnerID = *in.PartnerID
	}

	if in.DateOrder != nil && !in.DateOrder.IsZero() {
		order.DateOrder = *in.DateOrder
	}
	if in.ValidityDate != nil {
		order.ValidityDate = in.ValidityDate
	}
	if in.PricelistID != nil {
		order.PricelistID = in.PricelistID
	}
	if in.PaymentTermID != nil {
		order.PaymentTermID = in.PaymentTermID
	}
	if in.UserID != nil {
		order.UserID = in.UserID
	}
	if in.Currency != nil && *in.Currency != "" {
		order.Currency = *in.Currency
	}
	if in.Note != nil {
		order.Note = *in.Note
	}

	if len(in.Lines) > 0 {
		lines, err := uc.prepareLines(ctx, in.Lines, order.PricelistID)
		if err != nil {
			return nil, err
		}
		order.Lines = lines
	}

	order.RecomputeTotals()
	if err := order.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	if uc.chatterUC != nil {
		_ = uc.chatterUC.MessagePost(ctx, &activity.Message{
			Body:        "Order confirmed",
			MessageType: activity.MessageTypeNotification,
			ResModel:    "sale.order",
			ResID:       &order.ID,
			AuthorID:    order.UserID,
			CompanyID:   *order.CompanyID,
		})
	}

	uc.trackUpdate(ctx, &oldOrder, order, order.UserID)

	uc.logger.InfoContext(ctx, "sale order updated", "id", order.ID, "name", order.Name)
	return order, nil
}

// DeleteOrder deletes a quotation in draft or cancelled state.
func (uc *UseCase) DeleteOrder(ctx context.Context, id int64) error {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return err
	}

	if order.State != sale.OrderStateDraft && order.State != sale.OrderStateCancel {
		return platformerrors.Conflict(fmt.Sprintf("cannot delete order in state '%s'; must be draft or cancelled", order.State))
	}

	return uc.repo.DeleteOrder(ctx, id)
}

// ActionSend marks a quotation as sent to the customer.
func (uc *UseCase) ActionSend(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := order.ActionSend(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	if uc.chatterUC != nil {
		_ = uc.chatterUC.MessagePost(ctx, &activity.Message{
			Body:        "Quotation sent",
			MessageType: activity.MessageTypeNotification,
			ResModel:    "sale.order",
			ResID:       &order.ID,
			AuthorID:    order.UserID,
			CompanyID:   *order.CompanyID,
		})
	}

	uc.logger.InfoContext(ctx, "sale quotation sent", "id", order.ID, "name", order.Name)
	return order, nil
}

// ConfirmOrder confirms a quotation into a sales order.
func (uc *UseCase) ConfirmOrder(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var seq string
	if order.Name == "" || order.Name == "/" {
		newSeq, err := uc.repo.NextSequence(ctx, order.DateOrder.Year())
		if err != nil {
			return nil, err
		}
		seq = newSeq
	}

	if err := order.ActionConfirm(seq); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	if uc.chatterUC != nil {
		_ = uc.chatterUC.MessagePost(ctx, &activity.Message{
			Body:        "Order confirmed",
			MessageType: activity.MessageTypeNotification,
			ResModel:    "sale.order",
			ResID:       &order.ID,
			AuthorID:    order.UserID,
			CompanyID:   *order.CompanyID,
		})
	}

	// Phase 14: Sale-Stock Integration
	if uc.stockUC != nil && uc.stockRepo != nil {
		saleStock := NewSaleStockUseCase(uc.repo, uc.stockRepo, uc.stockUC, uc.logger)
		if err := saleStock.CreateDeliveriesFromOrder(ctx, order); err != nil {
			uc.logger.ErrorContext(ctx, "failed to create stock deliveries for sale order", "order_id", order.ID, "error", err)
			// We don't necessarily want to fail the whole confirmation if stock creation fails,
			// but in a strict system we might. For Phase 14, let's keep it robust.
			return nil, fmt.Errorf("order confirmed but stock integration failed: %w", err)
		}
	}

	// Phase 23: Loyalty & Rewards — settle coupons and add earned points.
	if uc.loyaltyUC != nil {
		if err := uc.loyaltyUC.SettleOrder(ctx, order); err != nil {
			uc.logger.ErrorContext(ctx, "failed to settle loyalty for sale order", "order_id", order.ID, "error", err)
			return nil, fmt.Errorf("order confirmed but loyalty settlement failed: %w", err)
		}
	}

	uc.logger.InfoContext(ctx, "sale order confirmed", "id", order.ID, "name", order.Name)
	return order, nil
}

// CancelOrder cancels a sale order or quotation.
func (uc *UseCase) CancelOrder(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	preState := order.State
	if err := order.ActionCancel(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	if uc.chatterUC != nil {
		_ = uc.chatterUC.MessagePost(ctx, &activity.Message{
			Body:        "Order cancelled",
			MessageType: activity.MessageTypeNotification,
			ResModel:    "sale.order",
			ResID:       &order.ID,
			AuthorID:    order.UserID,
			CompanyID:   *order.CompanyID,
		})
	}

	// Phase 23: Loyalty & Rewards — settle confirmed orders, reverse confirmed ones.
	if uc.loyaltyUC != nil {
		var err error
		if preState == sale.OrderStateSale || preState == sale.OrderStateDone {
			err = uc.loyaltyUC.ReverseOrder(ctx, order)
		} else {
			err = uc.loyaltyUC.ReleaseOrder(ctx, order)
		}
		if err != nil {
			uc.logger.ErrorContext(ctx, "failed to reverse loyalty for sale order", "order_id", order.ID, "error", err)
			return nil, fmt.Errorf("order cancelled but loyalty reversal failed: %w", err)
		}
	}

	uc.logger.InfoContext(ctx, "sale order cancelled", "id", order.ID, "name", order.Name)
	return order, nil
}

// ResetToDraft resets a cancelled order back to draft status for revision.
func (uc *UseCase) ResetToDraft(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := order.ActionDraft(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	if uc.chatterUC != nil {
		_ = uc.chatterUC.MessagePost(ctx, &activity.Message{
			Body:        "Order reset to draft",
			MessageType: activity.MessageTypeNotification,
			ResModel:    "sale.order",
			ResID:       &order.ID,
			AuthorID:    order.UserID,
			CompanyID:   *order.CompanyID,
		})
	}

	uc.logger.InfoContext(ctx, "sale order reset to draft", "id", order.ID, "name", order.Name)
	return order, nil
}

// CreateInvoiceFromOrder generates a customer invoice (AccountMove out_invoice) in accounting.
func (uc *UseCase) CreateInvoiceFromOrder(ctx context.Context, orderID int64, in CreateInvoiceFromOrderInput) (*accounting.AccountMove, error) {
	if uc.accountingUC == nil {
		return nil, platformerrors.Internal("accounting service is not configured", nil)
	}

	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.State != sale.OrderStateSale && order.State != sale.OrderStateDone {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot invoice order in state '%s'; must be confirmed ('sale' or 'done')", order.State))
	}

	if order.InvoiceStatus == sale.InvoiceStatusInvoiced {
		return nil, platformerrors.Conflict("sale order is already fully invoiced")
	}

	// Determine invoice quantities per line
	qtyMap := make(map[int64]float64)
	if len(in.Lines) > 0 {
		for _, reqLine := range in.Lines {
			if reqLine.Quantity <= 0 {
				continue
			}
			qtyMap[reqLine.LineID] = reqLine.Quantity
		}
	} else {
		// Default: invoice all remaining quantities
		for _, l := range order.Lines {
			remaining := l.ProductUomQty - l.QtyInvoiced
			if remaining > 0 {
				qtyMap[l.ID] = remaining
			}
		}
	}

	if len(qtyMap) == 0 {
		return nil, platformerrors.Conflict("no remaining quantities available to invoice for this order")
	}

	// Validate requested quantities against remaining un-invoiced quantities
	var invoiceItems []accountingusecase.InvoiceLineItemInput
	for i := range order.Lines {
		l := &order.Lines[i]
		qtyToInvoice, requested := qtyMap[l.ID]
		if !requested || qtyToInvoice <= 0 {
			continue
		}

		remaining := l.ProductUomQty - l.QtyInvoiced
		if qtyToInvoice > remaining+0.0001 {
			return nil, platformerrors.Validation("excessive invoice quantity", map[string]string{
				fmt.Sprintf("line_%d", l.ID): fmt.Sprintf("requested quantity %.4f exceeds remaining un-invoiced quantity %.4f", qtyToInvoice, remaining),
			})
		}

		pID := l.ProductID
		invoiceItems = append(invoiceItems, accountingusecase.InvoiceLineItemInput{
			ProductID: &pID,
			Name:      l.Name,
			Quantity:  qtyToInvoice,
			PriceUnit: l.UnitPrice,
			Discount:  l.Discount,
			TaxIDs:    l.TaxIDs,
			// Phase 12 — Anglo-Saxon COGS for the delivered portion of this line.
			CogsAmount: uc.computeCogs(ctx, l.ProductID, qtyToInvoice, l.QtyDelivered),
		})
	}

	if len(invoiceItems) == 0 {
		return nil, platformerrors.Conflict("no valid line items to invoice")
	}

	// Determine journal
	journalID := int64(1) // Default to standard sale journal
	if in.JournalID != nil && *in.JournalID > 0 {
		journalID = *in.JournalID
	} else if uc.accountingRepo != nil {
		journals, err := uc.accountingRepo.ListJournals(ctx)
		if err == nil {
			for _, j := range journals {
				if j.Type == accounting.JournalTypeSale && j.Active {
					journalID = j.ID
					break
				}
			}
		}
	}

	invoiceDate := in.Date
	if invoiceDate.IsZero() {
		invoiceDate = time.Now().UTC()
	}

	invInput := accountingusecase.CreateInvoiceInput{
		MoveType:      accounting.MoveTypeOutInvoice,
		PartnerID:     order.PartnerID,
		JournalID:     journalID,
		Date:          invoiceDate,
		InvoiceDate:   &invoiceDate,
		PaymentTermID: order.PaymentTermID,
		Currency:      order.Currency,
		Ref:           fmt.Sprintf("Sale Order %s", order.Name),
		Items:         invoiceItems,
	}

	move, err := uc.accountingUC.CreateInvoice(ctx, invInput)
	if err != nil {
		return nil, err
	}

	// Link invoice to order
	if err := uc.repo.LinkInvoice(ctx, order.ID, move.ID); err != nil {
		return nil, err
	}

	// Update QtyInvoiced on order lines
	for i := range order.Lines {
		l := &order.Lines[i]
		if qty, ok := qtyMap[l.ID]; ok {
			l.QtyInvoiced = roundTo4(l.QtyInvoiced + qty)
		}
	}
	order.UpdateInvoiceStatus()

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	if uc.chatterUC != nil {
		_ = uc.chatterUC.MessagePost(ctx, &activity.Message{
			Body:        "Order confirmed",
			MessageType: activity.MessageTypeNotification,
			ResModel:    "sale.order",
			ResID:       &order.ID,
			AuthorID:    order.UserID,
			CompanyID:   *order.CompanyID,
		})
	}

	uc.logger.InfoContext(ctx, "customer invoice created from sale order",
		"order_id", order.ID,
		"order_name", order.Name,
		"invoice_id", move.ID,
		"invoice_status", order.InvoiceStatus,
	)

	return move, nil
}

// GetOrderInvoices returns all accounting moves (invoices) linked to this sale order.
func (uc *UseCase) GetOrderInvoices(ctx context.Context, orderID int64) ([]accounting.AccountMove, error) {
	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var moves []accounting.AccountMove
	for _, moveID := range order.InvoiceIDs {
		if uc.accountingUC != nil {
			move, err := uc.accountingUC.GetMove(ctx, moveID)
			if err == nil && move != nil {
				moves = append(moves, *move)
				continue
			}
		}
		if uc.accountingRepo != nil {
			move, err := uc.accountingRepo.GetMoveByID(ctx, moveID)
			if err == nil && move != nil {
				moves = append(moves, *move)
			}
		}
	}

	return moves, nil
}

// prepareLines transforms line inputs, resolving product defaults, pricelist rules, and tax percentages.
func (uc *UseCase) prepareLines(ctx context.Context, lineInputs []CreateSaleOrderLineInput, pricelistID *int64) ([]sale.SaleOrderLine, error) {
	if len(lineInputs) == 0 {
		return []sale.SaleOrderLine{}, nil
	}

	lines := make([]sale.SaleOrderLine, len(lineInputs))
	for i, in := range lineInputs {
		if in.ProductID <= 0 {
			return nil, platformerrors.Validation("product is required", map[string]string{
				fmt.Sprintf("lines[%d].product_id", i): "must reference a valid product",
			})
		}

		var pt *product.ProductTemplate
		if uc.productRepo != nil {
			template, err := uc.productRepo.GetTemplateByID(ctx, in.ProductID)
			if err != nil {
				return nil, err
			}
			pt = template
		}

		desc := strings.TrimSpace(in.Name)
		if desc == "" && pt != nil {
			desc = pt.Name
		}
		if desc == "" {
			desc = fmt.Sprintf("Product #%d", in.ProductID)
		}

		uomID := in.ProductUom
		if uomID == nil && pt != nil {
			uomID = pt.UoMID
		}

		qty := in.ProductUomQty
		if qty <= 0 {
			qty = 1.0
		}

		unitPrice := 0.0
		if in.UnitPrice != nil {
			unitPrice = *in.UnitPrice
		} else if pt != nil {
			unitPrice = pt.SalePrice
			// Apply pricelist if configured
			if pricelistID != nil && *pricelistID > 0 && uc.productRepo != nil {
				items, err := uc.productRepo.GetPricelistItems(ctx, *pricelistID)
				if err == nil {
					for _, item := range items {
						unitPrice = item.CalculatePrice(unitPrice, qty)
						break
					}
				}
			}
		}

		taxIDs := in.TaxIDs
		if taxIDs == nil {
			taxIDs = []int64{}
		}

		// Calculate tax rates for line computation
		var taxRates []float64
		if uc.accountingRepo != nil && len(taxIDs) > 0 {
			for _, tid := range taxIDs {
				tax, err := uc.accountingRepo.GetTaxByID(ctx, tid)
				if err == nil && tax.Type == accounting.TaxTypePercent {
					taxRates = append(taxRates, tax.Amount)
				}
			}
		}

		line := sale.SaleOrderLine{
			Sequence:      (i + 1) * 10,
			ProductID:     in.ProductID,
			Name:          desc,
			ProductUomQty: qty,
			ProductUom:    uomID,
			UnitPrice:     unitPrice,
			Discount:      in.Discount,
			TaxIDs:        taxIDs,
		}

		line.ComputeAmounts(taxRates)
		lines[i] = line
	}

	return lines, nil
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}

// computeCogs returns the cost-of-goods-sold amount for the delivered portion of an
// invoiced line, valued at the product's current cost (avg cost when maintained, else standard).
func (uc *UseCase) computeCogs(ctx context.Context, productID int64, qtyToInvoice, qtyDelivered float64) float64 {
	if uc.productRepo == nil {
		return 0
	}
	delivered := qtyToInvoice
	if qtyDelivered < delivered {
		delivered = qtyDelivered
	}
	if delivered <= 0 {
		return 0
	}
	pt, err := uc.productRepo.GetTemplateByID(ctx, productID)
	if err != nil {
		return 0
	}
	cost := pt.CostPrice
	if pt.AvgCost > 0 {
		cost = pt.AvgCost
	}
	return roundTo4(delivered * cost)
}

func (uc *UseCase) trackUpdate(ctx context.Context, oldOrder, newOrder *sale.SaleOrder, authorID *int64) {
	if uc.chatterUC == nil {
		return
	}
	var tracking []activity.TrackingValue
	if oldOrder.PartnerID != newOrder.PartnerID {
		tracking = append(tracking, activity.TrackingValue{
			Field:        "partner_id",
			FieldName:    "Customer",
			OldValueText: fmt.Sprintf("%d", oldOrder.PartnerID),
			NewValueText: fmt.Sprintf("%d", newOrder.PartnerID),
		})
	}
	if math.Abs(oldOrder.AmountTotal-newOrder.AmountTotal) > 0.0001 {
		tracking = append(tracking, activity.TrackingValue{
			Field:        "amount_total",
			FieldName:    "Total",
			OldValueText: fmt.Sprintf("%.2f", oldOrder.AmountTotal),
			NewValueText: fmt.Sprintf("%.2f", newOrder.AmountTotal),
		})
	}
	if oldOrder.State != newOrder.State {
		tracking = append(tracking, activity.TrackingValue{
			Field:        "state",
			FieldName:    "Status",
			OldValueText: string(oldOrder.State),
			NewValueText: string(newOrder.State),
		})
	}

	if len(tracking) > 0 {
		msg := &activity.Message{
			Body:           "Order updated",
			MessageType:    activity.MessageTypeNotification,
			ResModel:       "sale.order",
			ResID:          &newOrder.ID,
			AuthorID:       authorID,
			TrackingValues: tracking,
			CompanyID:      *newOrder.CompanyID,
		}
		_ = uc.chatterUC.MessagePost(ctx, msg)
	}
}
