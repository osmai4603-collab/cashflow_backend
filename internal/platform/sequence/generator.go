package sequence

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainsequence "cashflow_backend/internal/domain/sequence"
)

// NumberSource provides atomic next-number generation backed by a
// storage layer that locks the sequence row (SELECT ... FOR UPDATE).
// The domain sequence.Repository satisfies this interface.
type NumberSource interface {
	NextValue(ctx context.Context, id int64) (int, error)
}

// Generator composes atomic number generation with Odoo-style formatting
// (prefix/suffix/padding and %(year)s %(month)s %(day)s placeholders).
type Generator struct {
	source NumberSource
}

// NewGenerator constructs a Generator backed by the given NumberSource.
func NewGenerator(source NumberSource) *Generator {
	return &Generator{source: source}
}

// Next atomically fetches the next value for the sequence and formats it.
// The timestamp is used to expand %(year)s, %(month)s, and %(day)s tokens.
func (g *Generator) Next(ctx context.Context, s *domainsequence.Sequence, ts time.Time) (string, error) {
	if g.source == nil {
		return "", fmt.Errorf("sequence generator has no number source")
	}

	next, err := g.source.NextValue(ctx, s.ID)
	if err != nil {
		return "", err
	}

	return Format(s, next, ts), nil
}

// Format expands date tokens and applies padding to the given sequence number.
func Format(s *domainsequence.Sequence, next int, ts time.Time) string {
	prefix := ExpandDateTokens(s.Prefix, ts)
	suffix := ExpandDateTokens(s.Suffix, ts)

	width := int(s.Padding)
	if width < 1 {
		width = 5
	}

	return fmt.Sprintf("%s%0*d%s", prefix, width, next, suffix)
}

// ExpandDateTokens replaces %(year)s, %(month)s, and %(day)s placeholders.
func ExpandDateTokens(template string, ts time.Time) string {
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
