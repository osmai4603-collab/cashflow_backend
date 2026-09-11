package recruitment

import "cashflow_backend/internal/platform/errors"

type JobRecruitmentState string

const (
	RecruitStateRecruiting JobRecruitmentState = "recruiting"
	RecruitStateClosed     JobRecruitmentState = "closed"
)

type JobPosition struct {
	ID                int64               `json:"id"`
	Name              string              `json:"name"`
	DepartmentID      *int64              `json:"department_id,omitempty"`
	ExpectedEmployees int                 `json:"expected_employees"`
	State             JobRecruitmentState `json:"state"`
	CompanyID         int64               `json:"company_id"`
}

func (position *JobPosition) Validate() error {
	if position.Name == "" || position.CompanyID <= 0 || position.ExpectedEmployees < 0 {
		return errors.Validation("job position requires name, company, and non-negative headcount", nil)
	}
	if position.State == "" {
		position.State = RecruitStateRecruiting
	}
	if position.State != RecruitStateRecruiting && position.State != RecruitStateClosed {
		return errors.Validation("invalid recruitment state", nil)
	}
	return nil
}
