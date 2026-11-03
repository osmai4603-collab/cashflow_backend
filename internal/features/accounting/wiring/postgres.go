package wiring

import (
	"log/slog"

	"cashflow_backend/internal/features/accounting"
	accountingrepo "cashflow_backend/internal/features/accounting/internal"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresUseCases(pool *pgxpool.Pool, publisher accounting.EventPublisher, logger *slog.Logger) *accounting.UseCases {
	return accounting.NewUseCases(accountingrepo.NewPostgresRepository(pool), publisher, logger)
}

func NewPostgresUseCasesWithMakerChecker(
	pool *pgxpool.Pool,
	publisher accounting.EventPublisher,
	settings accounting.MakerCheckerSettings,
	logger *slog.Logger,
) (*accounting.UseCases, error) {
	useCases, err := accounting.NewUseCasesWithMakerChecker(
		accountingrepo.NewPostgresRepository(pool), publisher, settings, logger,
	)
	if err != nil {
		return nil, err
	}
	return useCases, nil
}
