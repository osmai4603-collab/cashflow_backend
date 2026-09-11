package mrp

import (
	"context"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// WorkcenterCalendar defines a recurring working interval for a workcenter.
type WorkcenterCalendar struct {
	ID             int64   `json:"id"`
	WorkcenterID   int64   `json:"workcenter_id"`
	DayOfWeek      int     `json:"day_of_week"`
	HourFrom       float64 `json:"hour_from"`
	HourTo         float64 `json:"hour_to"`
	AttendanceType string  `json:"attendance_type"`
	CompanyID      int64   `json:"company_id"`
}

// CapacitySlot represents available and already allocated capacity.
type CapacitySlot struct {
	ID             int64     `json:"id"`
	WorkcenterID   int64     `json:"workcenter_id"`
	DateStart      time.Time `json:"date_start"`
	DateEnd        time.Time `json:"date_end"`
	AvailableHours float64   `json:"available_hours"`
	AllocatedHours float64   `json:"allocated_hours"`
	CompanyID      int64     `json:"company_id"`
}

func (s *CapacitySlot) Validate() error {
	if s.WorkcenterID <= 0 || !s.DateEnd.After(s.DateStart) {
		return platformerrors.Validation("invalid capacity slot", map[string]string{"date_end": "must be after date_start"})
	}
	if s.AvailableHours < 0 || s.AllocatedHours < 0 {
		return platformerrors.Validation("capacity hours cannot be negative", nil)
	}
	return nil
}

// CapacityProvider supplies persisted capacity slots to the scheduling engine.
type CapacityProvider interface {
	CheckCapacity(ctx context.Context, workcenterID int64, dateFrom, dateTo time.Time) (*CapacitySlot, error)
}
