package planningstorage

import (
	"context"
	"cashflow_backend/internal/domain/planning"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CreateRole(ctx context.Context, role *planning.PlanningRole) error { return nil }
func (r *PostgresRepo) GetRoleByID(ctx context.Context, id int64) (*planning.PlanningRole, error) { return nil, nil }
func (r *PostgresRepo) UpdateRole(ctx context.Context, role *planning.PlanningRole) error { return nil }
func (r *PostgresRepo) DeleteRole(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListRoles(ctx context.Context, companyID int64) ([]planning.PlanningRole, error) { return nil, nil }
func (r *PostgresRepo) CreateShift(ctx context.Context, shift *planning.PlanningShift) error { return nil }
func (r *PostgresRepo) GetShiftByID(ctx context.Context, id int64) (*planning.PlanningShift, error) { return nil, nil }
func (r *PostgresRepo) UpdateShift(ctx context.Context, shift *planning.PlanningShift) error { return nil }
func (r *PostgresRepo) DeleteShift(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListShifts(ctx context.Context, companyID int64) ([]planning.PlanningShift, error) { return nil, nil }
func (r *PostgresRepo) ListShiftsByEmployee(ctx context.Context, employeeID int64, companyID int64) ([]planning.PlanningShift, error) { return nil, nil }
