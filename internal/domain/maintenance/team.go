package maintenance

import (
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Team groups technicians that own a set of equipment and requests.
type Team struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	Color     int          `json:"color"`
	Active    bool         `json:"active"`
	MemberIDs []int64      `json:"member_ids,omitempty"`
	CompanyID *int64       `json:"company_id,omitempty"` // nil = shared team
	Audit     audit.Fields `json:"audit"`
}

// Validate ensures a team has a name.
func (t *Team) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return platformerrors.Validation("team name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	return nil
}

// ContainsMember reports whether userID is part of the team.
func (t *Team) ContainsMember(userID int64) bool {
	for _, id := range t.MemberIDs {
		if id == userID {
			return true
		}
	}
	return false
}