package planningusecase

import (
	"cashflow_backend/internal/domain/planning"
)

type UseCase struct {
	repo planning.Repository
}

func New(repo planning.Repository) *UseCase {
	return &UseCase{repo: repo}
}
