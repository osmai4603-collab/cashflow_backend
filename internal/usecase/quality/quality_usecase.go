package qualityusecase

import (
	"cashflow_backend/internal/domain/quality"
	"context"
)

type UseCase struct {
	repo quality.Repository
}

func (u *UseCase) ListControlPoints(ctx context.Context) ([]quality.QualityControlPoint, error) {
	return nil, nil
}

func (u *UseCase) CreateControlPoint(ctx context.Context, p *quality.QualityControlPoint) (*quality.QualityControlPoint, error) {
	return nil, nil
}

func (u *UseCase) ExecuteCheck(ctx context.Context, c *quality.QualityCheck) (*quality.QualityCheck, error) {
	return nil, nil
}

func (u *UseCase) ListAlerts(ctx context.Context) ([]quality.QualityAlert, error) {
	return nil, nil
}

func (u *UseCase) CreateAlert(ctx context.Context, a *quality.QualityAlert) (*quality.QualityAlert, error) {
	return nil, nil
}

func New(repo quality.Repository) *UseCase {
	return &UseCase{repo: repo}
}
