package sequence

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// SequenceType represents the numbering strategy.
type SequenceType string

const (
	SequenceTypeNormal    SequenceType = "normal"
	SequenceTypeDateRange SequenceType = "date_range"
)

// DateRangeGranularity defines the date reset period for date_range sequences.
type DateRangeGranularity string

const (
	DateRangeYear  DateRangeGranularity = "year"
	DateRangeMonth DateRangeGranularity = "month"
	DateRangeDay   DateRangeGranularity = "day"
)

// Sequence defines a centralized document numbering engine (ir.sequence in Odoo).
type Sequence struct {
	ID            int64                `json:"id"`
	Name          string               `json:"name"`
	Code          string               `json:"code"`
	Prefix        string               `json:"prefix,omitempty"`
	Suffix        string               `json:"suffix,omitempty"`
	Padding       int16                `json:"padding"`
	IncrementBy   int                  `json:"increment_by"`
	StartNumber   int                  `json:"start_number"`
	CurrentNumber int                  `json:"current_number"`
	SequenceType  SequenceType         `json:"sequence_type"`
	DateRange     DateRangeGranularity `json:"date_range,omitempty"`
	CompanyID     *int64               `json:"company_id,omitempty"`
	Active        bool                 `json:"active"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// Validate ensures the sequence entity satisfies all domain invariants.
func (s *Sequence) Validate() error {
	trimmedName := strings.TrimSpace(s.Name)
	if trimmedName == "" {
		return platformerrors.Validation("sequence name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	s.Name = trimmedName

	trimmedCode := strings.TrimSpace(s.Code)
	if trimmedCode == "" {
		return platformerrors.Validation("sequence code is required", map[string]string{
			"code": "cannot be empty",
		})
	}
	s.Code = trimmedCode

	if s.Padding < 1 || s.Padding > 20 {
		return platformerrors.Validation("invalid padding value", map[string]string{
			"padding": "must be between 1 and 20",
		})
	}

	if s.IncrementBy < 1 {
		return platformerrors.Validation("increment_by must be positive", map[string]string{
			"increment_by": "must be at least 1",
		})
	}

	if s.SequenceType != SequenceTypeNormal && s.SequenceType != SequenceTypeDateRange {
		return platformerrors.Validation("invalid sequence type", map[string]string{
			"sequence_type": "must be 'normal' or 'date_range'",
		})
	}

	if s.SequenceType == SequenceTypeDateRange {
		switch s.DateRange {
		case DateRangeYear, DateRangeMonth, DateRangeDay:
		default:
			return platformerrors.Validation("invalid date_range granularity", map[string]string{
				"date_range": "must be 'year', 'month', or 'day' when sequence_type is 'date_range'",
			})
		}
	}

	return nil
}

// NextNumber calculates the next number to be formatted into a document reference.
func (s *Sequence) NextNumber() int {
	if s.CurrentNumber == 0 {
		return s.StartNumber
	}
	return s.CurrentNumber + s.IncrementBy
}
