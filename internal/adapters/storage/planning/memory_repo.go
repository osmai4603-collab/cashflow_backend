package planningstorage

import (
	"context"
	"sort"
	"sync"

	"cashflow_backend/internal/domain/planning"
)

type MemoryRepo struct {
	mu     sync.RWMutex
	roles  map[int64]*planning.PlanningRole
	shifts map[int64]*planning.PlanningShift
	nextID int64
}

func (r *MemoryRepo) allocateID() int64 {
	r.nextID++
	return r.nextID
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
	if role.ID == 0 {
		role.ID = r.allocateID()
	}
	r.roles[role.ID] = role
	return nil
}

func (r *MemoryRepo) GetRoleByID(ctx context.Context, id int64) (*planning.PlanningRole, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	role, ok := r.roles[id]
	if !ok {
		return nil, nil
	}
	return role, nil
}

func (r *MemoryRepo) UpdateRole(ctx context.Context, role *planning.PlanningRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if role.ID == 0 {
		role.ID = r.allocateID()
	}
	r.roles[role.ID] = role
	return nil
}

func (r *MemoryRepo) DeleteRole(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.roles, id)
	return nil
}

func (r *MemoryRepo) ListRoles(ctx context.Context, companyID int64) ([]planning.PlanningRole, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]planning.PlanningRole, 0, len(r.roles))
	for _, role := range r.roles {
		if companyID == 0 || role.CompanyID == companyID {
			result = append(result, *role)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateShift(ctx context.Context, shift *planning.PlanningShift) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if shift.ID == 0 {
		shift.ID = r.allocateID()
	}
	r.shifts[shift.ID] = shift
	return nil
}

func (r *MemoryRepo) GetShiftByID(ctx context.Context, id int64) (*planning.PlanningShift, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	shift, ok := r.shifts[id]
	if !ok {
		return nil, nil
	}
	return shift, nil
}

func (r *MemoryRepo) UpdateShift(ctx context.Context, shift *planning.PlanningShift) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if shift.ID == 0 {
		shift.ID = r.allocateID()
	}
	r.shifts[shift.ID] = shift
	return nil
}

func (r *MemoryRepo) DeleteShift(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.shifts, id)
	return nil
}

func (r *MemoryRepo) ListShifts(ctx context.Context, companyID int64) ([]planning.PlanningShift, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]planning.PlanningShift, 0, len(r.shifts))
	for _, shift := range r.shifts {
		if companyID == 0 || shift.CompanyID == companyID {
			result = append(result, *shift)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartAt.Before(result[j].StartAt) })
	return result, nil
}

func (r *MemoryRepo) ListShiftsByEmployee(ctx context.Context, employeeID int64, companyID int64) ([]planning.PlanningShift, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]planning.PlanningShift, 0)
	for _, shift := range r.shifts {
		if shift.EmployeeID != nil && *shift.EmployeeID == employeeID && (companyID == 0 || shift.CompanyID == companyID) {
			result = append(result, *shift)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartAt.Before(result[j].StartAt) })
	return result, nil
}
