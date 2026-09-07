package userstorage

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/user"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory implementation of user.Repository.
type MemoryRepo struct {
	mu     sync.RWMutex
	users  map[int64]*user.User
	lastID int64
}

// NewMemoryRepo creates an initialized MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		users: make(map[int64]*user.User),
	}
}

func (r *MemoryRepo) Create(ctx context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	u.ID = r.lastID
	u.Active = true

	now := time.Now().UTC()
	if u.Audit.CreatedAt.IsZero() {
		u.Audit.CreatedAt = now
	}
	if u.Audit.UpdatedAt.IsZero() {
		u.Audit.UpdatedAt = now
	}

	clone := *u
	r.users[u.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, exists := r.users[id]
	if !exists || !u.Active {
		return nil, platformerrors.NotFound("user not found")
	}

	clone := *u
	return &clone, nil
}

func (r *MemoryRepo) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	login = strings.ToLower(strings.TrimSpace(login))
	for _, u := range r.users {
		if u.Active && strings.EqualFold(u.Login, login) {
			clone := *u
			return &clone, nil
		}
	}

	return nil, platformerrors.NotFound("user not found")
}

func (r *MemoryRepo) Update(ctx context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.users[u.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("user not found")
	}

	u.Audit.UpdatedAt = time.Now().UTC()
	clone := *u
	r.users[u.ID] = &clone
	return nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.users[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("user not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[user.User], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*user.User
	for _, u := range r.users {
		if !r.matchesFilter(u, f) {
			continue
		}
		filtered = append(filtered, u)
	}

	sort.Slice(filtered, func(i, j int) bool {
		switch strings.ToLower(page.SortBy) {
		case "name":
			if page.OrderDirection() == "DESC" {
				return filtered[i].Name > filtered[j].Name
			}
			return filtered[i].Name < filtered[j].Name
		case "login":
			if page.OrderDirection() == "DESC" {
				return filtered[i].Login > filtered[j].Login
			}
			return filtered[i].Login < filtered[j].Login
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

	var items []user.User
	if offset < len(filtered) {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		for _, u := range filtered[offset:end] {
			items = append(items, *u)
		}
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *MemoryRepo) UpdateLastLogin(ctx context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.users[userID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("user not found")
	}

	now := time.Now().UTC()
	existing.LastLoginAt = &now
	return nil
}

func (r *MemoryRepo) matchesFilter(u *user.User, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 {
		return u.Active
	}

	for _, crit := range f.Criteria {
		switch strings.ToLower(crit.Field) {
		case "active":
			if val, ok := crit.Value.(bool); ok && u.Active != val {
				return false
			}
		case "name":
			if val, ok := crit.Value.(string); ok {
				if crit.Operator == filter.OpILike || crit.Operator == filter.OpLike {
					if !strings.Contains(strings.ToLower(u.Name), strings.ToLower(val)) {
						return false
					}
				} else if u.Name != val {
					return false
				}
			}
		case "login":
			if val, ok := crit.Value.(string); ok {
				if crit.Operator == filter.OpILike || crit.Operator == filter.OpLike {
					if !strings.Contains(strings.ToLower(u.Login), strings.ToLower(val)) {
						return false
					}
				} else if !strings.EqualFold(u.Login, val) {
					return false
				}
			}
		case "email":
			if val, ok := crit.Value.(string); ok && !strings.EqualFold(u.Email, val) {
				return false
			}
		case "company_id":
			if val, ok := crit.Value.(int64); ok && u.CompanyID != val {
				return false
			}
		case "partner_id":
			if val, ok := crit.Value.(int64); ok && u.PartnerID != val {
				return false
			}
		}
	}

	return true
}
