package qualitystorage

import (
	"context"
	"sort"
	"sync"

	"cashflow_backend/internal/domain/quality"
)

type MemoryRepo struct {
	mu     sync.RWMutex
	points map[int64]*quality.QualityControlPoint
	checks map[int64]*quality.QualityCheck
	alerts map[int64]*quality.QualityAlert
	nextID int64
}

func (r *MemoryRepo) allocateID() int64 {
	r.nextID++
	return r.nextID
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
	if point.ID == 0 {
		point.ID = r.allocateID()
	}
	r.points[point.ID] = point
	return nil
}

func (r *MemoryRepo) GetControlPointByID(ctx context.Context, id int64) (*quality.QualityControlPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	point, ok := r.points[id]
	if !ok {
		return nil, nil
	}
	return point, nil
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

func (r *MemoryRepo) ListControlPoints(ctx context.Context, companyID int64) ([]quality.QualityControlPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]quality.QualityControlPoint, 0)
	for _, point := range r.points {
		if companyID == 0 || point.CompanyID == companyID {
			result = append(result, *point)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) ListControlPointsByTrigger(ctx context.Context, companyID int64, trigger quality.QualityPointTrigger) ([]quality.QualityControlPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]quality.QualityControlPoint, 0)
	for _, point := range r.points {
		if (companyID == 0 || point.CompanyID == companyID) && point.Trigger == trigger {
			result = append(result, *point)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateQualityCheck(ctx context.Context, check *quality.QualityCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if check.ID == 0 {
		check.ID = r.allocateID()
	}
	r.checks[check.ID] = check
	return nil
}

func (r *MemoryRepo) GetQualityCheckByID(ctx context.Context, id int64) (*quality.QualityCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	check, ok := r.checks[id]
	if !ok {
		return nil, nil
	}
	return check, nil
}

func (r *MemoryRepo) UpdateQualityCheck(ctx context.Context, check *quality.QualityCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[check.ID] = check
	return nil
}

func (r *MemoryRepo) ListQualityChecks(ctx context.Context, companyID int64) ([]quality.QualityCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]quality.QualityCheck, 0)
	for _, check := range r.checks {
		if companyID == 0 || check.CompanyID == companyID {
			result = append(result, *check)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateAlert(ctx context.Context, alert *quality.QualityAlert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if alert.ID == 0 {
		alert.ID = r.allocateID()
	}
	r.alerts[alert.ID] = alert
	return nil
}

func (r *MemoryRepo) GetAlertByID(ctx context.Context, id int64) (*quality.QualityAlert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	alert, ok := r.alerts[id]
	if !ok {
		return nil, nil
	}
	return alert, nil
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

func (r *MemoryRepo) ListAlerts(ctx context.Context, companyID int64) ([]quality.QualityAlert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]quality.QualityAlert, 0)
	for _, alert := range r.alerts {
		if companyID == 0 || alert.CompanyID == companyID {
			result = append(result, *alert)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
