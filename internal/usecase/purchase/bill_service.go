package purchaseusecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/purchase"
	platformerrors "cashflow_backend/internal/platform/errors"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// BillPolicy dictates whether vendor bills are created based on ordered or received quantities.
type BillPolicy string

const (
	BillPolicyReceived BillPolicy = "received" // 3-way match: based on goods received
	BillPolicyOrder    BillPolicy = "order"    // 2-way match: based on ordered quantities
)

// AccountingBillCreator represents the accounting interface required by BillService.
type AccountingBillCreator interface {
	CreateInvoice(ctx context.Context, in accountingusecase.CreateInvoiceInput) (*accounting.AccountMove, error)
}

// BillService generates vendor bills (in_invoice) from purchase orders.
// Corresponds to Odoo 19 purchase.order.action_create_invoice().
type BillService struct {
	purchaseRepo  purchase.Repository
	accountingSvc AccountingBillCreator
	policy        BillPolicy
	logger        *slog.Logger
}

// NewBillService creates a new instance of BillService.
func NewBillService(
	purchaseRepo purchase.Repository,
	accountingSvc AccountingBillCreator,
	logger *slog.Logger,
	policy ...BillPolicy,
) *BillService {
	pol := BillPolicyReceived
	if len(policy) > 0 && policy[0] != "" {
		pol = policy[0]
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &BillService{
		purchaseRepo:  purchaseRepo,
		accountingSvc: accountingSvc,
		policy:        pol,
		logger:        logger,
	}
}

// SetPolicy sets the billing policy.
func (s *BillService) SetPolicy(policy BillPolicy) {
	s.policy = policy
}

// CreateBillFromOrder generates a vendor bill (in_invoice) from a confirmed purchase order.
//
// Rules:
//  1. Order must be in 'purchase' or 'done' state.
//  2. Quantities to bill are computed based on the policy (Received or Ordered) minus QtyInvoiced.
//  3. At least one line item must have a billable quantity > 0.
//  4. Line QtyInvoiced is incremented.
//  5. Order's InvoiceStatus is recomputed and bill is linked via LinkBill.
func (s *BillService) CreateBillFromOrder(ctx context.Context, orderID int64) (*accounting.AccountMove, error) {
	order, err := s.purchaseRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.State != purchase.OrderStatePurchase && order.State != purchase.OrderStateDone {
		return nil, platformerrors.Conflict(
			fmt.Sprintf("cannot bill purchase order '%s' in state '%s'; order must be confirmed ('purchase') or done", order.Name, order.State),
		)
	}

	type lineBillPlan struct {
		lineIndex  int
		qtyToBill  float64
	}

	var plans []lineBillPlan
	var billItems []accountingusecase.InvoiceLineItemInput

	for i, line := range order.Lines {
		var targetQty float64
		if s.policy == BillPolicyReceived {
			targetQty = line.QtyReceived
		} else {
			targetQty = line.ProductQty
		}

		qty := targetQty - line.QtyInvoiced
		if qty <= 0 {
			continue
		}

		plans = append(plans, lineBillPlan{
			lineIndex: i,
			qtyToBill: qty,
		})

		lineName := line.Name
		if lineName == "" {
			lineName = fmt.Sprintf("Product #%d", line.ProductID)
		}

		prodID := line.ProductID
		billItems = append(billItems, accountingusecase.InvoiceLineItemInput{
			ProductID: &prodID,
			Name:      lineName,
			Quantity:  qty,
			PriceUnit: line.UnitPrice,
			Discount:  line.Discount,
			TaxIDs:    line.TaxIDs,
		})
	}

	if len(billItems) == 0 {
		return nil, platformerrors.Conflict(
			fmt.Sprintf("there is nothing to bill for purchase order '%s'", order.Name),
		)
	}

	now := time.Now().UTC()
	currency := order.Currency
	if currency == "" {
		currency = "USD"
	}

	journalID := int64(2) // Default Vendor Bills journal

	bill, err := s.accountingSvc.CreateInvoice(ctx, accountingusecase.CreateInvoiceInput{
		MoveType:      accounting.MoveTypeInInvoice,
		PartnerID:     order.PartnerID,
		JournalID:     journalID,
		Date:          now,
		InvoiceDate:   &now,
		PaymentTermID: order.PaymentTermID,
		Currency:      currency,
		Ref:           order.Name,
		Items:         billItems,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create vendor bill for purchase order", "order_id", order.ID, "error", err)
		return nil, err
	}

	// Update order line invoiced quantities
	for _, p := range plans {
		order.Lines[p.lineIndex].QtyInvoiced += p.qtyToBill
	}

	order.UpdateBillStatus()
	order.BillIDs = append(order.BillIDs, bill.ID)

	if err := s.purchaseRepo.UpdateOrder(ctx, order); err != nil {
		s.logger.ErrorContext(ctx, "failed to update purchase order after bill creation", "order_id", order.ID, "error", err)
		return nil, err
	}

	if err := s.purchaseRepo.LinkBill(ctx, order.ID, bill.ID); err != nil {
		s.logger.WarnContext(ctx, "failed to link vendor bill to purchase order", "order_id", order.ID, "bill_id", bill.ID, "error", err)
	}

	s.logger.InfoContext(ctx, "vendor bill created from purchase order",
		"order_id", order.ID,
		"order_name", order.Name,
		"bill_id", bill.ID,
		"bill_name", bill.Name,
		"lines_count", len(billItems),
	)

	return bill, nil
}
