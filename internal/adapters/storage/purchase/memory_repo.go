package purchasestorage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe, high-fidelity in-memory implementation of purchase.Repository.
type MemoryRepo struct {
	mu          sync.RWMutex
	orders      map[int64]*purchase.PurchaseOrder
	orderLines  map[int64]*purchase.PurchaseOrderLine
	orderBills  map[int64][]int64
	seqCounter  map[int]int64
	lastOrderID int64
	lastLineID  int64
}

// NewMemoryRepo initializes an empty MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		orders:     make(map[int64]*purchase.PurchaseOrder),
		orderLines: make(map[int64]*purchase.PurchaseOrderLine),
		orderBills: make(map[int64][]int64),
		seqCounter: make(map[int]int64),
	}
}

// CreateOrder persists a new purchase order and its lines atomically.
func (r *MemoryRepo) CreateOrder(ctx context.Context, order *purchase.PurchaseOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if order.CompanyID == nil {
		order.CompanyID = audit.CompanyIDFromContext(ctx)
	}

	// Check name uniqueness if not draft placeholder
	if order.Name != "" && order.Name != "/" {
		for _, o := range r.orders {
			if o.Active && companyMatches(ctx, o.CompanyID) && strings.EqualFold(o.Name, order.Name) {
				return platformerrors.Conflict(fmt.Sprintf("purchase order name '%s' already exists", order.Name))
			}
		}
	}

	r.lastOrderID++
	order.ID = r.lastOrderID
	now := time.Now().UTC()
	order.Audit.CreatedAt = now
	order.Audit.UpdatedAt = now
	order.Active = true

	storedLines := make([]purchase.PurchaseOrderLine, len(order.Lines))
	for i, line := range order.Lines {
		r.lastLineID++
		lineClone := line
		lineClone.ID = r.lastLineID
		lineClone.OrderID = order.ID
		lineClone.CreatedAt = now
		lineClone.UpdatedAt = now

		r.orderLines[lineClone.ID] = &lineClone
		storedLines[i] = lineClone
	}
	order.Lines = storedLines

	orderClone := *order
	r.orders[order.ID] = &orderClone
	return nil
}

// GetOrderByID retrieves an order by its ID along with lines and linked bills.
func (r *MemoryRepo) GetOrderByID(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok || !order.Active || !companyMatches(ctx, order.CompanyID) {
		return nil, platformerrors.NotFound("purchase order not found")
	}

	return r.populateOrder(order), nil
}

// GetOrderByName retrieves an order by its sequence name.
func (r *MemoryRepo) GetOrderByName(ctx context.Context, name string) (*purchase.PurchaseOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, o := range r.orders {
		if o.Active && companyMatches(ctx, o.CompanyID) && strings.EqualFold(o.Name, name) {
			return r.populateOrder(o), nil
		}
	}
	return nil, platformerrors.NotFound("purchase order not found")
}

// UpdateOrder updates the purchase order master record and replaces lines.
func (r *MemoryRepo) UpdateOrder(ctx context.Context, order *purchase.PurchaseOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.orders[order.ID]
	if !ok || !existing.Active || !companyMatches(ctx, existing.CompanyID) {
		return platformerrors.NotFound("purchase order not found")
	}

	now := time.Now().UTC()
	order.Audit.CreatedAt = existing.Audit.CreatedAt
	order.Audit.UpdatedAt = now

	// Remove old lines
	for id, line := range r.orderLines {
		if line.OrderID == order.ID {
			delete(r.orderLines, id)
		}
	}

	// Insert updated lines
	storedLines := make([]purchase.PurchaseOrderLine, len(order.Lines))
	for i, line := range order.Lines {
		if line.ID <= 0 {
			r.lastLineID++
			line.ID = r.lastLineID
		}
		line.OrderID = order.ID
		line.UpdatedAt = now
		if line.CreatedAt.IsZero() {
			line.CreatedAt = now
		}

		lineClone := line
		r.orderLines[line.ID] = &lineClone
		storedLines[i] = lineClone
	}
	order.Lines = storedLines

	orderClone := *order
	r.orders[order.ID] = &orderClone
	return nil
}

