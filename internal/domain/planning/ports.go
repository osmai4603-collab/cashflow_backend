package planning

import (
	"context"
	"time"
)

type Repository interface {
	CreateRole(ctx context.Context, role *PlanningRole) error
	GetRoleByID(ctx context.Context, id int64) (*PlanningRole, error)
	UpdateRole(ctx context.Context, role *PlanningRole) error
	DeleteRole(ctx context.Context, id int64) error
	ListRoles(ctx context.Context, companyID int64) ([]PlanningRole, error)

	CreateShift(ctx context.Context, shift *PlanningShift) error
	GetShiftByID(ctx context.Context, id int64) (*PlanningShift, error)
	UpdateShift(ctx context.Context, shift *PlanningShift) error
	DeleteShift(ctx context.Context, id int64) error
	ListShifts(ctx context.Context, companyID int64) ([]PlanningShift, error)
	ListShiftsByEmployee(ctx context.Context, employeeID int64, companyID int64) ([]PlanningShift, error)
}

type HRPort interface {
	IsEmployeeOnLeave(ctx context.Context, employeeID int64, start time.Time, end time.Time) (bool, error)
}
