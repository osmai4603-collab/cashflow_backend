package accounting

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/platform/money"

	"github.com/google/uuid"
)

type fxBalanceRepository interface {
	GetFXMonetaryBalances(ctx context.Context, asOf time.Time) ([]*FXMonetaryBalance, error)
}

// FXRevaluationSettings contains the accounts and journal used for generated
// unrealized foreign-exchange adjustments.
type FXRevaluationSettings struct {
	JournalID     uuid.UUID
	GainAccountID uuid.UUID
	LossAccountID uuid.UUID
	BaseCurrency  string
}

// RepositoryFXRevaluationProvider combines persisted ledger balances with
// closing rates from an injected rate source.
type RepositoryFXRevaluationProvider struct {
	repository fxBalanceRepository
	rates      FXClosingRateProvider
	settings   FXRevaluationSettings
}

func NewRepositoryFXRevaluationProvider(
	repository fxBalanceRepository,
	rates FXClosingRateProvider,
	settings FXRevaluationSettings,
) (*RepositoryFXRevaluationProvider, error) {
	if repository == nil {
		return nil, errors.New("FX balance repository is required")
	}
	if rates == nil {
		return nil, errors.New("FX closing rate provider is required")
	}
	if settings.JournalID == uuid.Nil || settings.GainAccountID == uuid.Nil || settings.LossAccountID == uuid.Nil {
		return nil, errors.New("FX journal, gain account, and loss account are required")
	}
	if settings.GainAccountID == settings.LossAccountID {
		return nil, errors.New("FX gain and loss accounts must be different")
	}
	baseCurrency := strings.ToUpper(strings.TrimSpace(settings.BaseCurrency))
	if _, ok := money.SupportedCurrencies[baseCurrency]; !ok {
		return nil, fmt.Errorf("unsupported FX base currency %q", settings.BaseCurrency)
	}
	settings.BaseCurrency = baseCurrency
	return &RepositoryFXRevaluationProvider{repository: repository, rates: rates, settings: settings}, nil
}

func (p *RepositoryFXRevaluationProvider) GetFXRevaluationSnapshot(
	ctx context.Context,
	asOf time.Time,
) (*FXRevaluationSnapshot, error) {
	if asOf.IsZero() {
		return nil, errors.New("FX revaluation date is required")
	}
	asOf = time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)
	balances, err := p.repository.GetFXMonetaryBalances(ctx, asOf)
	if err != nil {
		return nil, fmt.Errorf("get FX monetary balances: %w", err)
	}
	snapshot := &FXRevaluationSnapshot{
		JournalID:     p.settings.JournalID,
		GainAccountID: p.settings.GainAccountID,
		LossAccountID: p.settings.LossAccountID,
		BaseCurrency:  p.settings.BaseCurrency,
	}
	for i, balance := range balances {
		if balance == nil {
			return nil, fmt.Errorf("FX balance repository returned a nil balance at index %d", i)
		}
		currency := strings.ToUpper(strings.TrimSpace(balance.CurrencyID))
		if _, ok := money.SupportedCurrencies[currency]; !ok {
			return nil, fmt.Errorf("FX balance %d has unsupported currency %q", i, balance.CurrencyID)
		}
		if currency == p.settings.BaseCurrency || balance.ForeignAmount.IsZero() {
			continue
		}
		rate, err := p.rates.GetClosingRate(ctx, currency, p.settings.BaseCurrency, asOf)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s closing rate for %s: %w", currency, p.settings.BaseCurrency, asOf.Format(dateLayout), err)
		}
		if !rate.IsPositive() {
			return nil, fmt.Errorf("closing rate for %s/%s must be positive", currency, p.settings.BaseCurrency)
		}
		snapshot.MonetaryPositions = append(snapshot.MonetaryPositions, FXMonetaryPosition{
			AccountID:                balance.AccountID,
			AccountType:              balance.AccountType,
			PartnerID:                cloneUUID(balance.PartnerID),
			CostCenterID:             cloneUUID(balance.CostCenterID),
			CurrencyID:               currency,
			ForeignAmount:            balance.ForeignAmount,
			FunctionalCarryingAmount: balance.FunctionalCarryingAmount,
			ClosingRate:              rate,
		})
	}
	return snapshot, nil
}

var _ FXRevaluationProvider = (*RepositoryFXRevaluationProvider)(nil)