// DeleteOrder soft-deletes a draft or cancelled purchase order.
func (r *MemoryRepo) DeleteOrder(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[id]
	if !ok || !order.Active || !companyMatches(ctx, order.CompanyID) {
		return platformerrors.NotFound("purchase order not found")
	}

	if order.State != purchase.OrderStateDraft && order.State != purchase.OrderStateCancel {
		return platformerrors.Conflict(fmt.Sprintf("cannot delete purchase order in state '%s'; only draft or cancel allowed", order.State))
	}

	order.Active = false
	for _, line := range r.orderLines {
		if line.OrderID == id {
			delete(r.orderLines, line.ID)
		}
	}
	return nil
}

// ListOrders retrieves paginated orders matching filters.
func (r *MemoryRepo) ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[purchase.PurchaseOrder], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matching []purchase.PurchaseOrder
	for _, o := range r.orders {
		if !o.Active || !companyMatches(ctx, o.CompanyID) {
			continue
		}

		if f != nil {
			match := true
			for _, crit := range f.Criteria {
				switch crit.Field {
				case "partner_id":
					pidStr := fmt.Sprintf("%d", o.PartnerID)
					if pidStr != crit.Value {
						match = false
					}
				case "state":
					if string(o.State) != crit.Value {
						match = false
					}
				case "invoice_status":
					if string(o.InvoiceStatus) != crit.Value {
						match = false
					}
				case "name":
					if !strings.Contains(strings.ToLower(o.Name), strings.ToLower(fmt.Sprintf("%v", crit.Value))) {
						match = false
					}
				case "requisition_id":
					if o.RequisitionID == nil || fmt.Sprintf("%d", *o.RequisitionID) != fmt.Sprintf("%v", crit.Value) {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}

		matching = append(matching, *r.populateOrder(o))
	}

	// Sort orders descending by ID (newest first)
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].ID > matching[j].ID
	})

	total := int64(len(matching))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]purchase.PurchaseOrder{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(matching[offset:end], total, page), nil
}

func companyMatches(ctx context.Context, companyID *int64) bool {
	current := audit.CompanyIDFromContext(ctx)
	return current == nil || (companyID != nil && *companyID == *current)
}

// NextSequence generates a thread-safe sequence number formatted as "PO/YYYY/NNNNN".
func (r *MemoryRepo) NextSequence(ctx context.Context, year int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seqCounter[year]++
	count := r.seqCounter[year]
	return fmt.Sprintf("PO/%d/%05d", year, count), nil
}

// LinkBill links an accounting move (vendor bill) to a purchase order.
func (r *MemoryRepo) LinkBill(ctx context.Context, orderID int64, moveID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing := r.orderBills[orderID]
	for _, m := range existing {
		if m == moveID {
			return nil // already linked
		}
	}
	r.orderBills[orderID] = append(existing, moveID)
	return nil
}

// GetLinkedBillIDs returns all accounting move IDs (vendor bills) linked to a purchase order.
func (r *MemoryRepo) GetLinkedBillIDs(ctx context.Context, orderID int64) ([]int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bills := r.orderBills[orderID]
	res := make([]int64, len(bills))
	copy(res, bills)
	return res, nil
}

func (r *MemoryRepo) populateOrder(o *purchase.PurchaseOrder) *purchase.PurchaseOrder {
	clone := *o
	var lines []purchase.PurchaseOrderLine
	for _, l := range r.orderLines {
		if l.OrderID == o.ID {
			lines = append(lines, *l)
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].Sequence == lines[j].Sequence {
			return lines[i].ID < lines[j].ID
		}
		return lines[i].Sequence < lines[j].Sequence
	})
	clone.Lines = lines

	bills := r.orderBills[o.ID]
	clone.BillIDs = make([]int64, len(bills))
	copy(clone.BillIDs, bills)

	return &clone
}
