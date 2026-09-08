package crm

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// Tag represents a categorization label for CRM records (crm.tag in Odoo).
type Tag struct {
	ID        int64     `json:"id"`
	Name      i18n.TranslationString    `json:"name"`
	Color     int       `json:"color"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks whether tag satisfies constraints.
func (t *Tag) Validate() error {
	if len(t.Name) == 0 {
		return platformerrors.Validation("tag name is required", nil)
	}
	return nil
}
