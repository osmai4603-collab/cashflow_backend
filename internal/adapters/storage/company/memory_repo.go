package companystorage

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/company"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory implementation of company.Repository.
type MemoryRepo struct {
	mu        sync.RWMutex
	companies map[int64]*company.Company
	lastID    int64
}

// NewMemoryRepo creates an initialized MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		companies: make(map[int64]*company.Company),
	}
}

func (r *MemoryRepo) Create(ctx context.Context, c *company.Company) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	c.ID = r.lastID
	c.Active = true

	now := time.Now().UTC()
	if c.Audit.CreatedAt.IsZero() {
		c.Audit.CreatedAt = now
	}
	if c.Audit.UpdatedAt.IsZero() {
		c.Audit.UpdatedAt = now
	}

	clone := *c
	r.companies[c.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*company.Company, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, exists := r.companies[id]
	if !exists || !c.Active {
		return nil, platformerrors.NotFound("company not found")
	}

	clone := *c
	return &clone, nil
}

func (r *MemoryRepo) Update(ctx context.Context, c *company.Company) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.companies[c.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("company not found")
	}

	c.Audit.UpdatedAt = time.Now().UTC()
	clone := *c
	r.companies[c.ID] = &clone
	return nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.companies[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("company not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[company.Company], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*company.Company
	for _, c := range r.companies {
		if !r.matchesFilter(c, f) {
			continue
		}
		filtered = append(filtered, c)
	}

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

	var items []company.Company
	if offset < len(filtered) {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		for _, c := range filtered[offset:end] {
			items = append(items, *c)
		}
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *MemoryRepo) GetDefaultCompany(ctx context.Context) (*company.Company, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if c, exists := r.companies[1]; exists && c.Active {
		clone := *c
		return &clone, nil
	}

	for _, c := range r.companies {
		if c.Active {
			clone := *c
			return &clone, nil
		}
	}

	return nil, platformerrors.NotFound("no active company found")
}

func (r *MemoryRepo) matchesFilter(c *company.Company, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 {
		return c.Active
	}

	for _, crit := range f.Criteria {
		switch strings.ToLower(crit.Field) {
		case "active":
			if val, ok := crit.Value.(bool); ok && c.Active != val {
				return false
			}
		case "name":
			if val, ok := crit.Value.(string); ok {
				if crit.Operator == filter.OpILike || crit.Operator == filter.OpLike {
					if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(val)) {
						return false
					}
				} else if c.Name != val {
					return false
				}
			}
		case "currency_id":
			if val, ok := crit.Value.(int64); ok && c.CurrencyID != val {
				return false
			}
		case "id":
			if val, ok := crit.Value.(int64); ok && c.ID != val {
				return false
			}
		case "country":
			if val, ok := crit.Value.(string); ok && !strings.EqualFold(c.Country, val) {
				return false
			}
		}
	}

	return true
}
