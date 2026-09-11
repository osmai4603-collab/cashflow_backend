package planning

import (
	"time"
)

type PlanningShift struct {
	ID             int64      `json:"id"`
	EmployeeID     *int64     `json:"employee_id,omitempty"`
	RoleID         int64      `json:"role_id"`
	StartAt        time.Time  `json:"start_at"`
	EndAt          time.Time  `json:"end_at"`
	AllocatedHours float64    `json:"allocated_hours"`
	IsPublished    bool       `json:"is_published"`
	CompanyID      int64      `json:"company_id"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (s *PlanningShift) Validate() error {
	if s.RoleID <= 0 {
		return errInvalid("role ID is required")
	}
	if s.StartAt.IsZero() || s.EndAt.IsZero() {
		return errInvalid("start and end times are required")
	}
	if s.StartAt.After(s.EndAt) {
		return errInvalid("start time cannot be after end time")
	}
	if s.AllocatedHours <= 0 {
		return errInvalid("allocated hours must be greater than zero")
	}
	if s.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	return nil
}

func (s *PlanningShift) Overlaps(other *PlanningShift) bool {
	if s.EmployeeID == nil || other.EmployeeID == nil || *s.EmployeeID != *other.EmployeeID {
		return false
	}
	return s.StartAt.Before(other.EndAt) && s.EndAt.After(other.StartAt)
}
