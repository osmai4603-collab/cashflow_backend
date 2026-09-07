package salestorage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe, high-fidelity in-memory implementation of sale.Repository.
type MemoryRepo struct {
	mu            sync.RWMutex
	orders        map[int64]*sale.SaleOrder
	orderLines    map[int64]*sale.SaleOrderLine
	orderInvoices map[int64][]int64
	seqCounter    map[int]int64

	lastOrderID int64
	lastLineID  int64
}

// NewMemoryRepo initializes an empty MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		orders:        make(map[int64]*sale.SaleOrder),
		orderLines:    make(map[int64]*sale.SaleOrderLine),
		orderInvoices: make(map[int64][]int64),
		seqCounter:    make(map[int]int64),
	}
}

// CreateOrder persists a new sale order and its lines atomically.
func (r *MemoryRepo) CreateOrder(ctx context.Context, order *sale.SaleOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if order.CompanyID == nil {
		order.CompanyID = audit.CompanyIDFromContext(ctx)
	}

	// Check name uniqueness if not draft placeholder
	if order.Name != "" && order.Name != "/" {
		for _, o := range r.orders {
			if o.Active && strings.EqualFold(o.Name, order.Name) {
				return platformerrors.Conflict(fmt.Sprintf("sale order name '%s' already exists", order.Name))
			}
		}
	}

	r.lastOrderID++
	order.ID = r.lastOrderID
	now := time.Now().UTC()
	order.Audit.CreatedAt = now
	order.Audit.UpdatedAt = now
	order.Active = true

	storedLines := make([]sale.SaleOrderLine, len(order.Lines))
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

// GetOrderByID retrieves an order by its ID along with lines and linked invoices.
func (r *MemoryRepo) GetOrderByID(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok || !order.Active || !companyMatches(ctx, order.CompanyID) {
		return nil, platformerrors.NotFound("sale order not found")
	}

	return r.populateOrder(order), nil
}

// GetOrderByName retrieves an order by its sequence name.
func (r *MemoryRepo) GetOrderByName(ctx context.Context, name string) (*sale.SaleOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, o := range r.orders {
		if o.Active && companyMatches(ctx, o.CompanyID) && strings.EqualFold(o.Name, name) {
			return r.populateOrder(o), nil
		}
	}
	return nil, platformerrors.NotFound("sale order not found")
}

// UpdateOrder updates the sale order master record and replaces lines.
func (r *MemoryRepo) UpdateOrder(ctx context.Context, order *sale.SaleOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.orders[order.ID]
	if !ok || !existing.Active || !companyMatches(ctx, existing.CompanyID) {
		return platformerrors.NotFound("sale order not found")
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
	storedLines := make([]sale.SaleOrderLine, len(order.Lines))
	for i, line := range order.Lines {
		lineClone := line
		if lineClone.ID <= 0 {
			r.lastLineID++
			lineClone.ID = r.lastLineID
			lineClone.CreatedAt = now
		}
		lineClone.OrderID = order.ID
		lineClone.UpdatedAt = now

		r.orderLines[lineClone.ID] = &lineClone
		storedLines[i] = lineClone
	}
	order.Lines = storedLines

	orderClone := *order
	r.orders[order.ID] = &orderClone
	return nil
}

// DeleteOrder soft-deletes a sale order.
func (r *MemoryRepo) DeleteOrder(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.orders[id]
	if !ok || !existing.Active || !companyMatches(ctx, existing.CompanyID) {
		return platformerrors.NotFound("sale order not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

// ListOrders returns paginated sale orders matching filters.
func (r *MemoryRepo) ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[sale.SaleOrder], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []sale.SaleOrder
	for _, o := range r.orders {
		if !o.Active || !companyMatches(ctx, o.CompanyID) {
			continue
		}

		match := true
		if f != nil {
			for _, c := range f.Criteria {
				switch c.Field {
				case "partner_id":
					if fmt.Sprintf("%v", o.PartnerID) != fmt.Sprintf("%v", c.Value) {
						match = false
					}
				case "state":
					if !strings.EqualFold(string(o.State), fmt.Sprintf("%v", c.Value)) {
						match = false
					}
				case "invoice_status":
					if !strings.EqualFold(string(o.InvoiceStatus), fmt.Sprintf("%v", c.Value)) {
						match = false
					}
				case "name":
					if !strings.Contains(strings.ToLower(o.Name), strings.ToLower(fmt.Sprintf("%v", c.Value))) {
						match = false
					}
				}
			}
		}

		if match {
			result = append(result, *r.populateOrder(o))
		}
	}

	// Sort descending by ID
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	var paginated []sale.SaleOrder
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		paginated = result[offset:end]
	} else {
		paginated = []sale.SaleOrder{}
	}

	return pagination.NewPageResult(paginated, total, page), nil
}

func companyMatches(ctx context.Context, companyID *int64) bool {
	current := audit.CompanyIDFromContext(ctx)
	return current == nil || (companyID != nil && *companyID == *current)
}

// NextSequence generates a monotonic sequence formatted as "SO/YYYY/NNNNN".
func (r *MemoryRepo) NextSequence(ctx context.Context, year int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if year <= 0 {
		year = time.Now().UTC().Year()
	}

	r.seqCounter[year]++
	seq := fmt.Sprintf("SO/%d/%05d", year, r.seqCounter[year])
	return seq, nil
}

// LinkInvoice links an invoice to an order.
func (r *MemoryRepo) LinkInvoice(ctx context.Context, orderID int64, moveID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[orderID]
	if !ok || !order.Active || !companyMatches(ctx, order.CompanyID) {
		return platformerrors.NotFound("sale order not found")
	}

	for _, id := range r.orderInvoices[orderID] {
		if id == moveID {
			return nil // Already linked
		}
	}

	r.orderInvoices[orderID] = append(r.orderInvoices[orderID], moveID)
	return nil
}

// GetLinkedInvoiceIDs returns linked invoice IDs.
func (r *MemoryRepo) GetLinkedInvoiceIDs(ctx context.Context, orderID int64) ([]int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	invoices := r.orderInvoices[orderID]
	res := make([]int64, len(invoices))
	copy(res, invoices)
	return res, nil
}

func (r *MemoryRepo) populateOrder(o *sale.SaleOrder) *sale.SaleOrder {
	clone := *o
	var lines []sale.SaleOrderLine
	for _, l := range r.orderLines {
		if l.OrderID == o.ID {
			lines = append(lines, *l)
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].Sequence != lines[j].Sequence {
			return lines[i].Sequence < lines[j].Sequence
		}
		return lines[i].ID < lines[j].ID
	})
	clone.Lines = lines

	invoices := r.orderInvoices[o.ID]
	invClone := make([]int64, len(invoices))
	copy(invClone, invoices)
	clone.InvoiceIDs = invClone

	return &clone
}
