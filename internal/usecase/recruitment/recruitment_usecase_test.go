package recruitmentusecase

import (
	"context"
	"testing"

	recruitmentstorage "cashflow_backend/internal/adapters/storage/recruitment"
	"cashflow_backend/internal/domain/recruitment"
)

type employeeMakerStub struct{ id int64 }

func (stub employeeMakerStub) CreateEmployeeFromApplicant(ctx context.Context, applicant *recruitment.Applicant) (int64, error) {
	return stub.id, nil
}

func TestHireApplicantCreatesEmployeeLink(t *testing.T) {
	repo := recruitmentstorage.NewMemoryRepo()
	useCase := New(repo, employeeMakerStub{id: 42})
	applicant, err := useCase.CreateApplicant(context.Background(), &recruitment.Applicant{PartnerName: "Candidate", Email: "candidate@example.com", JobID: 1, StageID: 1, CompanyID: 1})
	if err != nil {
		t.Fatalf("create applicant: %v", err)
	}
	employeeID, err := useCase.Hire(context.Background(), applicant.ID)
	if err != nil || employeeID != 42 {
		t.Fatalf("hire result: id=%v error=%v", employeeID, err)
	}
	updated, err := useCase.GetApplicant(context.Background(), applicant.ID)
	if err != nil || updated.EmployeeID == nil || *updated.EmployeeID != 42 {
		t.Fatalf("missing employee link: %+v error=%v", updated, err)
	}
}
