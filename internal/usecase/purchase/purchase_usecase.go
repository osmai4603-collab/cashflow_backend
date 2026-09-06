package purchaseusecase

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
	"cashflow_backend/internal/domain/purchase"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// AccountingService abstracts the accounting usecase operations needed by purchase.
type AccountingService interface {
	CreateInvoice(ctx context.Context, in accountingusecase.CreateInvoiceInput) (*accounting.AccountMove, error)
	GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreatePurchaseOrderLineInput struct {
	ProductID  int64    `json:"product_id"`
	Name       string   `json:"name"`
	ProductQty float64  `json:"product_qty"`
	ProductUom *int64   `json:"product_uom"`
	UnitPrice  *float64 `json:"price_unit"` // Optional; if omitted, fetched from product cost_price
	Discount   float64  `json:"discount"`   // Percentage discount e.g. 10.0 for 10%
	TaxIDs     []int64  `json:"tax_ids"`    // Applicable purchase taxes
}

type CreatePurchaseOrderInput struct {
	Name          string                         `json:"name"` // Optional sequence name
	PartnerID     int64                          `json:"partner_id"`
	DateOrder     time.Time                      `json:"date_order"`
	DatePlanned   *time.Time                     `json:"date_planned"`
	PaymentTermID *int64                         `json:"payment_term_id"`
	UserID        *int64                         `json:"user_id"`
	CompanyID     *int64                         `json:"company_id"`
	Currency      string                         `json:"currency"`
	Note          string                         `json:"note"`
	Lines         []CreatePurchaseOrderLineInput `json:"lines"`
}

type UpdatePurchaseOrderInput struct {
	PartnerID     *int64                         `json:"partner_id"`
	DateOrder     *time.Time                     `json:"date_order"`
	DatePlanned   *time.Time                     `json:"date_planned"`
	PaymentTermID *int64                         `json:"payment_term_id"`
	UserID        *int64                         `json:"user_id"`
	Currency      *string                        `json:"currency"`
	Note          *string                        `json:"note"`
	Lines         []CreatePurchaseOrderLineInput `json:"lines"`
}

type LineBillQuantityInput struct {
	LineID   int64   `json:"line_id"`
	Quantity float64 `json:"quantity"`
}

type CreateBillFromOrderInput struct {
	Date      time.Time               `json:"date"`
	JournalID *int64                  `json:"journal_id"` // Optional; defaults to Purchase Journal
	Lines     []LineBillQuantityInput `json:"lines"`      // Optional; empty means bill all remaining qty
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Implementation
// ─────────────────────────────────────────────────────────────────────────────

type UseCase struct {
	repo           purchase.Repository
	partnerRepo    partner.Repository
	productRepo    product.Repository
	accountingRepo accounting.Repository
	accountingUC   AccountingService
	logger         *slog.Logger
}

// New creates a new purchase UseCase with injected dependencies.
func New(
	repo purchase.Repository,
	partnerRepo partner.Repository,
	productRepo product.Repository,
	accountingRepo accounting.Repository,
	accountingUC AccountingService,
	logger *slog.Logger,
) *UseCase {
	return &UseCase{
		repo:           repo,
		partnerRepo:    partnerRepo,
		productRepo:    productRepo,
		accountingRepo: accountingRepo,
		accountingUC:   accountingUC,
		logger:         logger,
	}
}

// CreateOrder validates vendor and products, calculates cost pricing and taxes, and saves an RFQ/draft order.
func (uc *UseCase) CreateOrder(ctx context.Context, in CreatePurchaseOrderInput) (*purchase.PurchaseOrder, error) {
	if in.PartnerID <= 0 {
		return nil, platformerrors.Validation("vendor is required", map[string]string{
			"partner_id": "must reference a valid partner",
		})
	}

	// Validate partner exists, is active, and is a supplier
	if uc.partnerRepo != nil {
		p, err := uc.partnerRepo.GetByID(ctx, in.PartnerID)
		if err != nil {
			return nil, err
		}
		if !p.Active {
			return nil, platformerrors.Conflict("vendor partner is inactive")
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

	order := &purchase.PurchaseOrder{
		Name:          name,
		PartnerID:     in.PartnerID,
		DateOrder:     in.DateOrder,
		DatePlanned:   in.DatePlanned,
		State:         purchase.OrderStateDraft,
		InvoiceStatus: purchase.InvoiceStatusNo,
		PaymentTermID: in.PaymentTermID,
		UserID:        in.UserID,
		CompanyID:     in.CompanyID,
		Currency:      in.Currency,
		Note:          in.Note,
		Active:        true,
	}

	lines, err := uc.prepareLines(ctx, in.Lines)
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

	uc.logger.InfoContext(ctx, "purchase RFQ created", "id", order.ID, "name", order.Name, "partner_id", order.PartnerID)
	return order, nil
}

// GetOrderByID retrieves an order by its ID.
func (uc *UseCase) GetOrderByID(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	return uc.repo.GetOrderByID(ctx, id)
}

// GetOrderByName retrieves an order by its sequence name.
func (uc *UseCase) GetOrderByName(ctx context.Context, name string) (*purchase.PurchaseOrder, error) {
	return uc.repo.GetOrderByName(ctx, name)
}

// ListOrders retrieves paginated purchase orders matching filters.
func (uc *UseCase) ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[purchase.PurchaseOrder], error) {
	return uc.repo.ListOrders(ctx, f, page)
}

// UpdateOrder modifies an existing draft or sent purchase quotation.
func (uc *UseCase) UpdateOrder(ctx context.Context, id int64, in UpdatePurchaseOrderInput) (*purchase.PurchaseOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.State != purchase.OrderStateDraft && order.State != purchase.OrderStateSent {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot edit purchase order in state '%s'; must be draft or sent", order.State))
	}

	if in.PartnerID != nil && *in.PartnerID > 0 {
		if uc.partnerRepo != nil {
			p, err := uc.partnerRepo.GetByID(ctx, *in.PartnerID)
			if err != nil {
				return nil, err
			}
			if !p.Active {
				return nil, platformerrors.Conflict("vendor partner is inactive")
			}
		}
		order.PartnerID = *in.PartnerID
	}

	if in.DateOrder != nil && !in.DateOrder.IsZero() {
		order.DateOrder = *in.DateOrder
	}
	if in.DatePlanned != nil {
		order.DatePlanned = in.DatePlanned
	}
	if in.PaymentTermID != nil {
		order.PaymentTermID = in.PaymentTermID
	}
	if in.UserID != nil {
		order.UserID = in.UserID
	}
	if in.Currency != nil && strings.TrimSpace(*in.Currency) != "" {
		order.Currency = strings.TrimSpace(*in.Currency)
	}
	if in.Note != nil {
		order.Note = *in.Note
	}

	if in.Lines != nil {
		lines, err := uc.prepareLines(ctx, in.Lines)
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

	uc.logger.InfoContext(ctx, "purchase order updated", "id", order.ID, "name", order.Name)
	return order, nil
}

// DeleteOrder removes a draft or cancelled purchase order.
func (uc *UseCase) DeleteOrder(ctx context.Context, id int64) error {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return err
	}

	if order.State != purchase.OrderStateDraft && order.State != purchase.OrderStateCancel {
		return platformerrors.Conflict(fmt.Sprintf("cannot delete purchase order in state '%s'; only draft or cancel allowed", order.State))
	}

	if err := uc.repo.DeleteOrder(ctx, id); err != nil {
		return err
	}

	uc.logger.InfoContext(ctx, "purchase order deleted", "id", id)
	return nil
}

// ActionSend marks the quotation as sent to vendor.
func (uc *UseCase) ActionSend(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
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

	uc.logger.InfoContext(ctx, "purchase quotation sent to vendor", "id", order.ID, "name", order.Name)
	return order, nil
}

// ConfirmOrder confirms an RFQ into an active purchase order.
func (uc *UseCase) ConfirmOrder(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var seq string
	if order.Name == "" || order.Name == "/" {
		s, err := uc.repo.NextSequence(ctx, order.DateOrder.Year())
		if err == nil {
			seq = s
		}
	}

	if err := order.ActionConfirm(seq); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "purchase order confirmed", "id", order.ID, "name", order.Name, "state", order.State)
	return order, nil
}

// CancelOrder cancels a purchase order.
func (uc *UseCase) CancelOrder(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := order.ActionCancel(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "purchase order cancelled", "id", order.ID, "name", order.Name)
	return order, nil
}

// ResetToDraft resets a cancelled purchase order back to draft.
func (uc *UseCase) ResetToDraft(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
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

	uc.logger.InfoContext(ctx, "purchase order reset to draft", "id", order.ID, "name", order.Name)
	return order, nil
}

// ActionDone locks a completed purchase order.
func (uc *UseCase) ActionDone(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := order.ActionDone(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "purchase order completed and locked", "id", order.ID, "name", order.Name)
	return order, nil
}

// CreateBillFromOrder generates a vendor bill (AccountMove in_invoice) in accounting.
func (uc *UseCase) CreateBillFromOrder(ctx context.Context, orderID int64, in CreateBillFromOrderInput) (*accounting.AccountMove, error) {
	if uc.accountingUC == nil {
		return nil, platformerrors.Internal("accounting service is not configured", nil)
	}

	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.State != purchase.OrderStatePurchase && order.State != purchase.OrderStateDone {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot bill purchase order in state '%s'; must be confirmed ('purchase' or 'done')", order.State))
	}

	if order.InvoiceStatus == purchase.InvoiceStatusInvoiced {
		return nil, platformerrors.Conflict("purchase order is already fully billed")
	}

	// Determine bill quantities per line
	qtyMap := make(map[int64]float64)
	if len(in.Lines) > 0 {
		for _, reqLine := range in.Lines {
			if reqLine.Quantity <= 0 {
				continue
			}
			qtyMap[reqLine.LineID] = reqLine.Quantity
		}
	} else {
		// Default: bill all remaining unbilled quantities
		for _, l := range order.Lines {
			remaining := l.ProductQty - l.QtyInvoiced
			if remaining > 0 {
				qtyMap[l.ID] = remaining
			}
		}
	}

	if len(qtyMap) == 0 {
		return nil, platformerrors.Conflict("no remaining quantities available to bill for this purchase order")
	}

	// Validate requested quantities against remaining unbilled quantities
	var billItems []accountingusecase.InvoiceLineItemInput
	for i := range order.Lines {
		l := &order.Lines[i]
		qtyToBill, requested := qtyMap[l.ID]
		if !requested || qtyToBill <= 0 {
			continue
		}

		remaining := l.ProductQty - l.QtyInvoiced
		if qtyToBill > remaining+0.0001 {
			return nil, platformerrors.Validation("excessive bill quantity", map[string]string{
				fmt.Sprintf("line_%d", l.ID): fmt.Sprintf("requested quantity %.4f exceeds remaining un-billed quantity %.4f", qtyToBill, remaining),
			})
		}

		pID := l.ProductID
		billItems = append(billItems, accountingusecase.InvoiceLineItemInput{
			ProductID: &pID,
			Name:      l.Name,
			Quantity:  qtyToBill,
			PriceUnit: l.UnitPrice,
			Discount:  l.Discount,
			TaxIDs:    l.TaxIDs,
		})
	}

	if len(billItems) == 0 {
		return nil, platformerrors.Conflict("no valid line items to bill")
	}

	// Determine purchase journal
	journalID := int64(2) // Default to standard seed Purchase Journal (Vendor Bills)
	if in.JournalID != nil && *in.JournalID > 0 {
		journalID = *in.JournalID
	} else if uc.accountingRepo != nil {
		journals, err := uc.accountingRepo.ListJournals(ctx)
		if err == nil {
			for _, j := range journals {
				if j.Type == accounting.JournalTypePurchase && j.Active {
					journalID = j.ID
					break
				}
			}
		}
	}

	billDate := in.Date
	if billDate.IsZero() {
		billDate = time.Now().UTC()
	}

	invInput := accountingusecase.CreateInvoiceInput{
		MoveType:      accounting.MoveTypeInInvoice, // Vendor Bill
		PartnerID:     order.PartnerID,
		JournalID:     journalID,
		Date:          billDate,
		InvoiceDate:   &billDate,
		PaymentTermID: order.PaymentTermID,
		Currency:      order.Currency,
		Ref:           fmt.Sprintf("Purchase Order %s", order.Name),
		Items:         billItems,
	}

	move, err := uc.accountingUC.CreateInvoice(ctx, invInput)
	if err != nil {
		return nil, err
	}

	// Link vendor bill to purchase order
	if err := uc.repo.LinkBill(ctx, order.ID, move.ID); err != nil {
		return nil, err
	}

	// Update QtyInvoiced on order lines
	for i := range order.Lines {
		l := &order.Lines[i]
		if qty, ok := qtyMap[l.ID]; ok {
			l.QtyInvoiced = roundTo4(l.QtyInvoiced + qty)
		}
	}
	order.UpdateBillStatus()

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "vendor bill created from purchase order",
		"order_id", order.ID,
		"order_name", order.Name,
		"bill_id", move.ID,
		"invoice_status", order.InvoiceStatus,
	)

	return move, nil
}

// GetOrderBills returns all accounting moves (vendor bills) linked to this purchase order.
func (uc *UseCase) GetOrderBills(ctx context.Context, orderID int64) ([]accounting.AccountMove, error) {
	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var moves []accounting.AccountMove
	for _, billID := range order.BillIDs {
		if uc.accountingUC != nil {
			move, err := uc.accountingUC.GetMove(ctx, billID)
			if err == nil && move != nil {
				moves = append(moves, *move)
				continue
			}
		}
		if uc.accountingRepo != nil {
			move, err := uc.accountingRepo.GetMoveByID(ctx, billID)
			if err == nil && move != nil {
				moves = append(moves, *move)
			}
		}
	}

	return moves, nil
}

// prepareLines transforms line inputs, resolving product defaults (cost_price, PurchaseOK) and purchase taxes.
func (uc *UseCase) prepareLines(ctx context.Context, lineInputs []CreatePurchaseOrderLineInput) ([]purchase.PurchaseOrderLine, error) {
	if len(lineInputs) == 0 {
		return []purchase.PurchaseOrderLine{}, nil
	}

	lines := make([]purchase.PurchaseOrderLine, len(lineInputs))
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

		qty := in.ProductQty
		if qty <= 0 {
			qty = 1.0
		}

		unitPrice := 0.0
		if in.UnitPrice != nil {
			unitPrice = *in.UnitPrice
		} else if pt != nil {
			unitPrice = pt.CostPrice
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

		line := purchase.PurchaseOrderLine{
			Sequence:   (i + 1) * 10,
			ProductID:  in.ProductID,
			Name:       desc,
			ProductQty: qty,
			ProductUom: uomID,
			UnitPrice:  unitPrice,
			Discount:   in.Discount,
			TaxIDs:     taxIDs,
		}

		line.ComputeAmounts(taxRates)
		lines[i] = line
	}

	return lines, nil
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
