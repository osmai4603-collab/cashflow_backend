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
		{Code: "101000", Name: i18n.TranslationString{"en_US": "Cash on Hand", "ar_SA": "النقدية بالصندوق"}, Type: accounting.AccountTypeAssetCash},
		{Code: "102000", Name: i18n.TranslationString{"en_US": "Bank Current Account", "ar_SA": "حساب البنك الجاري"}, Type: accounting.AccountTypeAssetCash},
		{Code: "105000", Name: i18n.TranslationString{"en_US": "Accounts Receivable", "ar_SA": "المدينون"}, Type: accounting.AccountTypeAssetReceivable},
		{Code: "201000", Name: i18n.TranslationString{"en_US": "Accounts Payable", "ar_SA": "الدائنون"}, Type: accounting.AccountTypeLiabilityPayable},
		{Code: "205000", Name: i18n.TranslationString{"en_US": "VAT Payable (15%)", "ar_SA": "ضريبة القيمة المضافة المستحقة (15%)"}, Type: accounting.AccountTypeLiabilityCurrent},
		{Code: "401000", Name: i18n.TranslationString{"en_US": "Sales Revenue", "ar_SA": "إيرادات المبيعات"}, Type: accounting.AccountTypeIncome},
		{Code: "501000", Name: i18n.TranslationString{"en_US": "Cost of Goods Sold", "ar_SA": "تكلفة البضاعة المباعة"}, Type: accounting.AccountTypeExpenseDirectCost},
	}
}

// GetSaudiTaxTemplate returns the standard 15% VAT for Saudi Arabia.
func GetSaudiTaxTemplate() *accounting.Tax {
	return &accounting.Tax{
		Name:            "VAT 15%",
		Amount:          15.0,
		AmountType:      accounting.TaxAmountPercentage,
		TypeTaxUse:      accounting.TaxUseSale,
		IncludeBase:     false,
		Active:          true,
		CountryCode:     "SA",
	}
}
