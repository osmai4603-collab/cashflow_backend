package accounting

import (
	"context"
	"fmt"
	"log/slog"

	platformlogger "cashflow_backend/internal/platform/logger"
)

type UseCases struct {
	CreateAccount        *CreateAccountUseCase
	ListAccounts         *ListAccountsUseCase
	ListAccountsTree     *ListAccountsTreeUseCase
	GetMove              *GetMoveUseCase
	CreateFiscalYear     *CreateFiscalYearUseCase
	ChangePeriodState    *ChangePeriodStateUseCase
	CreateDraftMove      *CreateDraftMoveUseCase
	PostMove             *PostMoveUseCase
	ReverseMove          *ReverseMoveUseCase
	RevalueFX            *RevalueFXUseCase
	GenerateTrialBalance *GenerateTrialBalanceUseCase
	IngestSubledgerEvent *IngestSubledgerEventUseCase
}

func NewUseCases(repo AccountingRepository, publisher EventPublisher, logger *slog.Logger) *UseCases {
	return NewUseCasesWithFXProvider(repo, publisher, nil, logger)
}

func NewUseCasesWithFXProvider(
	repo AccountingRepository,
	publisher EventPublisher,
	fxProvider FXRevaluationProvider,
	logger *slog.Logger,
) *UseCases {
	return newUseCases(repo, publisher, fxProvider, MakerCheckerSettings{FunctionalCurrency: "SAR"}, logger)
}

func NewUseCasesWithMakerChecker(
	repo AccountingRepository,
	publisher EventPublisher,
	settings MakerCheckerSettings,
	logger *slog.Logger,
) (*UseCases, error) {
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("configure maker-checker: %w", err)
	}
	return newUseCases(repo, publisher, nil, settings, logger), nil
}

func newUseCases(
	repo AccountingRepository,
	publisher EventPublisher,
	fxProvider FXRevaluationProvider,
	settings MakerCheckerSettings,
	logger *slog.Logger,
) *UseCases {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With(slog.String("feature", "accounting"))
	return &UseCases{
		CreateAccount:        NewCreateAccountUseCase(repo, logger),
		ListAccounts:         NewListAccountsUseCase(repo, logger),
		ListAccountsTree:     NewListAccountsTreeUseCase(repo, logger),
		GetMove:              NewGetMoveUseCase(repo, logger),
		CreateFiscalYear:     NewCreateFiscalYearUseCase(repo, logger),
		ChangePeriodState:    NewChangePeriodStateUseCase(repo, logger),
		CreateDraftMove:      NewCreateDraftMoveUseCase(repo, logger),
		PostMove:             newPostMoveUseCase(repo, publisher, settings, logger),
		ReverseMove:          NewReverseMoveUseCase(repo, publisher, logger),
		RevalueFX:            NewRevalueFXUseCase(repo, fxProvider, logger),
		GenerateTrialBalance: NewGenerateTrialBalanceUseCase(repo, logger),
		IngestSubledgerEvent: NewIngestSubledgerEventUseCase(repo, logger),
	}
}

func NewUseCasesWithFXRates(
	repo AccountingRepository,
	publisher EventPublisher,
	rates FXClosingRateProvider,
	settings FXRevaluationSettings,
	logger *slog.Logger,
) (*UseCases, error) {
	provider, err := NewRepositoryFXRevaluationProvider(repo, rates, settings)
	if err != nil {
		return nil, fmt.Errorf("configure FX revaluation provider: %w", err)
	}
	return NewUseCasesWithFXProvider(repo, publisher, provider, logger), nil
}

func useCaseLogger(ctx context.Context, fallback *slog.Logger, name string) *slog.Logger {
	return platformlogger.FromContextOr(ctx, fallback).With(slog.String("usecase", name))
}
