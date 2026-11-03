package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CreateDraftMoveUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewCreateDraftMoveUseCase(repo AccountingRepository, logger *slog.Logger) *CreateDraftMoveUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &CreateDraftMoveUseCase{repo: repo, logger: logger}
}

func (uc *CreateDraftMoveUseCase) Execute(ctx context.Context, request CreateMoveRequest) (*MoveResponse, error) {
	return uc.ExecuteAs(ctx, uuid.Nil, request)
}

func (uc *CreateDraftMoveUseCase) ExecuteAs(ctx context.Context, actorID uuid.UUID, request CreateMoveRequest) (*MoveResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "create_draft_move")
	if err := request.Validate(); err != nil {
		return nil, err
	}
	moveDate, err := parseDate(request.Date, "date")
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
	period, err := uc.repo.GetPeriodByDate(ctx, moveDate)
	if err != nil {
		return nil, fmt.Errorf("get fiscal period: %w", err)
	}
	if period == nil {
		return nil, ErrPeriodNotFound
	}
	if err := period.CanPost(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	move := &AccountMove{
		ID:        uuid.New(),
		JournalID: journal.ID,
		PeriodID:  period.ID,
		Date:      moveDate,
		State:     MoveStateDraft,
		Ref:       request.Ref,
		CreatedBy: uuidPointer(actorID),
		CreatedAt: now,
		Lines:     make([]AccountMoveLine, len(request.Lines)),
	}
	checkedAccounts := make(map[uuid.UUID]*Account)
	for i, requestedLine := range request.Lines {
		account, ok := checkedAccounts[requestedLine.AccountID]
		if !ok {
			account, err = uc.repo.GetAccountByID(ctx, requestedLine.AccountID)
			if err != nil {
				return nil, fmt.Errorf("get account for move line %d: %w", i+1, err)
			}
			if account == nil || !account.IsActive {
				return nil, fmt.Errorf("account %s on move line %d must be active", requestedLine.AccountID, i+1)
			}
			checkedAccounts[requestedLine.AccountID] = account
		}
		move.Lines[i] = AccountMoveLine{
			ID:             uuid.New(),
			MoveID:         move.ID,
			AccountID:      requestedLine.AccountID,
			PartnerID:      cloneUUID(requestedLine.PartnerID),
			Debit:          requestedLine.Debit,
			Credit:         requestedLine.Credit,
			AmountCurrency: requestedLine.AmountCurrency,
			CurrencyID:     requestedLine.CurrencyID,
			CostCenterID:   cloneUUID(requestedLine.CostCenterID),
			MatchingNumber: requestedLine.MatchingNumber,
			Date:           moveDate,
			CreatedAt:      now,
		}
	}
	if err := uc.repo.SaveDraftMove(ctx, move); err != nil {
		logger.ErrorContext(ctx, "draft move creation failed", slog.String("move_id", move.ID.String()), slog.Any("error", err))
		return nil, fmt.Errorf("save draft move: %w", err)
	}

	logger.InfoContext(ctx, "draft move created", slog.String("move_id", move.ID.String()))
	response := ToMoveResponse(move)
	return &response, nil
}
