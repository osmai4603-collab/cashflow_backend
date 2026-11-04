package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/features/accounting"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PostgresFXRateProvider struct {
	pool *pgxpool.Pool
}

func NewPostgresFXRateProvider(pool *pgxpool.Pool) *PostgresFXRateProvider {
	return &PostgresFXRateProvider{pool: pool}
}

var _ accounting.FXClosingRateProvider = (*PostgresFXRateProvider)(nil)

func (p *PostgresFXRateProvider) GetClosingRate(
	ctx context.Context,
	foreignCurrency, baseCurrency string,
	asOf time.Time,
) (decimal.Decimal, error) {
	foreignCurrency = strings.ToUpper(strings.TrimSpace(foreignCurrency))
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if foreignCurrency == "" || baseCurrency == "" {
		return decimal.Zero, errors.New("foreign and base currencies are required")
	}
	if foreignCurrency == baseCurrency {
		return decimal.NewFromInt(1), nil
	}
	rateDate := asOf.UTC().Format("2006-01-02")
	const query = `
		SELECT rate
		FROM currency_exchange_rates
		WHERE currency = $1 AND base_currency = $2 AND rate_date <= $3
		ORDER BY rate_date DESC
		LIMIT 1`
	var rate decimal.Decimal
	err := p.pool.QueryRow(ctx, query, foreignCurrency, baseCurrency, rateDate).Scan(&rate)
	if errors.Is(err, pgx.ErrNoRows) {
		var inverse decimal.Decimal
		err = p.pool.QueryRow(ctx, query, baseCurrency, foreignCurrency, rateDate).Scan(&inverse)
		if err == nil {
			rate = decimal.NewFromInt(1).Div(inverse)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, fmt.Errorf("%w: no exchange rate defined for currency %s on date %s", accounting.ErrExchangeRateMissing, foreignCurrency, rateDate)
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("lookup exchange rate %s/%s on %s: %w", foreignCurrency, baseCurrency, rateDate, err)
	}
	if !rate.IsPositive() {
		return decimal.Zero, fmt.Errorf("exchange rate %s/%s on %s must be positive", foreignCurrency, baseCurrency, rateDate)
	}
	return rate, nil
}
