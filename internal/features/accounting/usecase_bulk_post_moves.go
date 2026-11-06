package accounting

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type BulkPostMovesUseCase struct {
	postMoveUC *PostMoveUseCase
	logger     *slog.Logger
}

func NewBulkPostMovesUseCase(postMoveUC *PostMoveUseCase, logger *slog.Logger) *BulkPostMovesUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &BulkPostMovesUseCase{postMoveUC: postMoveUC, logger: logger}
}

func (uc *BulkPostMovesUseCase) Execute(ctx context.Context, actorID uuid.UUID, moveIDs []uuid.UUID) (*BulkPostMovesResponse, error) {
	if actorID == uuid.Nil {
		return nil, ErrAccountingActorRequired
	}
	if len(moveIDs) == 0 {
		return &BulkPostMovesResponse{PostedCount: 0, Moves: []MoveResponse{}}, nil
	}

	posted := make([]MoveResponse, 0, len(moveIDs))
	for _, id := range moveIDs {
		res, err := uc.postMoveUC.Execute(ctx, id, actorID)
		if err != nil {
			return nil, fmt.Errorf("post move %s: %w", id, err)
		}
		posted = append(posted, *res)
	}

	return &BulkPostMovesResponse{
		PostedCount: len(posted),
		Moves:       posted,
	}, nil
}

