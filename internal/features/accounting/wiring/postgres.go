package wiring

import (
	"log/slog"

	"cashflow_backend/internal/features/accounting"
	accountingrepo "cashflow_backend/internal/features/accounting/internal"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresUseCases(pool *pgxpool.Pool, publisher accounting.EventPublisher, logger *slog.Logger) *accounting.UseCases {
	repo := accountingrepo.NewPostgresRepository(pool)
	rates := accountingrepo.NewPostgresFXRateProvider(pool)
	return accounting.NewUseCasesWithFXRateProvider(repo, publisher, rates, logger)
}

func NewPostgresUseCasesWithMakerChecker(
	pool *pgxpool.Pool,
	publisher accounting.EventPublisher,
	settings accounting.MakerCheckerSettings,
	logger *slog.Logger,
) (*accounting.UseCases, error) {
	repo := accountingrepo.NewPostgresRepository(pool)
	rates := accountingrepo.NewPostgresFXRateProvider(pool)
	useCases, err := accounting.NewUseCasesWithFXRatesAndMakerChecker(
		repo, publisher, rates, settings,
		logger,
	)
	if err != nil {
		return nil, err
	}
	return useCases, nil
}
