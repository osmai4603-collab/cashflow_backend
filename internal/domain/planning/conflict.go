package planning

import (
	"context"
	"fmt"
)

type ConflictDetector struct {
	repo   Repository
	hrPort HRPort
}

func NewConflictDetector(repo Repository, hrPort HRPort) *ConflictDetector {
	return &ConflictDetector{
		repo:   repo,
		hrPort: hrPort,
	}
}

func (d *ConflictDetector) CheckConflict(ctx context.Context, shift *PlanningShift) error {
	if shift.EmployeeID == nil {
		return nil
	}

	// 1. Check if employee is on vacation/leave via HR port
	onLeave, err := d.hrPort.IsEmployeeOnLeave(ctx, *shift.EmployeeID, shift.StartAt, shift.EndAt)
	if err != nil {
		return fmt.Errorf("failed to verify employee leave status: %w", err)
	}
	if onLeave {
		return errConflict(fmt.Sprintf("employee %d is on leave during this shift period", *shift.EmployeeID))
	}

	// 2. Check for overlapping shifts in persistent store
	existingShifts, err := d.repo.ListShiftsByEmployee(ctx, *shift.EmployeeID, shift.CompanyID)
	if err != nil {
		return fmt.Errorf("failed to list employee shifts: %w", err)
	}

	for _, existing := range existingShifts {
		if existing.ID != shift.ID && shift.Overlaps(&existing) {
			return errConflict(fmt.Sprintf("shift overlaps with an existing shift (ID %d) for employee %d", existing.ID, *shift.EmployeeID))
		}
	}

	return nil
}
