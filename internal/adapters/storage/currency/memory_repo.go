package currencystorage

import (
	"context"
	"sort"
	"strings"
	"sync"

	"cashflow_backend/internal/domain/currency"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory implementation of currency.Repository.
type MemoryRepo struct {
	mu         sync.RWMutex
	currencies map[int64]*currency.Currency
	lastID     int64
}

// NewMemoryRepo creates an initialized MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		currencies: make(map[int64]*currency.Currency),
	}
}

func (r *MemoryRepo) Create(ctx context.Context, c *currency.Currency) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	c.ID = r.lastID
	c.Active = true

	clone := *c
	r.currencies[c.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*currency.Currency, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, exists := r.currencies[id]
	if !exists || !c.Active {
		return nil, platformerrors.NotFound("currency not found")
	}

	clone := *c
	return &clone, nil
}

func (r *MemoryRepo) GetByName(ctx context.Context, name string) (*currency.Currency, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name = strings.ToUpper(strings.TrimSpace(name))
	for _, c := range r.currencies {
		if c.Active && strings.EqualFold(c.Name, name) {
			clone := *c
			return &clone, nil
		}
	}

	return nil, platformerrors.NotFound("currency not found")
}

func (r *MemoryRepo) Update(ctx context.Context, c *currency.Currency) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.currencies[c.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("currency not found")
	}

	clone := *c
	r.currencies[c.ID] = &clone
	return nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.currencies[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("currency not found")
	}

	existing.Active = false
	return nil
}

func (r *MemoryRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[currency.Currency], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*currency.Currency
	for _, c := range r.currencies {
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
		case "full_name":
			if page.OrderDirection() == "DESC" {
				return filtered[i].FullName > filtered[j].FullName
			}
			return filtered[i].FullName < filtered[j].FullName
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

	var items []currency.Currency
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

func (r *MemoryRepo) matchesFilter(c *currency.Currency, f *filter.Filter) bool {
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
				} else if !strings.EqualFold(c.Name, val) {
					return false
				}
			}
		case "full_name":
			if val, ok := crit.Value.(string); ok && !strings.EqualFold(c.FullName, val) {
				return false
			}
		case "id":
			if val, ok := crit.Value.(int64); ok && c.ID != val {
				return false
			}
		}
	}

	return true
}
