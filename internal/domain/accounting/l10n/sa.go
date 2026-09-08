package l10n

import (
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/platform/i18n"
)

// SaudiAccountTemplate represents a standard account in the KSA chart of accounts.
type SaudiAccountTemplate struct {
	Code string
	Name i18n.TranslationString
	Type accounting.AccountType
}

// GetSaudiChartOfAccounts returns a simplified version of the Saudi Chart of Accounts.
func GetSaudiChartOfAccounts() []SaudiAccountTemplate {
	return []SaudiAccountTemplate{
		{Code: "101000", Name: i18n.NewTranslation("Cash on Hand"), Type: accounting.AccountTypeAssetCash},
		{Code: "102000", Name: i18n.NewTranslation("Bank Current Account"), Type: accounting.AccountTypeAssetCash},
		{Code: "105000", Name: i18n.NewTranslation("Accounts Receivable"), Type: accounting.AccountTypeAssetReceivable},
		{Code: "201000", Name: i18n.NewTranslation("Accounts Payable"), Type: accounting.AccountTypeLiabilityPayable},
		{Code: "205000", Name: i18n.NewTranslation("VAT Payable (15%)"), Type: accounting.AccountTypeLiabilityCurrent},
		{Code: "401000", Name: i18n.NewTranslation("Sales Revenue"), Type: accounting.AccountTypeIncome},
		{Code: "501000", Name: i18n.NewTranslation("Cost of Goods Sold"), Type: accounting.AccountTypeExpenseDirectCost},
	}
}

// GetSaudiTaxTemplate returns the standard 15% VAT for Saudi Arabia.
func GetSaudiTaxTemplate() *accounting.Tax {
	return &accounting.Tax{
		Name:        i18n.NewTranslation("VAT 15%"),
		Type:        accounting.TaxTypePercent,
		TypeTaxUse:  accounting.TaxScopeSale,
		Amount:      15.0,
		AmountType:  accounting.TaxTypePercent,
		IncludeBase: false,
		Active:      true,
		CountryCode: "SA",
	}
}
