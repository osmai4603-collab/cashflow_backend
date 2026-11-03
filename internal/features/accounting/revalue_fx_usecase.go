package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cashflow_backend/internal/platform/money"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FXRevaluationSnapshot is an as-of snapshot supplied by an FX valuation adapter.
type FXRevaluationSnapshot struct {
	JournalID         uuid.UUID
	GainAccountID     uuid.UUID
	LossAccountID     uuid.UUID
	BaseCurrency      string
	MonetaryPositions []FXMonetaryPosition
}

// FXMonetaryPosition represents signed foreign and functional-currency
// balances for one asset or liability account.
type FXMonetaryPosition struct {
	AccountID                uuid.UUID
	AccountType              AccountType
	PartnerID                *uuid.UUID
	CostCenterID             *uuid.UUID
	CurrencyID               string
	ForeignAmount            decimal.Decimal
	FunctionalCarryingAmount decimal.Decimal
	ClosingRate              decimal.Decimal
}

type RevalueFXUseCase struct {
	repo     AccountingRepository
	provider FXRevaluationProvider
	logger   *slog.Logger
}

func NewRevalueFXUseCase(repo AccountingRepository, provider FXRevaluationProvider, logger *slog.Logger) *RevalueFXUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &RevalueFXUseCase{repo: repo, provider: provider, logger: logger}
}

func (uc *RevalueFXUseCase) Execute(ctx context.Context, asOf time.Time) (*MoveResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "revalue_fx")
	if asOf.IsZero() {
		return nil, errors.New("FX revaluation date is required")
	}
	if uc.provider == nil {
		return nil, errors.New("FX revaluation provider is not configured")
	}
	asOf = time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)
	snapshot, err := uc.provider.GetFXRevaluationSnapshot(ctx, asOf)
	if err != nil {
		return nil, fmt.Errorf("get FX revaluation snapshot: %w", err)
	}
	if snapshot == nil {
		return nil, errors.New("FX revaluation provider returned a nil snapshot")
	}
	baseCurrency := strings.ToUpper(strings.TrimSpace(snapshot.BaseCurrency))
	baseSpec, ok := money.SupportedCurrencies[baseCurrency]
	if !ok {
		return nil, fmt.Errorf("unsupported FX base currency %q", snapshot.BaseCurrency)
	}
	journal, err := uc.repo.GetJournalByID(ctx, snapshot.JournalID)
	if err != nil {
		return nil, fmt.Errorf("get FX revaluation journal: %w", err)
	}
	if journal == nil || !journal.IsActive {
		return nil, errors.New("FX revaluation journal must exist and be active")
	}
	gainAccount, err := uc.repo.GetAccountByID(ctx, snapshot.GainAccountID)
	if err != nil {
		return nil, fmt.Errorf("get FX gain account: %w", err)
	}
	if gainAccount == nil || !gainAccount.IsActive || gainAccount.Type != AccountTypeIncome {
		return nil, errors.New("FX gain account must be active and have income type")
	}
	lossAccount, err := uc.repo.GetAccountByID(ctx, snapshot.LossAccountID)
	if err != nil {
		return nil, fmt.Errorf("get FX loss account: %w", err)
	}
	if lossAccount == nil || !lossAccount.IsActive || lossAccount.Type != AccountTypeExpense {
		return nil, errors.New("FX loss account must be active and have expense type")
	}
	if gainAccount.ID == lossAccount.ID {
		return nil, errors.New("FX gain and loss accounts must be different")
	}
	period, err := uc.repo.GetPeriodByDate(ctx, asOf)
	if err != nil {
		return nil, fmt.Errorf("get FX revaluation period: %w", err)
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
		Date:      asOf,
		State:     MoveStateDraft,
		Ref:       "FX revaluation as of " + asOf.Format(dateLayout),
		CreatedAt: now,
	}
	accounts := map[uuid.UUID]*Account{gainAccount.ID: gainAccount, lossAccount.ID: lossAccount}
	appendLine := func(accountID uuid.UUID, partnerID, costCenterID *uuid.UUID, debit, credit decimal.Decimal) {
		move.Lines = append(move.Lines, AccountMoveLine{
			ID:             uuid.New(),
			MoveID:         move.ID,
			AccountID:      accountID,
			PartnerID:      cloneUUID(partnerID),
			Debit:          debit,
			Credit:         credit,
			AmountCurrency: debit.Sub(credit),
			CurrencyID:     baseCurrency,
			CostCenterID:   cloneUUID(costCenterID),
			Date:           asOf,
			CreatedAt:      now,
		})
	}
	for i, position := range snapshot.MonetaryPositions {
		if position.AccountID == uuid.Nil || !position.ClosingRate.IsPositive() {
			return nil, fmt.Errorf("invalid FX monetary position at index %d", i)
		}
		if position.AccountType != AccountTypeAsset && position.AccountType != AccountTypeLiability {
			return nil, fmt.Errorf("FX position %d must be an asset or liability", i)
		}
		if position.AccountID == gainAccount.ID || position.AccountID == lossAccount.ID {
			return nil, fmt.Errorf("FX position %d cannot use a gain or loss account", i)
		}
		currency := strings.ToUpper(strings.TrimSpace(position.CurrencyID))
		if _, ok := money.SupportedCurrencies[currency]; !ok || currency == baseCurrency {
			return nil, fmt.Errorf("FX position %d has invalid foreign currency %q", i, position.CurrencyID)
		}
		account, ok := accounts[position.AccountID]
		if !ok {
			account, err = uc.repo.GetAccountByID(ctx, position.AccountID)
			if err != nil {
				return nil, fmt.Errorf("get FX position account %s: %w", position.AccountID, err)
			}
			accounts[position.AccountID] = account
		}
		if account == nil || !account.IsActive || account.Type != position.AccountType {
			return nil, fmt.Errorf("FX position account %s must be active and match its declared type", position.AccountID)
		}
		if position.ForeignAmount.IsZero() {
			continue
		}
		revalued := position.ForeignAmount.Mul(position.ClosingRate).Round(baseSpec.Exponent)
		delta := revalued.Sub(position.FunctionalCarryingAmount.Round(baseSpec.Exponent))
		if delta.IsZero() {
			continue
		}
		amount := delta.Abs()
		if delta.IsPositive() {
			appendLine(position.AccountID, position.PartnerID, position.CostCenterID, amount, decimal.Zero)
			appendLine(gainAccount.ID, position.PartnerID, position.CostCenterID, decimal.Zero, amount)
		} else {
			appendLine(lossAccount.ID, position.PartnerID, position.CostCenterID, amount, decimal.Zero)
			appendLine(position.AccountID, position.PartnerID, position.CostCenterID, decimal.Zero, amount)
		}
	}
	if len(move.Lines) == 0 {
		return nil, errors.New("FX revaluation produced no adjustments")
	}
	if err := move.ValidateBalance(); err != nil {
		return nil, fmt.Errorf("validate FX revaluation move: %w", err)
	}
	if err := uc.repo.SaveDraftMove(ctx, move); err != nil {
		logger.ErrorContext(ctx, "FX revaluation draft creation failed", slog.String("move_id", move.ID.String()), slog.Any("error", err))
		return nil, fmt.Errorf("save FX revaluation draft: %w", err)
	}
	logger.InfoContext(ctx, "FX revaluation draft created", slog.String("move_id", move.ID.String()), slog.Int("line_count", len(move.Lines)))
	response := ToMoveResponse(move)
	return &response, nil
}
