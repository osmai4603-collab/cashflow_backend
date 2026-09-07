package project

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type Milestone struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	ProjectID    int64      `json:"project_id"`
	DateDeadline *time.Time `json:"date_deadline,omitempty"`
	IsReached    bool       `json:"is_reached"`
	ReachedDate  *time.Time `json:"reached_date,omitempty"`
	Sequence     int        `json:"sequence"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (m *Milestone) Validate() error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return platformerrors.Validation("milestone name is required", nil)
	}
	if m.ProjectID <= 0 {
		return platformerrors.Validation("project_id is required", nil)
	}
	return nil
}

func (m *Milestone) Reach(now time.Time, hasOpenTasks bool) error {
	if hasOpenTasks {
		return platformerrors.Conflict("milestone cannot be reached while linked tasks are open")
	}
	m.IsReached = true
	now = now.UTC()
	m.ReachedDate = &now
	return nil
}
