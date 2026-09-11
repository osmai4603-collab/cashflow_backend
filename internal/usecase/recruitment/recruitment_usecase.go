package recruitmentusecase

import (
	"context"

	"cashflow_backend/internal/domain/recruitment"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo          recruitment.Repository
	employeeMaker recruitment.EmployeeCreator
}

func New(repo recruitment.Repository, employeeMaker recruitment.EmployeeCreator) *UseCase {
	return &UseCase{repo: repo, employeeMaker: employeeMaker}
}

func (useCase *UseCase) CreateStage(ctx context.Context, stage *recruitment.RecruitmentStage) (*recruitment.RecruitmentStage, error) {
	if err := stage.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateStage(ctx, stage); err != nil {
		return nil, err
	}
	return stage, nil
}

func (useCase *UseCase) ListStages(ctx context.Context, companyID int64) ([]recruitment.RecruitmentStage, error) {
	return useCase.repo.ListStages(ctx, companyID)
}

func (useCase *UseCase) CreateApplicant(ctx context.Context, applicant *recruitment.Applicant) (*recruitment.Applicant, error) {
	if err := applicant.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateApplicant(ctx, applicant); err != nil {
		return nil, err
	}
	return applicant, nil
}

func (useCase *UseCase) GetApplicant(ctx context.Context, id int64) (*recruitment.Applicant, error) {
	return useCase.repo.GetApplicant(ctx, id)
}
func (useCase *UseCase) ListApplicants(ctx context.Context, companyID int64) ([]recruitment.Applicant, error) {
	return useCase.repo.ListApplicants(ctx, companyID)
}

func (useCase *UseCase) MoveApplicant(ctx context.Context, id, stageID int64) (*recruitment.Applicant, error) {
	applicant, err := useCase.repo.GetApplicant(ctx, id)
	if err != nil {
		return nil, err
	}
	if stageID <= 0 {
		return nil, platformerrors.Validation("stage_id must be positive", nil)
	}
	applicant.StageID = stageID
	if err := useCase.repo.UpdateApplicant(ctx, applicant); err != nil {
		return nil, err
	}
	return applicant, nil
}

func (useCase *UseCase) CreateInterview(ctx context.Context, interview *recruitment.ApplicantInterview) (*recruitment.ApplicantInterview, error) {
	if err := interview.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateInterview(ctx, interview); err != nil {
		return nil, err
	}
	return interview, nil
}

func (useCase *UseCase) Hire(ctx context.Context, applicantID int64) (int64, error) {
	if useCase.employeeMaker == nil {
		return 0, platformerrors.NotImplemented("employee creation is not configured")
	}
	applicant, err := useCase.repo.GetApplicant(ctx, applicantID)
	if err != nil {
		return 0, err
	}
	if applicant.EmployeeID != nil {
		return 0, platformerrors.Conflict("applicant is already hired")
	}
	employeeID, err := useCase.employeeMaker.CreateEmployeeFromApplicant(ctx, applicant)
	if err != nil {
		return 0, err
	}
	applicant.EmployeeID = &employeeID
	if err := useCase.repo.UpdateApplicant(ctx, applicant); err != nil {
		return 0, err
	}
	return employeeID, nil
}
