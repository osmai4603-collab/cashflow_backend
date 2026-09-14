package examples

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NumberSource fetches the next atomic number with database locking (SELECT ... FOR UPDATE).
type NumberSource interface {
	NextValue(ctx context.Context, sequenceID string) (int64, error)
}

// SequenceConfig defines prefix, padding, and suffix for formatted codes.
type SequenceConfig struct {
	ID      string
	Prefix  string // Supports %(year)s, %(month)s, %(day)s
	Suffix  string
	Padding int
}

// SequenceEngine formats sequential identifiers.
type SequenceEngine struct {
	source NumberSource
}

func NewSequenceEngine(source NumberSource) *SequenceEngine {
	return &SequenceEngine{source: source}
}

func (e *SequenceEngine) Next(ctx context.Context, cfg SequenceConfig, ts time.Time) (string, error) {
	if e.source == nil {
		return "", errors.New("missing number source")
	}

	nextVal, err := e.source.NextValue(ctx, cfg.ID)
	if err != nil {
		return "", err
	}

	prefix := expandTokens(cfg.Prefix, ts)
	suffix := expandTokens(cfg.Suffix, ts)

	padWidth := cfg.Padding
	if padWidth < 1 {
		padWidth = 4
	}

	return fmt.Sprintf("%s%0*d%s", prefix, padWidth, nextVal, suffix), nil
}

func expandTokens(template string, ts time.Time) string {
	if template == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"%(year)s", fmt.Sprintf("%d", ts.Year()),
		"%(month)s", fmt.Sprintf("%02d", int(ts.Month())),
		"%(day)s", fmt.Sprintf("%02d", ts.Day()),
	)
	return replacer.Replace(template)
}

// ExchangeRate holds the rate relative to the company's anchor currency (e.g. USD = 1.0, SAR = 3.75).
type ExchangeRate struct {
	CurrencyCode string
	Rate         float64 // Rate > 0
}

// ExchangeRateSource provides rates on a given date.
type ExchangeRateSource interface {
	GetRate(ctx context.Context, currencyCode string, date time.Time) (*ExchangeRate, error)
}

// CurrencyConverter executes anchor-based conversions.
type CurrencyConverter struct {
	source ExchangeRateSource
}

func NewCurrencyConverter(source ExchangeRateSource) *CurrencyConverter {
	return &CurrencyConverter{source: source}
}

// Convert converts amount in minor units (e.g. cents) from one currency to another.
func (c *CurrencyConverter) Convert(ctx context.Context, amountCents int64, fromCur, toCur string, date time.Time) (int64, error) {
	if fromCur == toCur {
		return amountCents, nil
	}

	fromRate, err := c.source.GetRate(ctx, fromCur, date)
	if err != nil {
		return 0, fmt.Errorf("failed to get rate for %s: %w", fromCur, err)
	}

	toRate, err := c.source.GetRate(ctx, toCur, date)
	if err != nil {
		return 0, fmt.Errorf("failed to get rate for %s: %w", toCur, err)
	}

	if fromRate.Rate <= 0 {
		return 0, errors.New("invalid non-positive source rate")
	}

	// Formula: target = source * (toRate / fromRate)
	converted := float64(amountCents) * toRate.Rate / fromRate.Rate
	return int64(converted + 0.5), nil // Round half up
}
