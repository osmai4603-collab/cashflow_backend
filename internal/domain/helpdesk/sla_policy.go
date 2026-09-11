package helpdesk

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// SLAPolicy defines targets for response and resolution times (helpdesk.sla).
type SLAPolicy struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	TeamID             int64   `json:"team_id"`
	Priority           string  `json:"priority"` // "0", "1", "2", "3"
	MaxHoursFirstResp  float64 `json:"max_hours_first_resp"`
	MaxHoursResolution float64 `json:"max_hours_resolution"`
	WorkingCalendarID  *int64  `json:"working_calendar_id,omitempty"`
	Active             bool    `json:"active"`
	CompanyID          int64   `json:"company_id"`
}

func (p *SLAPolicy) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return platformerrors.Validation("policy name is required", nil)
	}
	if p.TeamID <= 0 {
		return platformerrors.Validation("team_id is required", nil)
	}
	if p.MaxHoursFirstResp < 0 || p.MaxHoursResolution < 0 {
		return platformerrors.Validation("SLA hours cannot be negative", nil)
	}
	return nil
}
