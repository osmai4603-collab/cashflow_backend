package recruitment

import (
	"net/mail"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type Applicant struct {
	ID              int64     `json:"id"`
	PartnerName     string    `json:"partner_name"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	JobID           int64     `json:"job_id"`
	DepartmentID    *int64    `json:"department_id,omitempty"`
	StageID         int64     `json:"stage_id"`
	RecruiterUserID *int64    `json:"recruiter_user_id,omitempty"`
	Priority        int       `json:"priority"`
	SalaryExpected  float64   `json:"salary_expected"`
	SalaryProposed  float64   `json:"salary_proposed"`
	Availability    time.Time `json:"availability,omitempty"`
	RefusalReason   string    `json:"refusal_reason,omitempty"`
	ResumeURL       string    `json:"resume_url,omitempty"`
	EmployeeID      *int64    `json:"employee_id,omitempty"`
	CompanyID       int64     `json:"company_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (applicant *Applicant) Validate() error {
	applicant.PartnerName = strings.TrimSpace(applicant.PartnerName)
	applicant.Email = strings.TrimSpace(applicant.Email)
	if applicant.PartnerName == "" || applicant.JobID <= 0 || applicant.StageID <= 0 || applicant.CompanyID <= 0 {
		return platformerrors.Validation("applicant requires name, job, stage, and company", nil)
	}
	if _, err := mail.ParseAddress(applicant.Email); err != nil {
		return platformerrors.Validation("applicant email is invalid", nil)
	}
	if applicant.Priority < 0 || applicant.Priority > 3 || applicant.SalaryExpected < 0 || applicant.SalaryProposed < 0 {
		return platformerrors.Validation("invalid applicant priority or salary", nil)
	}
	return nil
}
