package bankstatement

import (
	"fmt"
	"math"
	"regexp"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// PartialReconcile links a debit line with a credit line settled together
// (account.partial.reconcile in Odoo). Its balance-zeroing target is tracked by
// a FullReconcile once the participating lines are fully settled.
type PartialReconcile struct {
	ID             int64     `json:"id"`
	DebitMoveID    int64     `json:"debit_move_id"`
	CreditMoveID   int64     `json:"credit_move_id"`
	DebitLineID    int64     `json:"debit_line_id"`
	CreditLineID   int64     `json:"credit_line_id"`
	Amount         float64   `json:"amount"`
	AmountCurrency float64   `json:"amount_currency"`
	Currency       string    `json:"currency"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// FullReconcile represents a set of lines fully settled, identified by a matching number
// (account.full.reconcile in Odoo). exchange_move_id points at the FX difference move when the
// reconciled lines use different currencies.
type FullReconcile struct {
	ID             int64     `json:"id"`
	MatchingNumber string    `json:"matching_number"`
	ExchangeMoveID *int64    `json:"exchange_move_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReconcileMatchNature constrains when a model applies based on money direction.
type ReconcileMatchNature string

const (
	MatchNatureBoth     ReconcileMatchNature = "both"
	MatchNatureMoneyIn  ReconcileMatchNature = "money_in"
	MatchNatureMoneyOut ReconcileMatchNature = "money_out"
)

// ReconcileMatchAmount constrains the applicable amount range.
type ReconcileMatchAmount string

const (
	MatchAmountLower   ReconcileMatchAmount = "lower"
	MatchAmountGreater ReconcileMatchAmount = "greater"
	MatchAmountBetween ReconcileMatchAmount = "between"
)

// ReconcileMatchLabel constrains the matching label comparison.
type ReconcileMatchLabel string

const (
	MatchLabelContains    ReconcileMatchLabel = "contains"
	MatchLabelNotContains ReconcileMatchLabel = "not_contains"
	MatchLabelRegex       ReconcileMatchLabel = "match_regex"
)

// ReconcileLineAmountType determines how a model line computes its amount (Odoo v19).
type ReconcileLineAmountType string

const (
	LineAmountFixed            ReconcileLineAmountType = "fixed"
	LineAmountPercentage       ReconcileLineAmountType = "percentage"
	LineAmountPercentageStLine ReconcileLineAmountType = "percentage_st_line"
	LineAmountRegex            ReconcileLineAmountType = "regex"
)

// ReconcileModel is an automated reconciliation rule (account.reconcile.model in Odoo v19).
type ReconcileModel struct {
	ID              int64                 `json:"id"`
	Name            string                `json:"name"`
	Sequence        int                   `json:"sequence"`
	IsAutoReconcile bool                  `json:"is_auto_reconcile"`
	MatchNature     ReconcileMatchNature  `json:"match_nature"`
	MatchAmount     *ReconcileMatchAmount `json:"match_amount,omitempty"`
	MatchAmountMin  *float64              `json:"match_amount_min,omitempty"`
	MatchAmountMax  *float64              `json:"match_amount_max,omitempty"`
	MatchLabel      *ReconcileMatchLabel  `json:"match_label,omitempty"`
	MatchLabelParam string                `json:"match_label_param"`
	MatchJournalIDs []int64               `json:"match_journal_ids,omitempty"`
	MatchPartnerIDs []int64               `json:"match_partner_ids,omitempty"`
	MappedPartnerID *int64                `json:"mapped_partner_id,omitempty"`
	Active          bool                  `json:"active"`
	Lines           []ReconcileModelLine  `json:"lines,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// ReconcileModelLine produces one leg of the reconciliation entry when a model applies
// (account.reconcile.model.line in Odoo v19).
type ReconcileModelLine struct {
	ID               int64                   `json:"id"`
	ReconcileModelID int64                   `json:"reconcile_model_id"`
	AmountType       ReconcileLineAmountType `json:"amount_type"`
	Amount           string                  `json:"amount"`
	AccountID        int64                   `json:"account_id"`
	Label            string                  `json:"label,omitempty"`
	TaxIDs           []int64                 `json:"tax_ids,omitempty"`
}

// Validate checks a reconcile model's constraints.
func (m *ReconcileModel) Validate() error {
	if m.Name == "" {
		return platformerrors.Validation("reconcile model name is required", map[string]string{"name": "cannot be empty"})
	}
	switch m.MatchNature {
	case "", MatchNatureBoth, MatchNatureMoneyIn, MatchNatureMoneyOut:
	default:
		return platformerrors.Validation("invalid match nature", map[string]string{"match_nature": string(m.MatchNature)})
	}
	if m.MatchAmount != nil {
		switch *m.MatchAmount {
		case MatchAmountLower, MatchAmountGreater, MatchAmountBetween:
		default:
			return platformerrors.Validation("invalid match amount operator", map[string]string{"match_amount": string(*m.MatchAmount)})
		}
	}
	if m.MatchLabel != nil {
		switch *m.MatchLabel {
		case MatchLabelContains, MatchLabelNotContains, MatchLabelRegex:
		default:
			return platformerrors.Validation("invalid match label operator", map[string]string{"match_label": string(*m.MatchLabel)})
		}
	}
	for i, l := range m.Lines {
		switch l.AmountType {
		case LineAmountFixed, LineAmountPercentage, LineAmountPercentageStLine, LineAmountRegex:
		default:
			return platformerrors.Validation("invalid line amount type", map[string]string{
				fmt.Sprintf("lines[%d].amount_type", i): string(l.AmountType),
			})
		}
		if l.Amount == "" {
			return platformerrors.Validation("line amount is required", map[string]string{
				fmt.Sprintf("lines[%d].amount", i): "cannot be empty",
			})
		}
	}
	return nil
}

// MatchesNature checks money direction compatibility (money_in lines are positive).
func (m *ReconcileModel) MatchesNature(amount float64) bool {
	switch m.MatchNature {
	case "", MatchNatureBoth:
		return true
	case MatchNatureMoneyIn:
		return amount >= 0
	case MatchNatureMoneyOut:
		return amount < 0
	}
	return true
}

// MatchesAmount checks amount constraints.
func (m *ReconcileModel) MatchesAmount(amount float64) bool {
	abs := math.Abs(amount)
	if m.MatchAmount == nil {
		return true
	}
	switch *m.MatchAmount {
	case MatchAmountLower:
		return m.MatchAmountMin == nil || abs < *m.MatchAmountMin
	case MatchAmountGreater:
		return m.MatchAmountMin == nil || abs > *m.MatchAmountMin
	case MatchAmountBetween:
		return (m.MatchAmountMin == nil || abs >= *m.MatchAmountMin) &&
			(m.MatchAmountMax == nil || abs <= *m.MatchAmountMax)
	}
	return true
}

// MatchesLabel checks label constraints against the statement line label.
func (m *ReconcileModel) MatchesLabel(label string) bool {
	if m.MatchLabel == nil {
		return true
	}
	switch *m.MatchLabel {
	case MatchLabelContains:
		return m.MatchLabelParam != "" && containsFold(label, m.MatchLabelParam)
	case MatchLabelNotContains:
		return !(m.MatchLabelParam != "" && containsFold(label, m.MatchLabelParam))
	case MatchLabelRegex:
		re, err := regexp.Compile(m.MatchLabelParam)
		if err != nil {
			return false
		}
		return re.MatchString(label)
	}
	return true
}

// MatchesJournal checks journal restriction.
func (m *ReconcileModel) MatchesJournal(journalID int64) bool {
	if len(m.MatchJournalIDs) == 0 {
		return true
	}
	for _, id := range m.MatchJournalIDs {
		if id == journalID {
			return true
		}
	}
	return false
}

// Matches reports whether the model applies to a given statement line.
func (m *ReconcileModel) Matches(line *BankStatementLine) bool {
	if !m.Active {
		return false
	}
	if !m.MatchesNature(line.Amount) {
		return false
	}
	if !m.MatchesAmount(line.Amount) {
		return false
	}
	if !m.MatchesLabel(line.Name + " " + line.Ref) {
		return false
	}
	if line.JournalID != nil && !m.MatchesJournal(*line.JournalID) {
		return false
	}
	if len(m.MatchPartnerIDs) > 0 && line.PartnerID != nil {
		found := false
		for _, id := range m.MatchPartnerIDs {
			if id == *line.PartnerID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func containsFold(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	upper := func(b byte) byte {
		if b >= 'a' && b <= 'z' {
			return b - 'a' + 'A'
		}
		return b
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if upper(s[i+j]) != upper(sub[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
