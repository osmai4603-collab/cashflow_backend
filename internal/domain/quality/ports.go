package quality

import (
	"context"
)

type Repository interface {
	CreateControlPoint(ctx context.Context, point *QualityControlPoint) error
	GetControlPointByID(ctx context.Context, id int64) (*QualityControlPoint, error)
	UpdateControlPoint(ctx context.Context, point *QualityControlPoint) error
	DeleteControlPoint(ctx context.Context, id int64) error
	ListControlPoints(ctx context.Context, companyID int64) ([]QualityControlPoint, error)
	ListControlPointsByTrigger(ctx context.Context, companyID int64, trigger QualityPointTrigger) ([]QualityControlPoint, error)

	CreateQualityCheck(ctx context.Context, check *QualityCheck) error
	GetQualityCheckByID(ctx context.Context, id int64) (*QualityCheck, error)
	UpdateQualityCheck(ctx context.Context, check *QualityCheck) error
	ListQualityChecks(ctx context.Context, companyID int64) ([]QualityCheck, error)

	CreateAlert(ctx context.Context, alert *QualityAlert) error
	GetAlertByID(ctx context.Context, id int64) (*QualityAlert, error)
	UpdateAlert(ctx context.Context, alert *QualityAlert) error
	DeleteAlert(ctx context.Context, id int64) error
	ListAlerts(ctx context.Context, companyID int64) ([]QualityAlert, error)
}
