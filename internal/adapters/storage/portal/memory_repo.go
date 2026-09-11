package portalstorage

import (
	"context"
	"cashflow_backend/internal/domain/portal"
	"sync"
)

type MemoryRepo struct {
	mu    sync.RWMutex
	users map[int64]*portal.User
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		users: make(map[int64]*portal.User),
	}
}

func (r *MemoryRepo) GetUserByEmail(ctx context.Context, email string) (*portal.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) GetUserByID(ctx context.Context, id int64) (*portal.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.users[id], nil
}

func (r *MemoryRepo) SaveUser(ctx context.Context, user *portal.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID] = user
	return nil
}

func (r *MemoryRepo) GetDashboardSummary(ctx context.Context, partnerID int64) (*portal.DashboardSummary, error) {
	return nil, nil
}
