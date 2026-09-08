package crm

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// Stage represents a pipeline progression milestone (crm.stage in Odoo).
type Stage struct {
	ID           int64     `json:"id"`
	Name         i18n.TranslationString    `json:"name"`
	Sequence     int       `json:"sequence"`
	IsWon        bool      `json:"is_won"`
	IsClosed     bool      `json:"is_closed"`
	Fold         bool      `json:"fold"`
	Requirements string    `json:"requirements,omitempty"`
	CompanyID    *int64    `json:"company_id,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedBy    *int64    `json:"created_by,omitempty"`
	UpdatedBy    *int64    `json:"updated_by,omitempty"`
}

// Validate checks whether stage definition meets system requirements.
func (s *Stage) Validate() error {
	if len(s.Name) == 0 {
		return platformerrors.Validation("stage name is required", nil)
	}
	return nil
}
