package planningstorage

import (
	"context"
	"cashflow_backend/internal/domain/planning"
	"sync"
)

type MemoryRepo struct {
	mu sync.RWMutex
	roles  map[int64]*planning.PlanningRole
	shifts map[int64]*planning.PlanningShift
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		roles:  make(map[int64]*planning.PlanningRole),
		shifts: make(map[int64]*planning.PlanningShift),
	}
}

func (r *MemoryRepo) CreateRole(ctx context.Context, role *planning.PlanningRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[role.ID] = role
	return nil
}
func (r *MemoryRepo) GetRoleByID(ctx context.Context, id int64) (*planning.PlanningRole, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.roles[id], nil
}
func (r *MemoryRepo) UpdateRole(ctx context.Context, role *planning.PlanningRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[role.ID] = role
	return nil
}
func (r *MemoryRepo) DeleteRole(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.roles, id)
	return nil
}
func (r *MemoryRepo) ListRoles(ctx context.Context, companyID int64) ([]planning.PlanningRole, error) { return nil, nil }
func (r *MemoryRepo) CreateShift(ctx context.Context, shift *planning.PlanningShift) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shifts[shift.ID] = shift
	return nil
}
func (r *MemoryRepo) GetShiftByID(ctx context.Context, id int64) (*planning.PlanningShift, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.shifts[id], nil
}
func (r *MemoryRepo) UpdateShift(ctx context.Context, shift *planning.PlanningShift) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shifts[shift.ID] = shift
	return nil
}
func (r *MemoryRepo) DeleteShift(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.shifts, id)
	return nil
}
func (r *MemoryRepo) ListShifts(ctx context.Context, companyID int64) ([]planning.PlanningShift, error) { return nil, nil }
func (r *MemoryRepo) ListShiftsByEmployee(ctx context.Context, employeeID int64, companyID int64) ([]planning.PlanningShift, error) { return nil, nil }
