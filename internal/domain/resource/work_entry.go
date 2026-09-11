package resource

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type WorkEntry struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	EmployeeID    int64     `json:"employee_id"`
	WorkEntryType string    `json:"work_entry_type"`
	DateStart     time.Time `json:"date_start"`
	DateStop      time.Time `json:"date_stop"`
	DurationHours float64   `json:"duration_hours"`
	State         string    `json:"state"`
	CompanyID     int64     `json:"company_id"`
}

func (entry *WorkEntry) Validate() error {
	if entry.Name == "" || entry.EmployeeID <= 0 || entry.CompanyID <= 0 || entry.DateStart.IsZero() || !entry.DateStop.After(entry.DateStart) {
		return platformerrors.Validation("work entry has invalid required fields", nil)
	}
	entry.DurationHours = entry.DateStop.Sub(entry.DateStart).Hours()
	if entry.WorkEntryType == "" {
		entry.WorkEntryType = "attendance"
	}
	if entry.State == "" {
		entry.State = "draft"
	}
	return nil
}
