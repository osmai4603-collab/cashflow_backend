package accounting

import (
	"fmt"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// JournalType defines the transaction nature of a journal.
type JournalType string

const (
	JournalTypeSale     JournalType = "sale"
	JournalTypePurchase JournalType = "purchase"
	JournalTypeCash     JournalType = "cash"
	JournalTypeBank     JournalType = "bank"
	JournalTypeGeneral  JournalType = "general"
)

// Journal represents a financial journal for organizing transactions (account.journal in Odoo).
type Journal struct {
	ID                int64       `json:"id"`
	Name              string      `json:"name"`
	Code              string      `json:"code"`
	Type              JournalType `json:"type"`
	DefaultAccountID  *int64      `json:"default_account_id,omitempty"`
	SuspenseAccountID *int64      `json:"suspense_account_id,omitempty"`
	SequencePrefix    string      `json:"sequence_prefix"` // e.g. "INV/%Y/"
	NextNumber        int         `json:"next_number"`
	Active            bool        `json:"active"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// Validate checks Journal constraints.
func (j *Journal) Validate() error {
	j.Name = strings.TrimSpace(j.Name)
	if j.Name == "" {
		return platformerrors.Validation("journal name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	j.Code = strings.TrimSpace(strings.ToUpper(j.Code))
	if j.Code == "" {
		return platformerrors.Validation("journal code is required", map[string]string{
			"code": "cannot be empty",
		})
	}
	if len(j.Code) > 20 {
		return platformerrors.Validation("journal code exceeds maximum length", map[string]string{
			"code": "must not exceed 20 characters",
		})
	}

	switch j.Type {
	case JournalTypeSale, JournalTypePurchase, JournalTypeCash, JournalTypeBank, JournalTypeGeneral:
	default:
		return platformerrors.Validation("invalid journal type", map[string]string{
			"type": fmt.Sprintf("unsupported journal type '%s'", j.Type),
		})
	}

	if j.SequencePrefix == "" {
		j.SequencePrefix = fmt.Sprintf("%s/%%Y/", j.Code)
	}

	if j.NextNumber <= 0 {
		j.NextNumber = 1
	}

	return nil
}

// FormatSequence generates the formal document number (e.g. "INV/2026/00001").
func (j *Journal) FormatSequence(year int, seq int) string {
	prefix := j.SequencePrefix
	if prefix == "" {
		prefix = fmt.Sprintf("%s/%%Y/", j.Code)
	}
	formattedPrefix := strings.ReplaceAll(prefix, "%Y", fmt.Sprintf("%04d", year))
	return fmt.Sprintf("%s%05d", formattedPrefix, seq)
}
