package group

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Permission defines fine-grained access for a model/action pair.
type Permission struct {
	ID        int64  `json:"id"`
	GroupID   int64  `json:"group_id"`
	Model     string `json:"model"`
	CanRead   bool   `json:"can_read"`
	CanCreate bool   `json:"can_create"`
	CanUpdate bool   `json:"can_update"`
	CanDelete bool   `json:"can_delete"`
}

// Validate ensures the permission is well-formed.
func (p *Permission) Validate() error {
	p.Model = strings.TrimSpace(p.Model)
	if p.Model == "" {
		return platformerrors.Validation("permission model is required", map[string]string{
			"model": "cannot be empty",
		})
	}
	if p.GroupID <= 0 {
		return platformerrors.Validation("group is required for permission", map[string]string{
			"group_id": "must be a valid group",
		})
	}
	return nil
}

// Allows checks whether the permission grants an action.
func (p *Permission) Allows(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "read":
		return p.CanRead
	case "create":
		return p.CanCreate
	case "update":
		return p.CanUpdate
	case "delete":
		return p.CanDelete
	default:
		return false
	}
}
