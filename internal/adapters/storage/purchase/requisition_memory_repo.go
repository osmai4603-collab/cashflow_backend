package purchasestorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// MemoryRequisitionRepo is an in-memory store for purchase requisitions.
type MemoryRequisitionRepo struct {
	mu            sync.RWMutex
	items         map[int64]*purchase.PurchaseRequisition
	supplierInfos map[int64][]purchase.SupplierInfo
	lastID        int64
}

// NewMemoryRequisitionRepo creates an empty memory requisition repository.
func NewMemoryRequisitionRepo() *MemoryRequisitionRepo {
	return &MemoryRequisitionRepo{items: make(map[int64]*purchase.PurchaseRequisition), supplierInfos: make(map[int64][]purchase.SupplierInfo)}
}

// CreateRequisition stores a requisition and its lines.
func (r *MemoryRequisitionRepo) CreateRequisition(ctx context.Context, req *purchase.PurchaseRequisition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	req.ID = r.lastID
	if req.CompanyID == 0 {
		if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
			req.CompanyID = *companyID
		}
	}
	req.Active = true
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
	if !ok || !requisitionCompanyMatches(ctx, req.CompanyID) || !req.Active {
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
		if !requisitionCompanyMatches(ctx, req.CompanyID) || !req.Active {
			continue
		}
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

	stored, ok := r.items[req.ID]
	if !ok || !requisitionCompanyMatches(ctx, stored.CompanyID) || !stored.Active {
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
	stored, ok := r.items[id]
	if !ok || !requisitionCompanyMatches(ctx, stored.CompanyID) || !stored.Active {
		return platformerrors.NotFound(fmt.Sprintf("purchase requisition %d not found", id))
	}
	delete(r.items, id)
	return nil
}

func requisitionCompanyMatches(ctx context.Context, companyID int64) bool {
	scope := audit.CompanyIDFromContext(ctx)
	return scope == nil || companyID == 0 || companyID == *scope
}

func (r *MemoryRequisitionRepo) CreateForRequisition(_ context.Context, req *purchase.PurchaseRequisition) error {
	if req == nil || req.Type != purchase.RequisitionBlanketOrder || req.VendorID == nil {
		return platformerrors.Validation("blanket order supplier info requires a vendor", map[string]string{"vendor_id": "must be set"})
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	infos := make([]purchase.SupplierInfo, 0, len(req.Lines))
	for i, line := range req.Lines {
		infos = append(infos, purchase.SupplierInfo{ID: int64(i + 1), RequisitionID: req.ID, LineID: line.ID, ProductID: line.ProductID, VendorID: *req.VendorID, UOMID: line.ProductUOMID, Price: line.PriceUnit, CurrencyID: req.CurrencyID, Active: true})
	}
	r.supplierInfos[req.ID] = infos
	return nil
}

func (r *MemoryRequisitionRepo) DeleteForRequisition(_ context.Context, requisitionID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.supplierInfos, requisitionID)
	return nil
}

func (r *MemoryRequisitionRepo) UpdatePricesForRequisition(_ context.Context, req *purchase.PurchaseRequisition) error {
	if req == nil {
		return platformerrors.Validation("requisition is required", map[string]string{"requisition": "must not be nil"})
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.supplierInfos[req.ID] {
		for _, line := range req.Lines {
			if r.supplierInfos[req.ID][i].LineID == line.ID {
				r.supplierInfos[req.ID][i].Price = line.PriceUnit
			}
		}
	}
	return nil
}
