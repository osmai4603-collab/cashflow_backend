package project

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

type TaskTag struct {
	ID        int64     `json:"id"`
	Name      i18n.TranslationString    `json:"name"`
	Color     int       `json:"color"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (t *TaskTag) Validate() error {
	if len(t.Name) == 0 {
		return platformerrors.Validation("task tag name is required", nil)
	}
	return nil
}
