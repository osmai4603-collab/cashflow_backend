package repairstorage

import (
	"cashflow_backend/internal/domain/repair"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CreateOrder(ctx context.Context, order *repair.RepairOrder) error { return nil }
func (r *PostgresRepo) GetOrderByID(ctx context.Context, id int64) (*repair.RepairOrder, error) { return nil, nil }
func (r *PostgresRepo) GetOrderByName(ctx context.Context, name string) (*repair.RepairOrder, error) { return nil, nil }
func (r *PostgresRepo) UpdateOrder(ctx context.Context, order *repair.RepairOrder) error { return nil }
func (r *PostgresRepo) DeleteOrder(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListOrders(ctx context.Context, companyID int64) ([]repair.RepairOrder, error) { return nil, nil }
func (r *PostgresRepo) CreateOrderLine(ctx context.Context, line *repair.RepairOrderLine) error { return nil }
func (r *PostgresRepo) GetLinesByRepairID(ctx context.Context, repairID int64) ([]repair.RepairOrderLine, error) { return nil, nil }
func (r *PostgresRepo) DeleteOrderLine(ctx context.Context, id int64) error { return nil }
