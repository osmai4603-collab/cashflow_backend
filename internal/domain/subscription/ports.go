package subscription

import (
	"context"
	"time"
)

type Repository interface {
	CreatePlan(ctx context.Context, plan *SubscriptionPlan) error
	GetPlanByID(ctx context.Context, id int64) (*SubscriptionPlan, error)
	UpdatePlan(ctx context.Context, plan *SubscriptionPlan) error
	DeletePlan(ctx context.Context, id int64) error
	ListPlans(ctx context.Context, companyID int64) ([]SubscriptionPlan, error)

	CreateSubscription(ctx context.Context, sub *SaleSubscription) error
	GetSubscriptionByID(ctx context.Context, id int64) (*SaleSubscription, error)
	GetSubscriptionByCode(ctx context.Context, code string) (*SaleSubscription, error)
	UpdateSubscription(ctx context.Context, sub *SaleSubscription) error
	DeleteSubscription(ctx context.Context, id int64) error
	ListSubscriptionsByCompany(ctx context.Context, companyID int64) ([]SaleSubscription, error)
	ListDueSubscriptions(ctx context.Context, now time.Time, companyID int64) ([]SaleSubscription, error)
}

type InvoicePort interface {
	CreateSubscriptionInvoice(ctx context.Context, sub *SaleSubscription, plan *SubscriptionPlan) error
}
