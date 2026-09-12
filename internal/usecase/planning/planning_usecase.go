package planningusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/planning"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo   planning.Repository
	hrPort planning.HRPort
}

func New(repo planning.Repository, hrPort ...planning.HRPort) *UseCase {
	var port planning.HRPort
	if len(hrPort) > 0 {
		port = hrPort[0]
	}
	return &UseCase{repo: repo, hrPort: port}
}

func (u *UseCase) CreateRole(ctx context.Context, role *planning.PlanningRole) (*planning.PlanningRole, error) {
	if u == nil || u.repo == nil {
		return nil, platformerrors.Internal("planning repository is not configured")
	}
	if role == nil {
		return nil, platformerrors.Validation("planning role is required", nil)
	}
	if role.CompanyID == 0 {
		role.CompanyID = 1
	}
	if err := role.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (u *UseCase) ListRoles(ctx context.Context) ([]planning.PlanningRole, error) {
	if u == nil || u.repo == nil {
		return nil, platformerrors.Internal("planning repository is not configured")
	}
	return u.repo.ListRoles(ctx, 0)
}

func (u *UseCase) CreateShift(ctx context.Context, shift *planning.PlanningShift) (*planning.PlanningShift, error) {
	if u == nil || u.repo == nil {
		return nil, platformerrors.Internal("planning repository is not configured")
	}
	if shift == nil {
		return nil, platformerrors.Validation("planning shift is required", nil)
	}
	if shift.CompanyID == 0 {
		shift.CompanyID = 1
	}
	if shift.CreatedAt.IsZero() {
		shift.CreatedAt = time.Now().UTC()
	}
	if err := shift.Validate(); err != nil {
		return nil, err
	}
	if u.hrPort != nil {
		detector := planning.NewConflictDetector(u.repo, u.hrPort)
		if err := detector.CheckConflict(ctx, shift); err != nil {
			return nil, err
		}
	}
	if err := u.repo.CreateShift(ctx, shift); err != nil {
		return nil, err
	}
	return shift, nil
}

func (u *UseCase) GetShift(ctx context.Context, id int64) (*planning.PlanningShift, error) {
	if id <= 0 {
		return nil, platformerrors.Validation("shift_id is required", nil)
	}
	shift, err := u.repo.GetShiftByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("planning shift %d not found", id))
	}
	return shift, nil
}

func (u *UseCase) ListShifts(ctx context.Context, companyID int64) ([]planning.PlanningShift, error) {
	if u == nil || u.repo == nil {
		return nil, platformerrors.Internal("planning repository is not configured")
	}
	if companyID == 0 {
		companyID = 1
	}
	return u.repo.ListShifts(ctx, companyID)
}

func (u *UseCase) ListShiftsByEmployee(ctx context.Context, employeeID int64, companyID int64) ([]planning.PlanningShift, error) {
	if employeeID <= 0 {
		return nil, platformerrors.Validation("employee_id is required", nil)
	}
	if companyID == 0 {
		companyID = 1
	}
	return u.repo.ListShiftsByEmployee(ctx, employeeID, companyID)
}

func (u *UseCase) UpdateShift(ctx context.Context, shift *planning.PlanningShift) (*planning.PlanningShift, error) {
	if shift == nil {
		return nil, platformerrors.Validation("planning shift is required", nil)
	}
	if shift.ID <= 0 {
		return nil, platformerrors.Validation("shift_id is required", nil)
	}
	if shift.CompanyID == 0 {
		shift.CompanyID = 1
	}
	if shift.CreatedAt.IsZero() {
		shift.CreatedAt = time.Now().UTC()
	}
	if err := shift.Validate(); err != nil {
		return nil, err
	}
	if u.hrPort != nil {
		detector := planning.NewConflictDetector(u.repo, u.hrPort)
		if err := detector.CheckConflict(ctx, shift); err != nil {
			return nil, err
		}
	}
	if err := u.repo.UpdateShift(ctx, shift); err != nil {
		return nil, err
	}
	return shift, nil
}

func (u *UseCase) DeleteShift(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.Validation("shift_id is required", nil)
	}
	return u.repo.DeleteShift(ctx, id)
}

func NewConflictDetector(repo planning.Repository, hrPort planning.HRPort) *planning.ConflictDetector {
	return planning.NewConflictDetector(repo, hrPort)
}
