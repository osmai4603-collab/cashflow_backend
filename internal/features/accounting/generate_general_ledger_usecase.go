package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GenerateGeneralLedgerUseCase struct {
	repo AccountingRepository
}

func NewGenerateGeneralLedgerUseCase(repo AccountingRepository) *GenerateGeneralLedgerUseCase {
	return &GenerateGeneralLedgerUseCase{repo: repo}
}

func (uc *GenerateGeneralLedgerUseCase) Execute(
	ctx context.Context,
	accountID uuid.UUID,
	fromDate, toDate time.Time,
) ([]GeneralLedgerLineResponse, error) {
	if accountID == uuid.Nil {
		return nil, fmt.Errorf("account_id is required")
	}
	if fromDate.After(toDate) {
		return nil, fmt.Errorf("from date must not be after to date")
	}
	lines, err := uc.repo.GetGeneralLedgerLines(ctx, accountID, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get general ledger lines: %w", err)
	}
	responses := make([]GeneralLedgerLineResponse, len(lines))
	for i, line := range lines {
		responses[i] = ToGeneralLedgerLineResponse(line)
	}
	return responses, nil
}
