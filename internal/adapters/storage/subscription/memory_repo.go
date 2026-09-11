package subscriptionstorage

import (
	"context"
	"time"
	"cashflow_backend/internal/domain/subscription"
	"sync"
)

type MemoryRepo struct {
	mu sync.RWMutex
	plans map[int64]*subscription.SubscriptionPlan
	subs  map[int64]*subscription.SaleSubscription
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		plans: make(map[int64]*subscription.SubscriptionPlan),
		subs:  make(map[int64]*subscription.SaleSubscription),
	}
}

func (r *MemoryRepo) CreatePlan(ctx context.Context, plan *subscription.SubscriptionPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[plan.ID] = plan
	return nil
}

func (r *MemoryRepo) GetPlanByID(ctx context.Context, id int64) (*subscription.SubscriptionPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plans[id], nil
}

func (r *MemoryRepo) UpdatePlan(ctx context.Context, plan *subscription.SubscriptionPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[plan.ID] = plan
	return nil
}

func (r *MemoryRepo) DeletePlan(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plans, id)
	return nil
}

func (r *MemoryRepo) ListPlans(ctx context.Context, companyID int64) ([]subscription.SubscriptionPlan, error) {
	return nil, nil
}

func (r *MemoryRepo) CreateSubscription(ctx context.Context, sub *subscription.SaleSubscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[sub.ID] = sub
	return nil
}

func (r *MemoryRepo) GetSubscriptionByID(ctx context.Context, id int64) (*subscription.SaleSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.subs[id], nil
}

func (r *MemoryRepo) GetSubscriptionByCode(ctx context.Context, code string) (*subscription.SaleSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.subs {
		if s.Code == code {
			return s, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) UpdateSubscription(ctx context.Context, sub *subscription.SaleSubscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[sub.ID] = sub
	return nil
}

func (r *MemoryRepo) DeleteSubscription(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.subs, id)
	return nil
}

func (r *MemoryRepo) ListSubscriptionsByCompany(ctx context.Context, companyID int64) ([]subscription.SaleSubscription, error) {
	return nil, nil
}

func (r *MemoryRepo) ListDueSubscriptions(ctx context.Context, now time.Time, companyID int64) ([]subscription.SaleSubscription, error) {
	return nil, nil
}
