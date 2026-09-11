package portalstorage

import (
	"context"
	"cashflow_backend/internal/domain/portal"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) GetUserByEmail(ctx context.Context, email string) (*portal.User, error) {
	return nil, nil
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, id int64) (*portal.User, error) {
	return nil, nil
}

func (r *PostgresRepo) SaveUser(ctx context.Context, user *portal.User) error {
	return nil
}

func (r *PostgresRepo) GetDashboardSummary(ctx context.Context, partnerID int64) (*portal.DashboardSummary, error) {
	return nil, nil
}
