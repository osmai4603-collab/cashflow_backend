package hr

import (
	"net/mail"
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Gender values
const (
	GenderMale   = "male"
	GenderFemale = "female"
	GenderOther  = "other"
)

// MaritalStatus values
const (
	MaritalSingle     = "single"
	MaritalMarried    = "married"
	MaritalCohabitant = "cohabitant"
	MaritalWidower    = "widower"
	MaritalDivorced   = "divorced"
)

// Employee represents a company employee (hr.employee in Odoo).
type Employee struct {
	ID               int64        `json:"id"`
	Name             string       `json:"name"`
	PartnerID        *int64       `json:"partner_id,omitempty"` // linked partner in res_partners
	DepartmentID     *int64       `json:"department_id,omitempty"`
	JobID            *int64       `json:"job_id,omitempty"`
	JobTitle         string       `json:"job_title,omitempty"`
	ManagerID        *int64       `json:"manager_id,omitempty"` // direct supervisor
	WorkEmail        string       `json:"work_email,omitempty"`
	WorkPhone        string       `json:"work_phone,omitempty"`
	WorkLocation     string       `json:"work_location,omitempty"`
	HireDate         *time.Time   `json:"hire_date,omitempty"`
	Gender           string       `json:"gender,omitempty"`
	MaritalStatus    string       `json:"marital_status,omitempty"`
	IdentificationID string       `json:"identification_id,omitempty"`
	BankAccountNo    string       `json:"bank_account_no,omitempty"`
	ExpenseManagerID *int64       `json:"expense_manager_id,omitempty"`
	CompanyID        *int64       `json:"company_id,omitempty"`
	Active           bool         `json:"active"`
	OvertimeEmployeeThreshold int `json:"overtime_employee_threshold"`
	Audit            audit.Fields `json:"audit"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	CreatedBy        *int64       `json:"created_by,omitempty"`
	UpdatedBy        *int64       `json:"updated_by,omitempty"`
}

// Validate checks business invariants for the Employee entity.
func (e *Employee) Validate() error {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		return platformerrors.Validation("employee name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if e.ManagerID != nil && e.ID > 0 && *e.ManagerID == e.ID {
		return platformerrors.Validation("invalid employee manager", map[string]string{
			"manager_id": "an employee cannot be their own manager",
		})
	}

	e.WorkEmail = strings.TrimSpace(e.WorkEmail)
	if e.WorkEmail != "" {
		if _, err := mail.ParseAddress(e.WorkEmail); err != nil {
			return platformerrors.Validation("invalid work email format", map[string]string{
				"work_email": "must be a valid email address",
			})
		}
	}

	if e.Gender == "" {
		e.Gender = GenderOther
	}
	if e.MaritalStatus == "" {
		e.MaritalStatus = MaritalSingle
	}

	return nil
}
