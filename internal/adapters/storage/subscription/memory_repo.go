package subscriptionstorage

import (
	"cashflow_backend/internal/domain/subscription"
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryRepo struct {
	mu     sync.RWMutex
	plans  map[int64]*subscription.SubscriptionPlan
	subs   map[int64]*subscription.SaleSubscription
	nextID int64
}

func (r *MemoryRepo) allocateID() int64 { r.nextID++; return r.nextID }

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		plans: make(map[int64]*subscription.SubscriptionPlan),
		subs:  make(map[int64]*subscription.SaleSubscription),
	}
}

func (r *MemoryRepo) CreatePlan(ctx context.Context, plan *subscription.SubscriptionPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if plan.ID == 0 {
		plan.ID = r.allocateID()
	}
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]subscription.SubscriptionPlan, 0)
	for _, plan := range r.plans {
		if companyID == 0 || plan.CompanyID == companyID {
			result = append(result, *plan)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateSubscription(ctx context.Context, sub *subscription.SaleSubscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sub.ID == 0 {
		sub.ID = r.allocateID()
	}
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]subscription.SaleSubscription, 0)
	for _, sub := range r.subs {
		if companyID == 0 || sub.CompanyID == companyID {
			result = append(result, *sub)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) ListDueSubscriptions(ctx context.Context, now time.Time, companyID int64) ([]subscription.SaleSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]subscription.SaleSubscription, 0)
	for _, sub := range r.subs {
		if (companyID == 0 || sub.CompanyID == companyID) && sub.State == subscription.StateActive && !sub.NextBillingDate.After(now) && (sub.EndDate == nil || sub.EndDate.After(now)) {
			result = append(result, *sub)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].NextBillingDate.Before(result[j].NextBillingDate) })
	return result, nil
}
