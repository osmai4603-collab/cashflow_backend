package project

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type ProjectStage struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Sequence  int       `json:"sequence"`
	Fold      bool      `json:"fold"`
	Color     int       `json:"color"`
	CompanyID *int64    `json:"company_id,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaskStage struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Sequence  int       `json:"sequence"`
	Fold      bool      `json:"fold"`
	Color     int       `json:"color"`
	Active    bool      `json:"active"`
	CompanyID *int64    `json:"company_id,omitempty"`
	ProjectIDs []int64  `json:"project_ids,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func validateStageName(name *string) error {
	*name = strings.TrimSpace(*name)
	if *name == "" {
		return platformerrors.Validation("stage name is required", nil)
	}
	if len(*name) > 128 {
		return platformerrors.Validation("stage name cannot exceed 128 characters", nil)
	}
	return nil
}

func (s *ProjectStage) Validate() error { return validateStageName(&s.Name) }
func (s *TaskStage) Validate() error    { return validateStageName(&s.Name) }
