package qualityusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/quality"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo quality.Repository
}

func (u *UseCase) ListControlPoints(ctx context.Context) ([]quality.QualityControlPoint, error) {
	return u.repo.ListControlPoints(ctx, 0)
}

func (u *UseCase) CreateControlPoint(ctx context.Context, p *quality.QualityControlPoint) (*quality.QualityControlPoint, error) {
	if p == nil {
		return nil, platformerrors.Validation("quality control point is required", nil)
	}
	if p.CompanyID == 0 {
		p.CompanyID = 1
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.CreateControlPoint(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (u *UseCase) ExecuteCheck(ctx context.Context, c *quality.QualityCheck) (*quality.QualityCheck, error) {
	if c == nil {
		return nil, platformerrors.Validation("quality check is required", nil)
	}
	if c.CompanyID == 0 {
		c.CompanyID = 1
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	point, err := u.repo.GetControlPointByID(ctx, c.PointID)
	if err != nil {
		return nil, err
	}
	if point == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("quality control point %d not found", c.PointID))
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if point.TestType == "measure" && c.MeasureValue == nil {
		return nil, platformerrors.Validation("measure value is required", nil)
	}
	c.PerformCheck(point, c.MeasureValue)
	if err := u.repo.CreateQualityCheck(ctx, c); err != nil {
		return nil, err
	}
	if c.State == quality.CheckStateFail {
		alert := &quality.QualityAlert{
			Name:        fmt.Sprintf("%s failed", point.Name),
			ProductID:   c.ProductID,
			LotID:       c.LotID,
			PickingID:   c.PickingID,
			Description: fmt.Sprintf("Quality check %s failed for product %d.", point.Name, c.ProductID),
			Stage:       quality.StageNew,
			CompanyID:   c.CompanyID,
			CreatedAt:   time.Now().UTC(),
		}
		if err := u.repo.CreateAlert(ctx, alert); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (u *UseCase) ListAlerts(ctx context.Context) ([]quality.QualityAlert, error) {
	return u.repo.ListAlerts(ctx, 0)
}

func (u *UseCase) CreateAlert(ctx context.Context, a *quality.QualityAlert) (*quality.QualityAlert, error) {
	if a == nil {
		return nil, platformerrors.Validation("quality alert is required", nil)
	}
	if a.CompanyID == 0 {
		a.CompanyID = 1
	}
	if a.Stage == "" {
		a.Stage = quality.StageNew
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if err := u.repo.CreateAlert(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func New(repo quality.Repository) *UseCase {
	return &UseCase{repo: repo}
}
