package accounting

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type GetMoveUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewGetMoveUseCase(repo AccountingRepository, logger *slog.Logger) *GetMoveUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetMoveUseCase{repo: repo, logger: logger}
}

func (uc *GetMoveUseCase) Execute(ctx context.Context, moveID uuid.UUID) (*MoveResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "get_move")
	move, err := uc.repo.GetMoveByID(ctx, moveID)
	if err != nil {
		return nil, fmt.Errorf("get move: %w", err)
	}
	if move == nil {
		return nil, ErrMoveNotFound
	}

	response := ToMoveResponse(move)
	logger.DebugContext(ctx, "move retrieved", slog.String("move_id", moveID.String()))
	return &response, nil
}
