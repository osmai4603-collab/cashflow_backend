package helpdesk

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Stage represents a step in the ticket lifecycle (helpdesk.stage).
type Stage struct {
	ID        int64  `json:"id"`
	TeamID    int64  `json:"team_id"`
	Name      string `json:"name"`
	Sequence  int    `json:"sequence"`
	IsClosed  bool   `json:"is_closed"`
	CompanyID int64  `json:"company_id"`
}

func (s *Stage) Validate() error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return platformerrors.Validation("stage name is required", nil)
	}
	if s.TeamID <= 0 {
		return platformerrors.Validation("team_id is required", nil)
	}
	return nil
}
