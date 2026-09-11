package recruitment

import "context"

type Repository interface {
	CreateStage(ctx context.Context, stage *RecruitmentStage) error
	ListStages(ctx context.Context, companyID int64) ([]RecruitmentStage, error)
	CreateApplicant(ctx context.Context, applicant *Applicant) error
	GetApplicant(ctx context.Context, id int64) (*Applicant, error)
	UpdateApplicant(ctx context.Context, applicant *Applicant) error
	ListApplicants(ctx context.Context, companyID int64) ([]Applicant, error)
	CreateInterview(ctx context.Context, interview *ApplicantInterview) error
}

type EmployeeCreator interface {
	CreateEmployeeFromApplicant(ctx context.Context, applicant *Applicant) (int64, error)
}
