package hr

import (
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// Job represents a job position or title within an organization (hr.job in Odoo).
type Job struct {
	ID                int64        `json:"id"`
	Name              i18n.TranslationString       `json:"name"` // Job Title e.g. "Senior Software Engineer"
	DepartmentID      *int64       `json:"department_id,omitempty"`
	Description       string       `json:"description,omitempty"`
	ExpectedEmployees int          `json:"expected_employees"`
	NoOfEmployee      int          `json:"no_of_employee"`
	CompanyID         *int64       `json:"company_id,omitempty"`
	Active            bool         `json:"active"`
	Audit             audit.Fields `json:"audit"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	CreatedBy         *int64       `json:"created_by,omitempty"`
	UpdatedBy         *int64       `json:"updated_by,omitempty"`
}

// Validate verifies constraints for the Job entity.
func (j *Job) Validate() error {
	if len(j.Name) == 0 {
		return platformerrors.Validation("job name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if j.ExpectedEmployees < 0 {
		return platformerrors.Validation("invalid expected employees", map[string]string{
			"expected_employees": "must be greater than or equal to 0",
		})
	}

	return nil
}
