package crm

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// LostReason represents a formal justification why an opportunity was lost (crm.lost.reason in Odoo).
type LostReason struct {
	ID        int64     `json:"id"`
	Name      i18n.TranslationString    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	UpdatedBy *int64    `json:"updated_by,omitempty"`
}

// Validate checks whether the lost reason satisfies model constraints.
func (r *LostReason) Validate() error {
	if len(r.Name) == 0 {
		return platformerrors.Validation("lost reason name is required", nil)
	}
	return nil
}
