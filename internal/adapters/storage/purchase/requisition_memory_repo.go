package purchasestorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cashflow_backend/internal/domain/purchase"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// MemoryRequisitionRepo is an in-memory store for purchase requisitions.
type MemoryRequisitionRepo struct {
	mu       sync.RWMutex
	items    map[int64]*purchase.PurchaseRequisition
	lastID   int64
}

// NewMemoryRequisitionRepo creates an empty memory requisition repository.
func NewMemoryRequisitionRepo() *MemoryRequisitionRepo {
	return &MemoryRequisitionRepo{items: make(map[int64]*purchase.PurchaseRequisition)}
}

// CreateRequisition stores a requisition and its lines.
func (r *MemoryRequisitionRepo) CreateRequisition(ctx context.Context, req *purchase.PurchaseRequisition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	req.ID = r.lastID
	now := time.Now().UTC()
	for i := range req.Lines {
		req.Lines[i].ID = int64(i + 1)
		req.Lines[i].RequisitionID = req.ID
		req.Lines[i].CreatedAt = now
		req.Lines[i].UpdatedAt = now
	}
	req.CreatedAt = now
	req.UpdatedAt = now
	r.items[req.ID] = req
	return nil
}

// GetRequisitionByID loads a requisition by ID.
func (r *MemoryRequisitionRepo) GetRequisitionByID(ctx context.Context, id int64) (*purchase.PurchaseRequisition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, ok := r.items[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("purchase requisition %d not found", id))
	}
	clone := *req
	lines := make([]purchase.PurchaseRequisitionLine, len(req.Lines))
	copy(lines, req.Lines)
	clone.Lines = lines
	return &clone, nil
}

// ListRequisitions returns a simple list in newest-first order.
func (r *MemoryRequisitionRepo) ListRequisitions(ctx context.Context, page int, limit int) ([]purchase.PurchaseRequisition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]purchase.PurchaseRequisition, 0, len(r.items))
	for _, req := range r.items {
		clone := *req
		lines := make([]purchase.PurchaseRequisitionLine, len(req.Lines))
		copy(lines, req.Lines)
		clone.Lines = lines
		items = append(items, clone)
	}
	if len(items) == 0 {
		return items, nil
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = len(items)
	}
	start := (page - 1) * limit
	if start >= len(items) {
		return []purchase.PurchaseRequisition{}, nil
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], nil
}

// UpdateRequisition persists changes to an existing requisition.
func (r *MemoryRequisitionRepo) UpdateRequisition(ctx context.Context, req *purchase.PurchaseRequisition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[req.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("purchase requisition %d not found", req.ID))
	}
	now := time.Now().UTC()
	for i := range req.Lines {
		if req.Lines[i].ID == 0 {
			req.Lines[i].ID = int64(i + 1)
		}
		req.Lines[i].RequisitionID = req.ID
		req.Lines[i].UpdatedAt = now
	}
	req.UpdatedAt = now
	r.items[req.ID] = req
	return nil
}

// DeleteRequisition removes a requisition.
func (r *MemoryRequisitionRepo) DeleteRequisition(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("purchase requisition %d not found", id))
	}
	delete(r.items, id)
	return nil
}
