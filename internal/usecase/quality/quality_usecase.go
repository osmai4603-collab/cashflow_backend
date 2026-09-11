package qualityusecase

import (
	"context"
	"cashflow_backend/internal/domain/quality"
)

type UseCase struct {
	repo quality.Repository
}

func (u *UseCase) ListControlPoints(ctx context.Context) ([]quality.ControlPoint, error) {
	return nil, nil
}

func (u *UseCase) CreateControlPoint(ctx context.Context, p *quality.ControlPoint) (*quality.ControlPoint, error) {
	return nil, nil
}

func (u *UseCase) ExecuteCheck(ctx context.Context, c *quality.Check) (*quality.Check, error) {
	return nil, nil
}

func (u *UseCase) ListAlerts(ctx context.Context) ([]quality.Alert, error) {
	return nil, nil
}

func (u *UseCase) CreateAlert(ctx context.Context, a *quality.Alert) (*quality.Alert, error) {
	return nil, nil
}

func New(repo quality.Repository) *UseCase {
	return &UseCase{repo: repo}
}
