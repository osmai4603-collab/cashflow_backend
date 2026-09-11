package helpdesk

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Team represents a helpdesk support group (helpdesk.team).
type Team struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CompanyID int64  `json:"company_id"`
	Active    bool   `json:"active"`
}

func (t *Team) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return platformerrors.Validation("team name is required", nil)
	}
	if t.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	return nil
}
