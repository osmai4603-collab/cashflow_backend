package crm

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// LostReason represents a formal justification why an opportunity was lost (crm.lost.reason in Odoo).
type LostReason struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	UpdatedBy *int64    `json:"updated_by,omitempty"`
}

// Validate checks whether the lost reason satisfies model constraints.
func (r *LostReason) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return platformerrors.Validation("lost reason name is required", nil)
	}
	if len(r.Name) > 255 {
		return platformerrors.Validation("lost reason name cannot exceed 255 characters", nil)
	}
	return nil
}
