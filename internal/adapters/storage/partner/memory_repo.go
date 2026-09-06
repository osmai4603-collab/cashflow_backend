package partnerstorage

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/partner"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory implementation of partner.Repository.
type MemoryRepo struct {
	mu       sync.RWMutex
	partners map[int64]*partner.Partner
	lastID   int64
}

// NewMemoryRepo creates an initialized MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		partners: make(map[int64]*partner.Partner),
	}
}

func (r *MemoryRepo) Create(ctx context.Context, p *partner.Partner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	p.ID = r.lastID
	p.Active = true

	now := time.Now().UTC()
	if p.Audit.CreatedAt.IsZero() {
		p.Audit.CreatedAt = now
	}
	if p.Audit.UpdatedAt.IsZero() {
		p.Audit.UpdatedAt = now
	}

	clone := *p
	r.partners[p.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*partner.Partner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.partners[id]
	if !exists || !p.Active {
		return nil, platformerrors.NotFound("partner not found")
	}

	clone := *p
	return &clone, nil
}

func (r *MemoryRepo) Update(ctx context.Context, p *partner.Partner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.partners[p.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("partner not found")
	}

	p.Audit.UpdatedAt = time.Now().UTC()
	clone := *p
	r.partners[p.ID] = &clone
	return nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.partners[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("partner not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*partner.Partner
	for _, p := range r.partners {
		if !r.matchesFilter(p, f) {
			continue
		}
		filtered = append(filtered, p)
	}

	// Sorting
	sort.Slice(filtered, func(i, j int) bool {
		switch strings.ToLower(page.SortBy) {
		case "name":
			if page.OrderDirection() == "DESC" {
				return filtered[i].Name > filtered[j].Name
			}
			return filtered[i].Name < filtered[j].Name
		case "created_at":
			if page.OrderDirection() == "DESC" {
				return filtered[i].Audit.CreatedAt.After(filtered[j].Audit.CreatedAt)
			}
			return filtered[i].Audit.CreatedAt.Before(filtered[j].Audit.CreatedAt)
		default:
			if page.OrderDirection() == "DESC" {
				return filtered[i].ID > filtered[j].ID
			}
			return filtered[i].ID < filtered[j].ID
		}
	})

	totalItems := int64(len(filtered))
	offset := page.Offset()
	limit := page.LimitClamped()

	var items []partner.Partner
	if offset < len(filtered) {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		for _, p := range filtered[offset:end] {
			items = append(items, *p)
		}
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *MemoryRepo) ListCustomers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	f := filter.NewFilter(
		filter.Criterion{Field: "is_customer", Operator: filter.OpEqual, Value: true},
		filter.Criterion{Field: "active", Operator: filter.OpEqual, Value: true},
	)
	return r.List(ctx, f, page)
}

func (r *MemoryRepo) ListSuppliers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	f := filter.NewFilter(
		filter.Criterion{Field: "is_supplier", Operator: filter.OpEqual, Value: true},
		filter.Criterion{Field: "active", Operator: filter.OpEqual, Value: true},
	)
	return r.List(ctx, f, page)
}

func (r *MemoryRepo) matchesFilter(p *partner.Partner, f *filter.Filter) bool {
	// If no filter specified, default to active only
	if f == nil || len(f.Criteria) == 0 {
		return p.Active
	}

	for _, c := range f.Criteria {
		switch strings.ToLower(c.Field) {
		case "active":
			if val, ok := c.Value.(bool); ok && p.Active != val {
				return false
			}
		case "is_customer":
			if val, ok := c.Value.(bool); ok && p.IsCustomer != val {
				return false
			}
		case "is_supplier":
			if val, ok := c.Value.(bool); ok && p.IsSupplier != val {
				return false
			}
		case "type":
			if val, ok := c.Value.(string); ok && string(p.Type) != val {
				return false
			}
		case "name":
			if val, ok := c.Value.(string); ok {
				if c.Operator == filter.OpILike || c.Operator == filter.OpLike {
					if !strings.Contains(strings.ToLower(p.Name), strings.ToLower(val)) {
						return false
					}
				} else if p.Name != val {
					return false
				}
			}
		case "city":
			if val, ok := c.Value.(string); ok && !strings.EqualFold(p.City, val) {
				return false
			}
		case "country":
			if val, ok := c.Value.(string); ok && !strings.EqualFold(p.Country, val) {
				return false
			}
		}
	}

	return true
}
