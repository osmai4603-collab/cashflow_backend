package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type UpdateDraftMoveUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewUpdateDraftMoveUseCase(repo AccountingRepository, logger *slog.Logger) *UpdateDraftMoveUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &UpdateDraftMoveUseCase{repo: repo, logger: logger}
}

func (uc *UpdateDraftMoveUseCase) Execute(
	ctx context.Context,
	moveID uuid.UUID,
	request CreateMoveRequest,
) (*MoveResponse, error) {
	if moveID == uuid.Nil {
		return nil, errors.New("move id is required")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	move, err := uc.repo.GetMoveByID(ctx, moveID)
	if err != nil {
		return nil, fmt.Errorf("get draft move: %w", err)
	}
	if err := move.ValidateCanModify(); err != nil {
		return nil, err
	}
	date, err := parseDate(request.Date, "date")
	if err != nil {
		return nil, err
	}
	journal, err := uc.repo.GetJournalByID(ctx, request.JournalID)
	if err != nil {
		return nil, fmt.Errorf("get journal: %w", err)
	}
	if journal == nil || !journal.IsActive {
		return nil, errors.New("journal must exist and be active")
	}
	period, err := uc.repo.GetPeriodByDate(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("get fiscal period: %w", err)
	}
	if period == nil {
		return nil, ErrPeriodNotFound
	}
	if err := period.CanPost(); err != nil {
		return nil, err
	}
	lines := make([]AccountMoveLine, len(request.Lines))
	for i, requestedLine := range request.Lines {
		account, err := uc.repo.GetAccountByID(ctx, requestedLine.AccountID)
		if err != nil {
			return nil, fmt.Errorf("get account for move line %d: %w", i+1, err)
		}
		if account == nil || !account.IsActive {
			return nil, fmt.Errorf("account %s on move line %d must be active", requestedLine.AccountID, i+1)
		}
		lines[i] = AccountMoveLine{
			ID: uuid.New(), MoveID: move.ID, AccountID: requestedLine.AccountID,
			PartnerID: cloneUUID(requestedLine.PartnerID), Debit: requestedLine.Debit,
			Credit: requestedLine.Credit, AmountCurrency: requestedLine.AmountCurrency,
			CurrencyID: requestedLine.CurrencyID, CostCenterID: cloneUUID(requestedLine.CostCenterID),
			MatchingNumber: requestedLine.MatchingNumber, Date: date,
		}
	}
	move.JournalID, move.PeriodID, move.Date = journal.ID, period.ID, date
	move.Ref, move.Lines, move.UpdatedAt = request.Ref, lines, time.Now().UTC()
	if err := uc.repo.SaveDraftMove(ctx, move); err != nil {
		return nil, fmt.Errorf("save updated draft move: %w", err)
	}
	response := ToMoveResponse(move)
	return &response, nil
}
