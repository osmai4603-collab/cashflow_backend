package accounting

import (
	"fmt"
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// AccountType classifies the role of an account in financial statements.
type AccountType string

const (
	// Assets
	AccountTypeAssetReceivable AccountType = "asset_receivable"
	AccountTypeAssetCash       AccountType = "asset_cash"
	AccountTypeAssetCurrent    AccountType = "asset_current"
	AccountTypeAssetNonCurrent AccountType = "asset_non_current"

	// Liabilities
	AccountTypeLiabilityPayable    AccountType = "liability_payable"
	AccountTypeLiabilityCurrent    AccountType = "liability_current"
	AccountTypeLiabilityNonCurrent AccountType = "liability_non_current"

	// Equity
	AccountTypeEquity AccountType = "equity"

	// Income
	AccountTypeIncome      AccountType = "income"
	AccountTypeIncomeOther AccountType = "income_other"

	// Expenses
	AccountTypeExpense             AccountType = "expense"
	AccountTypeExpenseDepreciation AccountType = "expense_depreciation"
	AccountTypeExpenseDirectCost   AccountType = "expense_direct_cost"
)

// AllValidAccountTypes lists all supported account types.
var AllValidAccountTypes = map[AccountType]bool{
	AccountTypeAssetReceivable:     true,
	AccountTypeAssetCash:           true,
	AccountTypeAssetCurrent:        true,
	AccountTypeAssetNonCurrent:     true,
	AccountTypeLiabilityPayable:    true,
	AccountTypeLiabilityCurrent:    true,
	AccountTypeLiabilityNonCurrent: true,
	AccountTypeEquity:              true,
	AccountTypeIncome:              true,
	AccountTypeIncomeOther:         true,
	AccountTypeExpense:             true,
	AccountTypeExpenseDepreciation: true,
	AccountTypeExpenseDirectCost:   true,
}

// IsAsset returns true if the account is an asset.
func (t AccountType) IsAsset() bool {
	return t == AccountTypeAssetReceivable ||
		t == AccountTypeAssetCash ||
		t == AccountTypeAssetCurrent ||
		t == AccountTypeAssetNonCurrent
}

// IsLiability returns true if the account is a liability.
func (t AccountType) IsLiability() bool {
	return t == AccountTypeLiabilityPayable ||
		t == AccountTypeLiabilityCurrent ||
		t == AccountTypeLiabilityNonCurrent
}

// IsEquity returns true if the account is equity.
func (t AccountType) IsEquity() bool {
	return t == AccountTypeEquity
}

// IsIncome returns true if the account represents revenue or income.
func (t AccountType) IsIncome() bool {
	return t == AccountTypeIncome || t == AccountTypeIncomeOther
}

// IsExpense returns true if the account represents an expense or cost.
func (t AccountType) IsExpense() bool {
	return t == AccountTypeExpense ||
		t == AccountTypeExpenseDepreciation ||
		t == AccountTypeExpenseDirectCost
}

// NormalBalance returns "debit" or "credit" according to accounting principles.
func (t AccountType) NormalBalance() string {
	if t.IsAsset() || t.IsExpense() {
		return "debit"
	}
	return "credit"
}

// Account represents an entry in the Chart of Accounts (account.account in Odoo).
type Account struct {
	ID        int64        `json:"id"`
	Code      string       `json:"code"`
	Name      i18n.TranslationString `json:"name"`
	Type      AccountType  `json:"type"`
	Reconcile bool         `json:"reconcile"`
	Currency  string       `json:"currency"`
	ParentID  *int64       `json:"parent_id,omitempty"`
	CompanyID *int64       `json:"company_id,omitempty"`
	Active    bool         `json:"active"`
	Audit     audit.Fields `json:"audit"`
}

// Validate checks Account constraints and invariants.
func (a *Account) Validate() error {
	a.Code = strings.TrimSpace(a.Code)
	if a.Code == "" {
		return platformerrors.Validation("account code is required", map[string]string{
			"code": "cannot be empty",
		})
	}
	if len(a.Code) > 64 {
		return platformerrors.Validation("account code exceeds maximum length", map[string]string{
			"code": "must not exceed 64 characters",
		})
	}

	if len(a.Name) == 0 {
		return platformerrors.Validation("account name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if !AllValidAccountTypes[a.Type] {
		return platformerrors.Validation("invalid account type", map[string]string{
			"type": fmt.Sprintf("unsupported account type '%s'", a.Type),
		})
	}

	if a.Currency == "" {
		a.Currency = "USD"
	}

	if a.ParentID != nil && a.ID > 0 && *a.ParentID == a.ID {
		return platformerrors.Validation("account cannot be its own parent", map[string]string{
			"parent_id": "circular reference detected",
		})
	}

	return nil
}
