package accounting

import (
	"context"
	"time"
)

type VerifyAuditIntegrityUseCase struct {
	repo AccountingRepository
}

func NewVerifyAuditIntegrityUseCase(repo AccountingRepository) *VerifyAuditIntegrityUseCase {
	return &VerifyAuditIntegrityUseCase{repo: repo}
}

func (uc *VerifyAuditIntegrityUseCase) Execute(ctx context.Context) (*AuditIntegrityResponse, error) {
	ok, err := uc.repo.VerifyAuditIntegrity(ctx)
	now := time.Now().UTC()
	if err != nil {
		return &AuditIntegrityResponse{
			IsValid:   false,
			Reason:    err.Error(),
			CheckedAt: now,
		}, nil
	}
	return &AuditIntegrityResponse{
		IsValid:   ok,
		CheckedAt: now,
	}, nil
}

