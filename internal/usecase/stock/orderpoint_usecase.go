package stockusecase

import (
	"context"
	"math"
	"time"

	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateOrderpointInput struct {
	ProductID        int64                   `json:"product_id"`
	WarehouseID      int64                   `json:"warehouse_id"`
	LocationID       int64                   `json:"location_id"`
	VendorID         *int64                  `json:"vendor_id,omitempty"`
	MinQty           float64                 `json:"min_qty"`
	MaxQty           float64                 `json:"max_qty"`
	QtyMultiple      float64                 `json:"qty_multiple"`
	LeadDays         int                     `json:"lead_days"`
	Trigger          stock.OrderpointTrigger `json:"trigger"`
	SnoozedUntil     *time.Time              `json:"snoozed_until,omitempty"`
	QtyToOrderManual float64                 `json:"qty_to_order_manual"`
	CompanyID        int64                   `json:"company_id"`
}

type UpdateOrderpointInput struct {
	ProductID        *int64                   `json:"product_id,omitempty"`
	WarehouseID      *int64                   `json:"warehouse_id,omitempty"`
	LocationID       *int64                   `json:"location_id,omitempty"`
	VendorID         *int64                   `json:"vendor_id,omitempty"`
	MinQty           *float64                 `json:"min_qty,omitempty"`
	MaxQty           *float64                 `json:"max_qty,omitempty"`
	QtyMultiple      *float64                 `json:"qty_multiple,omitempty"`
	LeadDays         *int                     `json:"lead_days,omitempty"`
	Trigger          *stock.OrderpointTrigger `json:"trigger,omitempty"`
	SnoozedUntil     *time.Time               `json:"snoozed_until,omitempty"`
	ClearSnooze      *bool                    `json:"clear_snooze,omitempty"`
	QtyToOrderManual *float64                 `json:"qty_to_order_manual,omitempty"`
	Active           *bool                    `json:"active,omitempty"`
}

// ReplenishmentItem describes one reorder proposal produced by the worker or the
// suggestion/replenish endpoints (equivalent of an Odoo stock.orderpoint._run line).
type ReplenishmentItem struct {
	Orderpoint  *stock.Orderpoint `json:"orderpoint"`
	ProductName string            `json:"product_name"`
	VendorID    *int64            `json:"vendor_id,omitempty"`
	Qty         float64           `json:"qty"`
	POGenerated bool              `json:"po_generated"`
	POID        int64             `json:"po_id,omitempty"`
	POName      string            `json:"po_name,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Orderpoint CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateOrderpoint(ctx context.Context, in CreateOrderpointInput) (*stock.Orderpoint, error) {
	if in.ProductID <= 0 || in.WarehouseID <= 0 || in.LocationID <= 0 {
		return nil, platformerrors.Validation("orderpoint needs product, warehouse and location", map[string]string{
			"product_id": "required", "warehouse_id": "required", "location_id": "required",
		})
	}
	if uc.productRepo != nil {
		if _, err := uc.productRepo.GetTemplateByID(ctx, in.ProductID); err != nil {
			return nil, err
		}
	}
	if _, err := uc.repo.GetWarehouseByID(ctx, in.WarehouseID); err != nil {
		return nil, err
	}
	if _, err := uc.repo.GetLocationByID(ctx, in.LocationID); err != nil {
		return nil, err
	}
	if in.VendorID != nil && *in.VendorID > 0 && uc.partnerRepo != nil {
		p, err := uc.partnerRepo.GetByID(ctx, *in.VendorID)
		if err != nil {
			return nil, err
		}
		if !p.Active {
			return nil, platformerrors.Conflict("vendor partner is inactive")
		}
	}
	if in.CompanyID <= 0 {
		in.CompanyID = 1
	}

	op := &stock.Orderpoint{
		ProductID:        in.ProductID,
		WarehouseID:      in.WarehouseID,
		LocationID:       in.LocationID,
		VendorID:         in.VendorID,
		MinQty:           in.MinQty,
		MaxQty:           in.MaxQty,
		QtyMultiple:      in.QtyMultiple,
		LeadDays:         in.LeadDays,
		Source:           stock.OrderpointSourceBuy,
		Trigger:          in.Trigger,
		SnoozedUntil:     in.SnoozedUntil,
		QtyToOrderManual: in.QtyToOrderManual,
		CompanyID:        in.CompanyID,
	}
	if op.Trigger == "" {
		op.Trigger = stock.OrderpointTriggerManual
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}

	if err := uc.recreateOrderpoint(ctx, op); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateOrderpoint(ctx, op); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock orderpoint created", "id", op.ID, "name", op.Name, "product_id", op.ProductID)
	return op, nil
}

func (uc *UseCase) GetOrderpoint(ctx context.Context, id int64) (*stock.Orderpoint, error) {
	return uc.repo.GetOrderpointByID(ctx, id)
}

func (uc *UseCase) ListOrderpoints(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.Orderpoint], error) {
	return uc.repo.ListOrderpoints(ctx, f, page)
}

func (uc *UseCase) UpdateOrderpoint(ctx context.Context, id int64, in UpdateOrderpointInput) (*stock.Orderpoint, error) {
	op, err := uc.repo.GetOrderpointByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.ProductID != nil {
		op.ProductID = *in.ProductID
	}
	if in.WarehouseID != nil {
		op.WarehouseID = *in.WarehouseID
	}
	if in.LocationID != nil {
		op.LocationID = *in.LocationID
	}
	if in.VendorID != nil {
		op.VendorID = in.VendorID
	}
	if in.MinQty != nil {
		op.MinQty = *in.MinQty
	}
	if in.MaxQty != nil {
		op.MaxQty = *in.MaxQty
	}
	if in.QtyMultiple != nil {
		op.QtyMultiple = *in.QtyMultiple
	}
	if in.LeadDays != nil {
		op.LeadDays = *in.LeadDays
	}
	if in.Trigger != nil {
		op.Trigger = *in.Trigger
	}
	if in.ClearSnooze != nil && *in.ClearSnooze {
		op.SnoozedUntil = nil
	} else if in.SnoozedUntil != nil {
		op.SnoozedUntil = in.SnoozedUntil
	}
	if in.QtyToOrderManual != nil {
		op.QtyToOrderManual = *in.QtyToOrderManual
	}
	if in.Active != nil {
		op.Active = *in.Active
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	if err := uc.recreateOrderpoint(ctx, op); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateOrderpoint(ctx, op); err != nil {
		return nil, err
	}
	return op, nil
}

func (uc *UseCase) DeleteOrderpoint(ctx context.Context, id int64) error {
	return uc.repo.DeleteOrderpoint(ctx, id)
}

// recreateOrderpoint recomputes the forecast-derived fields of a single rule.
func (uc *UseCase) recreateOrderpoint(ctx context.Context, op *stock.Orderpoint) error {
	now := time.Now().UTC()
	return uc.recomputeOrderpoint(ctx, op, now)
}

// recomputeOrderpoint refreshes qty_on_hand / qty_forecast / qty_to_order / deadline_date
// for one rule (Odoo stock_orderpoint._compute_qty).
func (uc *UseCase) recomputeOrderpoint(ctx context.Context, op *stock.Orderpoint, now time.Time) error {
	horizon := now.AddDate(0, 0, op.LeadDays)
	onHand, incoming, outgoing, err := uc.repo.StockForecast(ctx, op.ProductID, op.LocationID, horizon)
	if err != nil {
		return err
	}
	op.QtyOnHand = round4(onHand)
	op.QtyForecast = round4(onHand + incoming - outgoing)
	op.ComputeQtyToOrder()
	if op.QtyToOrder > 0 {
		deadline := now.AddDate(0, 0, op.LeadDays)
		op.DeadlineDate = &deadline
	} else {
		op.DeadlineDate = nil
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Replenishment
// ─────────────────────────────────────────────────────────────────────────────

// ReplenishmentSuggestions recomputes every active rule and returns only the ones
// that currently need a purchase proposal, without creating any purchase order.
func (uc *UseCase) ReplenishmentSuggestions(ctx context.Context) ([]ReplenishmentItem, error) {
	now := time.Now().UTC()
	page := pagination.PageRequest{Page: 1, Limit: 10000}
	res, err := uc.repo.ListOrderpoints(ctx, filter.NewFilter().Add("active", filter.OpEqual, true), page)
	if err != nil {
		return nil, err
	}

	var items []ReplenishmentItem
	for i := range res.Items {
		op := res.Items[i]
		if err := uc.recomputeOrderpoint(ctx, &op, now); err != nil {
			uc.logger.WarnContext(ctx, "replenishment suggestion recompute failed", "orderpoint_id", op.ID, "error", err)
			continue
		}
		if err := uc.repo.UpdateOrderpoint(ctx, &op); err != nil {
			uc.logger.WarnContext(ctx, "replenishment suggestion persist failed", "orderpoint_id", op.ID, "error", err)
			continue
		}
		if !op.NeedsOrder(now) {
			continue
		}
		qty := op.EffectiveQtyToOrder()
		if qty <= 0 {
			continue
		}
		item := ReplenishmentItem{Orderpoint: &op, Qty: qty, VendorID: op.VendorID}
		if name, err := uc.productName(ctx, op.ProductID); err == nil {
			item.ProductName = name
		}
		items = append(items, item)
	}
	return items, nil
}

// RunReplenishment executes the reorder rules: manual rules with a preferred
// quantity and every auto rule that is below minimum produce purchase proposals
// (merged into existing draft POs of the same vendor) — Odoo stock.orderpoint._run.
func (uc *UseCase) RunReplenishment(ctx context.Context) ([]ReplenishmentItem, error) {
	now := time.Now().UTC()
	if uc.purchaseRepo == nil {
		return nil, platformerrors.Conflict("purchase module is not wired; cannot create purchase orders")
	}

	candidates, err := uc.repo.ListOrderpointsForReplenishment(ctx, now)
	if err != nil {
		return nil, err
	}

	var items []ReplenishmentItem
	for _, op := range candidates {
		if err := uc.recomputeOrderpoint(ctx, op, now); err != nil {
			uc.logger.WarnContext(ctx, "replenishment recompute failed", "orderpoint_id", op.ID, "error", err)
			continue
		}
		if !op.NeedsOrder(now) {
			op.QtyToOrder = 0
			if err := uc.repo.UpdateOrderpoint(ctx, op); err != nil {
				uc.logger.WarnContext(ctx, "replenishment persist failed", "orderpoint_id", op.ID, "error", err)
			}
			continue
		}
		qty := op.EffectiveQtyToOrder()
		if qty <= 0 {
			continue
		}

		item := ReplenishmentItem{Orderpoint: op, Qty: qty, VendorID: op.VendorID}
		if name, err := uc.productName(ctx, op.ProductID); err == nil {
			item.ProductName = name
		}
		if op.VendorID == nil {
			uc.logger.WarnContext(ctx, "replenishment skipped: no vendor on rule", "orderpoint_id", op.ID)
			items = append(items, item)
			continue
		}
		po, err := uc.ensurePurchaseOrder(ctx, op, qty, now)
		if err != nil {
			uc.logger.WarnContext(ctx, "replenishment PO creation failed", "orderpoint_id", op.ID, "error", err)
			items = append(items, item)
			continue
		}
		item.POGenerated = true
		item.POID = po.ID
		item.POName = po.Name
		items = append(items, item)
	}
	return items, nil
}

// ensurePurchaseOrder merges qty into an existing draft PO of the same vendor for the
// same product when possible, otherwise creates a new RFQ.
func (uc *UseCase) ensurePurchaseOrder(ctx context.Context, op *stock.Orderpoint, qty float64, now time.Time) (*purchase.PurchaseOrder, error) {
	vendorID := *op.VendorID

	page := pagination.PageRequest{Page: 1, Limit: 100}
	draftFilter := filter.NewFilter().
		Add("partner_id", filter.OpEqual, vendorID).
		Add("state", filter.OpEqual, string(purchase.OrderStateDraft))
	res, err := uc.purchaseRepo.ListOrders(ctx, draftFilter, page)
	if err == nil && len(res.Items) > 0 {
		order, err := uc.purchaseRepo.GetOrderByID(ctx, res.Items[0].ID)
		if err == nil {
			for i := range order.Lines {
				if order.Lines[i].ProductID == op.ProductID {
					order.Lines[i].ProductQty = round4(order.Lines[i].ProductQty + qty)
					order.Lines[i].ComputeAmounts(nil)
					order.RecomputeTotals()
					if err := uc.purchaseRepo.UpdateOrder(ctx, order); err == nil {
						return order, nil
					}
				}
			}
			productName := ""
			if tmpl, err := uc.productRepo.GetTemplateByID(ctx, op.ProductID); err == nil {
				productName = tmpl.Name
			}
			price := uc.productCost(ctx, op.ProductID)
			line := purchase.PurchaseOrderLine{
				Sequence:    (len(order.Lines) + 1) * 10,
				ProductID:   op.ProductID,
				Name:        productName,
				ProductQty:  qty,
				ProductUom:  uc.productUoMID(ctx, op.ProductID),
				UnitPrice:   price,
				QtyReceived: 0,
				QtyInvoiced: 0,
			}
			line.ComputeAmounts(nil)
			order.Lines = append(order.Lines, line)
			order.RecomputeTotals()
			_ = order.Validate()
			if err := uc.purchaseRepo.UpdateOrder(ctx, order); err != nil {
				return nil, err
			}
			return order, nil
		}
	}

	// No reusable draft PO → create a new RFQ.
	seq, err := uc.purchaseRepo.NextSequence(ctx, now.Year())
	if err != nil {
		return nil, err
	}
	productName := ""
	if tmpl, err := uc.productRepo.GetTemplateByID(ctx, op.ProductID); err == nil {
		productName = tmpl.Name
	}
	deadline := now.AddDate(0, 0, op.LeadDays)
	order := &purchase.PurchaseOrder{
		Name:          seq,
		PartnerID:     vendorID,
		DateOrder:     now,
		DatePlanned:   &deadline,
		State:         purchase.OrderStateDraft,
		InvoiceStatus: purchase.InvoiceStatusNo,
		CompanyID:     &op.CompanyID,
		Currency:      "USD",
		Note:          "Auto-replenishment from reorder rule " + op.Name,
		Active:        true,
	}
	line := purchase.PurchaseOrderLine{
		Sequence:   10,
		ProductID:  op.ProductID,
		Name:       productName,
		ProductQty: qty,
		ProductUom: uc.productUoMID(ctx, op.ProductID),
		UnitPrice:  uc.productCost(ctx, op.ProductID),
	}
	line.ComputeAmounts(nil)
	order.Lines = []purchase.PurchaseOrderLine{line}
	order.RecomputeTotals()
	if err := order.Validate(); err != nil {
		return nil, err
	}
	if err := uc.purchaseRepo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "reorder rule created purchase order", "orderpoint_id", op.ID, "po", order.Name, "qty", qty)
	return order, nil
}

func (uc *UseCase) productName(ctx context.Context, productID int64) (string, error) {
	if uc.productRepo == nil {
		return "", nil
	}
	t, err := uc.productRepo.GetTemplateByID(ctx, productID)
	if err != nil {
		return "", err
	}
	return t.Name, nil
}

func (uc *UseCase) productCost(ctx context.Context, productID int64) float64 {
	if uc.productRepo == nil {
		return 0
	}
	t, err := uc.productRepo.GetTemplateByID(ctx, productID)
	if err != nil {
		return 0
	}
	if t.CostMethod == "average" && t.AvgCost > 0 {
		return t.AvgCost
	}
	return t.CostPrice
}

func (uc *UseCase) productUoMID(ctx context.Context, productID int64) *int64 {
	if uc.productRepo == nil {
		return nil
	}
	t, err := uc.productRepo.GetTemplateByID(ctx, productID)
	if err != nil || t.UoMID == nil {
		return nil
	}
	res := *t.UoMID
	return &res
}

// round4 mirrors the stock domain rounding helper for usecase-level math (HALF_UP).
func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
