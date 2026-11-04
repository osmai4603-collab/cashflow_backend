package accounting

import (
	"context"
	"fmt"
	"log/slog"

	platformlogger "cashflow_backend/internal/platform/logger"
)

type UseCases struct {
	CreateAccount           *CreateAccountUseCase
	UpdateAccount           *UpdateAccountUseCase
	UpdateDraftMove         *UpdateDraftMoveUseCase
	DeleteDraftMove         *DeleteDraftMoveUseCase
	ListMoves               *ListMovesUseCase
	ListJournals            *ListJournalsUseCase
	ListFiscalYears         *ListFiscalYearsUseCase
	CloseFiscalYear         *CloseFiscalYearUseCase
	ListAccounts            *ListAccountsUseCase
	ListAccountsTree        *ListAccountsTreeUseCase
	GetMove                 *GetMoveUseCase
	CreateFiscalYear        *CreateFiscalYearUseCase
	ChangePeriodState       *ChangePeriodStateUseCase
	CreateDraftMove         *CreateDraftMoveUseCase
	PostMove                *PostMoveUseCase
	ReverseMove             *ReverseMoveUseCase
	RevalueFX               *RevalueFXUseCase
	GenerateTrialBalance    *GenerateTrialBalanceUseCase
	GenerateIncomeStatement *GenerateIncomeStatementUseCase
	GenerateBalanceSheet    *GenerateBalanceSheetUseCase
	GenerateCashFlow        *GenerateCashFlowUseCase
	GenerateGeneralLedger   *GenerateGeneralLedgerUseCase
	ReconcileMoveLines      *ReconcileMoveLinesUseCase
	UnreconcileMoveLines    *UnreconcileMoveLinesUseCase
	ListUnreconciledLines   *ListUnreconciledLinesUseCase
	IngestSubledgerEvent    *IngestSubledgerEventUseCase
}

func NewUseCases(repo AccountingRepository, publisher EventPublisher, logger *slog.Logger) *UseCases {
	return NewUseCasesWithFXProvider(repo, publisher, nil, logger)
}

func NewUseCasesWithFXRateProvider(
	repo AccountingRepository,
	publisher EventPublisher,
	rates FXClosingRateProvider,
	logger *slog.Logger,
) *UseCases {
	provider := &configuredFXRevaluationProvider{repository: repo, rates: rates, base: "SAR"}
	return newUseCases(
		repo, publisher, provider, rates,
		MakerCheckerSettings{FunctionalCurrency: "SAR"}, logger,
	)
}

func NewUseCasesWithFXProvider(
	repo AccountingRepository,
	publisher EventPublisher,
	fxProvider FXRevaluationProvider,
	logger *slog.Logger,
) *UseCases {
	return newUseCases(repo, publisher, fxProvider, nil, MakerCheckerSettings{FunctionalCurrency: "SAR"}, logger)
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
	return newUseCases(repo, publisher, nil, nil, settings, logger), nil
}

func newUseCases(
	repo AccountingRepository,
	publisher EventPublisher,
	fxProvider FXRevaluationProvider,
	rates FXClosingRateProvider,
	settings MakerCheckerSettings,
	logger *slog.Logger,
) *UseCases {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With(slog.String("feature", "accounting"))
	return &UseCases{
		CreateAccount:           NewCreateAccountUseCase(repo, logger),
		UpdateAccount:           NewUpdateAccountUseCase(repo, logger),
		UpdateDraftMove:         NewUpdateDraftMoveUseCase(repo, logger),
		DeleteDraftMove:         NewDeleteDraftMoveUseCase(repo),
		ListMoves:               NewListMovesUseCase(repo),
		ListJournals:            NewListJournalsUseCase(repo),
		ListFiscalYears:         NewListFiscalYearsUseCase(repo),
		CloseFiscalYear:         NewCloseFiscalYearUseCase(repo),
		ListAccounts:            NewListAccountsUseCase(repo, logger),
		ListAccountsTree:        NewListAccountsTreeUseCase(repo, logger),
		GetMove:                 NewGetMoveUseCase(repo, logger),
		CreateFiscalYear:        NewCreateFiscalYearUseCase(repo, logger),
		ChangePeriodState:       NewChangePeriodStateUseCase(repo, logger),
		CreateDraftMove:         NewCreateDraftMoveUseCase(repo, logger),
		PostMove:                newPostMoveUseCase(repo, publisher, settings, logger),
		ReverseMove:             NewReverseMoveUseCase(repo, publisher, logger),
		RevalueFX:               NewRevalueFXUseCase(repo, fxProvider, logger),
		GenerateTrialBalance:    NewGenerateTrialBalanceUseCase(repo, logger),
		GenerateIncomeStatement: NewGenerateIncomeStatementUseCase(repo),
		GenerateBalanceSheet:    NewGenerateBalanceSheetUseCase(repo),
		GenerateCashFlow:        NewGenerateCashFlowUseCase(repo),
		GenerateGeneralLedger:   NewGenerateGeneralLedgerUseCase(repo),
		ReconcileMoveLines:      NewReconcileMoveLinesUseCase(repo),
		UnreconcileMoveLines:    NewUnreconcileMoveLinesUseCase(repo),
		ListUnreconciledLines:   NewListUnreconciledLinesUseCase(repo),
		IngestSubledgerEvent:    NewIngestSubledgerEventUseCaseWithRates(repo, rates, settings.FunctionalCurrency, logger),
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
	return newUseCases(repo, publisher, provider, rates, MakerCheckerSettings{FunctionalCurrency: settings.BaseCurrency}, logger), nil
}

func NewUseCasesWithFXRatesAndMakerChecker(
	repo AccountingRepository,
	publisher EventPublisher,
	rates FXClosingRateProvider,
	settings MakerCheckerSettings,
	logger *slog.Logger,
) (*UseCases, error) {
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("configure maker-checker: %w", err)
	}
	provider, err := newConfiguredFXRevaluationProvider(repo, rates, settings.FunctionalCurrency)
	if err != nil {
		return nil, fmt.Errorf("configure FX revaluation provider: %w", err)
	}
	return newUseCases(repo, publisher, provider, rates, settings, logger), nil
}

func useCaseLogger(ctx context.Context, fallback *slog.Logger, name string) *slog.Logger {
	return platformlogger.FromContextOr(ctx, fallback).With(slog.String("usecase", name))
}
