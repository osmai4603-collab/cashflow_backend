package crm

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Tag represents a categorization label for CRM records (crm.tag in Odoo).
type Tag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     int       `json:"color"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks whether tag satisfies constraints.
func (t *Tag) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return platformerrors.Validation("tag name is required", nil)
	}
	if len(t.Name) > 64 {
		return platformerrors.Validation("tag name cannot exceed 64 characters", nil)
	}
	return nil
}
