package project

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type TaskTag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     int       `json:"color"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (t *TaskTag) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return platformerrors.Validation("task tag name is required", nil)
	}
	if len(t.Name) > 64 {
		return platformerrors.Validation("task tag name cannot exceed 64 characters", nil)
	}
	return nil
}
