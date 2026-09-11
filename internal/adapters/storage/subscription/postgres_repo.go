package subscriptionstorage

import (
	"context"
	"time"
	"cashflow_backend/internal/domain/subscription"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CreatePlan(ctx context.Context, plan *subscription.SubscriptionPlan) error { return nil }
func (r *PostgresRepo) GetPlanByID(ctx context.Context, id int64) (*subscription.SubscriptionPlan, error) { return nil, nil }
func (r *PostgresRepo) UpdatePlan(ctx context.Context, plan *subscription.SubscriptionPlan) error { return nil }
func (r *PostgresRepo) DeletePlan(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListPlans(ctx context.Context, companyID int64) ([]subscription.SubscriptionPlan, error) { return nil, nil }
func (r *PostgresRepo) CreateSubscription(ctx context.Context, sub *subscription.SaleSubscription) error { return nil }
func (r *PostgresRepo) GetSubscriptionByID(ctx context.Context, id int64) (*subscription.SaleSubscription, error) { return nil, nil }
func (r *PostgresRepo) GetSubscriptionByCode(ctx context.Context, code string) (*subscription.SaleSubscription, error) { return nil, nil }
func (r *PostgresRepo) UpdateSubscription(ctx context.Context, sub *subscription.SaleSubscription) error { return nil }
func (r *PostgresRepo) DeleteSubscription(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListSubscriptionsByCompany(ctx context.Context, companyID int64) ([]subscription.SaleSubscription, error) { return nil, nil }
func (r *PostgresRepo) ListDueSubscriptions(ctx context.Context, now time.Time, companyID int64) ([]subscription.SaleSubscription, error) { return nil, nil }
