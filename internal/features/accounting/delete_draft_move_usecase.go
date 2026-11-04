package accounting

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type DeleteDraftMoveUseCase struct {
	repo AccountingRepository
}

func NewDeleteDraftMoveUseCase(repo AccountingRepository) *DeleteDraftMoveUseCase {
	return &DeleteDraftMoveUseCase{repo: repo}
}

func (uc *DeleteDraftMoveUseCase) Execute(ctx context.Context, moveID uuid.UUID) error {
	if moveID == uuid.Nil {
		return fmt.Errorf("move id is required")
	}
	if err := uc.repo.DeleteDraftMove(ctx, moveID); err != nil {
		return fmt.Errorf("delete draft move: %w", err)
	}
	return nil
}
