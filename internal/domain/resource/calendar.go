package resource

import "cashflow_backend/internal/platform/errors"

type Calendar struct {
	ID                    int64   `json:"id"`
	Name                  string  `json:"name"`
	HoursPerDay           float64 `json:"hours_per_day"`
	FullTimeRequiredHours float64 `json:"full_time_required_hours"`
	CompanyID             int64   `json:"company_id"`
	Active                bool    `json:"active"`
}

func (calendar *Calendar) Validate() error {
	if calendar.Name == "" || calendar.HoursPerDay <= 0 || calendar.HoursPerDay > 24 || calendar.FullTimeRequiredHours <= 0 || calendar.CompanyID <= 0 {
		return errors.Validation("resource calendar has invalid hours or references", nil)
	}
	return nil
}
