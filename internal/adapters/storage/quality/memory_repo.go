package qualitystorage

import (
	"context"
	"cashflow_backend/internal/domain/quality"
	"sync"
)

type MemoryRepo struct {
	mu sync.RWMutex
	points map[int64]*quality.QualityControlPoint
	checks map[int64]*quality.QualityCheck
	alerts map[int64]*quality.QualityAlert
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		points: make(map[int64]*quality.QualityControlPoint),
		checks: make(map[int64]*quality.QualityCheck),
		alerts: make(map[int64]*quality.QualityAlert),
	}
}

func (r *MemoryRepo) CreateControlPoint(ctx context.Context, point *quality.QualityControlPoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.points[point.ID] = point
	return nil
}
func (r *MemoryRepo) GetControlPointByID(ctx context.Context, id int64) (*quality.QualityControlPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.points[id], nil
}
func (r *MemoryRepo) UpdateControlPoint(ctx context.Context, point *quality.QualityControlPoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.points[point.ID] = point
	return nil
}
func (r *MemoryRepo) DeleteControlPoint(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.points, id)
	return nil
}
func (r *MemoryRepo) ListControlPoints(ctx context.Context, companyID int64) ([]quality.QualityControlPoint, error) { return nil, nil }
func (r *MemoryRepo) ListControlPointsByTrigger(ctx context.Context, companyID int64, trigger quality.QualityPointTrigger) ([]quality.QualityControlPoint, error) { return nil, nil }
func (r *MemoryRepo) CreateQualityCheck(ctx context.Context, check *quality.QualityCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[check.ID] = check
	return nil
}
func (r *MemoryRepo) GetQualityCheckByID(ctx context.Context, id int64) (*quality.QualityCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.checks[id], nil
}
func (r *MemoryRepo) UpdateQualityCheck(ctx context.Context, check *quality.QualityCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[check.ID] = check
	return nil
}
func (r *MemoryRepo) ListQualityChecks(ctx context.Context, companyID int64) ([]quality.QualityCheck, error) { return nil, nil }
func (r *MemoryRepo) CreateAlert(ctx context.Context, alert *quality.QualityAlert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alerts[alert.ID] = alert
	return nil
}
func (r *MemoryRepo) GetAlertByID(ctx context.Context, id int64) (*quality.QualityAlert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.alerts[id], nil
}
func (r *MemoryRepo) UpdateAlert(ctx context.Context, alert *quality.QualityAlert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alerts[alert.ID] = alert
	return nil
}
func (r *MemoryRepo) DeleteAlert(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.alerts, id)
	return nil
}
func (r *MemoryRepo) ListAlerts(ctx context.Context, companyID int64) ([]quality.QualityAlert, error) { return nil, nil }
