package group

import (
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Group represents a role/group in the ERP RBAC model.
type Group struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	Category        string       `json:"category,omitempty"`
	Active          bool         `json:"active"`
	ImpliedGroupIDs []int64      `json:"implied_group_ids,omitempty"`
	Audit           audit.Fields `json:"audit"`
}

// Validate ensures the group entity satisfies the domain invariants.
func (g *Group) Validate() error {
	trimmedName := strings.TrimSpace(g.Name)
	if trimmedName == "" {
		return platformerrors.Validation("group name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if len(trimmedName) > 255 {
		return platformerrors.Validation("group name exceeds maximum length", map[string]string{
			"name": "must not exceed 255 characters",
		})
	}
	g.Name = trimmedName
	return nil
}
