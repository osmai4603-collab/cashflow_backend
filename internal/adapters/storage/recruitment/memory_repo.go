package recruitmentstorage

import (
	"context"
	"fmt"
	"sync"

	"cashflow_backend/internal/domain/recruitment"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type MemoryRepo struct {
	mu            sync.RWMutex
	stages        map[int64]*recruitment.RecruitmentStage
	applicants    map[int64]*recruitment.Applicant
	interviews    map[int64]*recruitment.ApplicantInterview
	nextStage     int64
	nextApplicant int64
	nextInterview int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{stages: make(map[int64]*recruitment.RecruitmentStage), applicants: make(map[int64]*recruitment.Applicant), interviews: make(map[int64]*recruitment.ApplicantInterview)}
}

func (repo *MemoryRepo) CreateStage(ctx context.Context, stage *recruitment.RecruitmentStage) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextStage++
	stage.ID = repo.nextStage
	clone := *stage
	repo.stages[stage.ID] = &clone
	return nil
}

func (repo *MemoryRepo) ListStages(ctx context.Context, companyID int64) ([]recruitment.RecruitmentStage, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]recruitment.RecruitmentStage, 0)
	for _, stage := range repo.stages {
		if stage.CompanyID == companyID {
			result = append(result, *stage)
		}
	}
	return result, nil
}

func (repo *MemoryRepo) CreateApplicant(ctx context.Context, applicant *recruitment.Applicant) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextApplicant++
	applicant.ID = repo.nextApplicant
	clone := *applicant
	repo.applicants[applicant.ID] = &clone
	return nil
}

func (repo *MemoryRepo) GetApplicant(ctx context.Context, id int64) (*recruitment.Applicant, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	applicant, ok := repo.applicants[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("applicant %d not found", id))
	}
	clone := *applicant
	return &clone, nil
}

func (repo *MemoryRepo) UpdateApplicant(ctx context.Context, applicant *recruitment.Applicant) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, ok := repo.applicants[applicant.ID]; !ok {
		return platformerrors.NotFound("applicant not found")
	}
	clone := *applicant
	repo.applicants[applicant.ID] = &clone
	return nil
}

func (repo *MemoryRepo) ListApplicants(ctx context.Context, companyID int64) ([]recruitment.Applicant, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]recruitment.Applicant, 0)
	for _, applicant := range repo.applicants {
		if applicant.CompanyID == companyID {
			result = append(result, *applicant)
		}
	}
	return result, nil
}

func (repo *MemoryRepo) CreateInterview(ctx context.Context, interview *recruitment.ApplicantInterview) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextInterview++
	interview.ID = repo.nextInterview
	clone := *interview
	repo.interviews[interview.ID] = &clone
	return nil
}
