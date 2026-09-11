package qualitystorage

import (
	"context"
	"cashflow_backend/internal/domain/quality"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CreateControlPoint(ctx context.Context, point *quality.QualityControlPoint) error { return nil }
func (r *PostgresRepo) GetControlPointByID(ctx context.Context, id int64) (*quality.QualityControlPoint, error) { return nil, nil }
func (r *PostgresRepo) UpdateControlPoint(ctx context.Context, point *quality.QualityControlPoint) error { return nil }
func (r *PostgresRepo) DeleteControlPoint(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListControlPoints(ctx context.Context, companyID int64) ([]quality.QualityControlPoint, error) { return nil, nil }
func (r *PostgresRepo) ListControlPointsByTrigger(ctx context.Context, companyID int64, trigger quality.QualityPointTrigger) ([]quality.QualityControlPoint, error) { return nil, nil }
func (r *PostgresRepo) CreateQualityCheck(ctx context.Context, check *quality.QualityCheck) error { return nil }
func (r *PostgresRepo) GetQualityCheckByID(ctx context.Context, id int64) (*quality.QualityCheck, error) { return nil, nil }
func (r *PostgresRepo) UpdateQualityCheck(ctx context.Context, check *quality.QualityCheck) error { return nil }
func (r *PostgresRepo) ListQualityChecks(ctx context.Context, companyID int64) ([]quality.QualityCheck, error) { return nil, nil }
func (r *PostgresRepo) CreateAlert(ctx context.Context, alert *quality.QualityAlert) error { return nil }
func (r *PostgresRepo) GetAlertByID(ctx context.Context, id int64) (*quality.QualityAlert, error) { return nil, nil }
func (r *PostgresRepo) UpdateAlert(ctx context.Context, alert *quality.QualityAlert) error { return nil }
func (r *PostgresRepo) DeleteAlert(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListAlerts(ctx context.Context, companyID int64) ([]quality.QualityAlert, error) { return nil, nil }
