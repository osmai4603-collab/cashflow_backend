package saleusecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/sale"
	platformerrors "cashflow_backend/internal/platform/errors"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// InvoicePolicy dictates whether products are invoiced based on ordered or delivered quantities.
type InvoicePolicy string

const (
	InvoicePolicyOrder    InvoicePolicy = "order"    // Invoice based on ordered quantities
	InvoicePolicyDelivery InvoicePolicy = "delivery" // Invoice based on delivered quantities
)

// AccountingInvoiceCreator represents the accounting interface required by InvoiceService.
type AccountingInvoiceCreator interface {
	CreateInvoice(ctx context.Context, in accountingusecase.CreateInvoiceInput) (*accounting.AccountMove, error)
}

// InvoiceService creates customer invoices from sales orders.
// Corresponds to Odoo 19 sale.order._create_invoices().
type InvoiceService struct {
	saleRepo      sale.Repository
	accountingSvc AccountingInvoiceCreator
	policy        InvoicePolicy
	logger        *slog.Logger
}

// NewInvoiceService creates a new instance of InvoiceService.
func NewInvoiceService(
	saleRepo sale.Repository,
	accountingSvc AccountingInvoiceCreator,
	logger *slog.Logger,
	policy ...InvoicePolicy,
) *InvoiceService {
	pol := InvoicePolicyOrder
	if len(policy) > 0 && policy[0] != "" {
		pol = policy[0]
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &InvoiceService{
		saleRepo:      saleRepo,
		accountingSvc: accountingSvc,
		policy:        pol,
		logger:        logger,
	}
}

// SetPolicy sets the default invoicing policy for this service.
func (s *InvoiceService) SetPolicy(policy InvoicePolicy) {
	s.policy = policy
}

// CreateInvoiceFromOrder generates a customer invoice (out_invoice) from a confirmed sales order.
//
// Rules:
//  1. Order must be in 'sale' or 'done' state.
//  2. Quantities to invoice are computed based on the policy (Order or Delivery) minus QtyInvoiced.
//  3. At least one line item must have a billable quantity > 0.
//  4. Line QtyInvoiced is incremented.
//  5. Order's InvoiceStatus is recomputed and invoice is linked via LinkInvoice.
func (s *InvoiceService) CreateInvoiceFromOrder(ctx context.Context, orderID int64) (*accounting.AccountMove, error) {
	order, err := s.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.State != sale.OrderStateSale && order.State != sale.OrderStateDone {
		return nil, platformerrors.Conflict(
			fmt.Sprintf("cannot invoice order '%s' in state '%s'; order must be confirmed ('sale') or done", order.Name, order.State),
		)
	}

	type lineBillPlan struct {
		lineIndex    int
		qtyToInvoice float64
	}

	var plans []lineBillPlan
	var invoiceItems []accountingusecase.InvoiceLineItemInput

	for i, line := range order.Lines {
		var targetQty float64
		if s.policy == InvoicePolicyDelivery {
			targetQty = line.QtyDelivered
		} else {
			targetQty = line.ProductUomQty
		}

		qty := targetQty - line.QtyInvoiced
		if qty <= 0 {
			continue
		}

		plans = append(plans, lineBillPlan{
			lineIndex:    i,
			qtyToInvoice: qty,
		})

		lineName := string(line.Name)
		if lineName == "" {
			lineName = fmt.Sprintf("Product #%d", line.ProductID)
		}

		prodID := line.ProductID
		invoiceItems = append(invoiceItems, accountingusecase.InvoiceLineItemInput{
			ProductID: &prodID,
			Name:      lineName,
			Quantity:  qty,
			PriceUnit: line.UnitPrice,
			Discount:  line.Discount,
			TaxIDs:    line.TaxIDs,
		})
	}

	if len(invoiceItems) == 0 {
		return nil, platformerrors.Conflict(
			fmt.Sprintf("there is nothing to invoice for order '%s'", order.Name),
		)
	}

	now := time.Now().UTC()
	currency := order.Currency
	if currency == "" {
		currency = "USD"
	}

	journalID := int64(1) // Default Customer Invoices journal

	invoice, err := s.accountingSvc.CreateInvoice(ctx, accountingusecase.CreateInvoiceInput{
		MoveType:      accounting.MoveTypeOutInvoice,
		PartnerID:     order.PartnerID,
		JournalID:     journalID,
		Date:          now,
		InvoiceDate:   &now,
		PaymentTermID: order.PaymentTermID,
		Currency:      currency,
		Ref:           order.Name,
		Items:         invoiceItems,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create invoice for order", "order_id", order.ID, "error", err)
		return nil, err
	}

	// Update order line invoiced quantities
	for _, p := range plans {
		order.Lines[p.lineIndex].QtyInvoiced += p.qtyToInvoice
	}

	order.UpdateInvoiceStatus()
	order.InvoiceIDs = append(order.InvoiceIDs, invoice.ID)

	if err := s.saleRepo.UpdateOrder(ctx, order); err != nil {
		s.logger.ErrorContext(ctx, "failed to update order after invoice creation", "order_id", order.ID, "error", err)
		return nil, err
	}

	if err := s.saleRepo.LinkInvoice(ctx, order.ID, invoice.ID); err != nil {
		s.logger.WarnContext(ctx, "failed to link invoice to order", "order_id", order.ID, "invoice_id", invoice.ID, "error", err)
	}

	s.logger.InfoContext(ctx, "customer invoice created from sales order",
		"order_id", order.ID,
		"order_name", order.Name,
		"invoice_id", invoice.ID,
		"invoice_name", invoice.Name,
		"lines_count", len(invoiceItems),
	)

	return invoice, nil
}
