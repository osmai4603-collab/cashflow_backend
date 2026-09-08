package accounting

import (
	"math"
	"time"
)

// ReportLine is a standard summary line in financial statements.
type ReportLine struct {
	AccountID   int64       `json:"account_id"`
	AccountCode string      `json:"account_code"`
	AccountName string      `json:"account_name"`
	AccountType AccountType `json:"account_type"`
	Amount      float64     `json:"amount"`
}

// TrialBalanceLine represents a single account's row in the Trial Balance.
type TrialBalanceLine struct {
	AccountID      int64       `json:"account_id"`
	AccountCode    string      `json:"account_code"`
	AccountName    string      `json:"account_name"`
	AccountType    AccountType `json:"account_type"`
	InitialBalance float64     `json:"initial_balance"`
	Debit          float64     `json:"debit"`
	Credit         float64     `json:"credit"`
	EndingBalance  float64     `json:"ending_balance"`
}

// TrialBalanceReport contains full trial balance aggregates.
type TrialBalanceReport struct {
	FromDate      time.Time          `json:"from_date"`
	ToDate        time.Time          `json:"to_date"`
	Lines         []TrialBalanceLine `json:"lines"`
	TotalDebit    float64            `json:"total_debit"`
	TotalCredit   float64            `json:"total_credit"`
	Difference    float64            `json:"difference"`
	IsBalanced    bool               `json:"is_balanced"`
}

// ComputeTotals calculates trial balance totals and balance verification.
func (r *TrialBalanceReport) ComputeTotals() {
	var totalDebit, totalCredit float64
	for _, l := range r.Lines {
		totalDebit += l.Debit
		totalCredit += l.Credit
	}
	r.TotalDebit = roundTo4(totalDebit)
	r.TotalCredit = roundTo4(totalCredit)
	r.Difference = roundTo4(math.Abs(r.TotalDebit - r.TotalCredit))
	r.IsBalanced = r.Difference <= 0.01
}

// ProfitAndLossReport contains income statement summaries.
type ProfitAndLossReport struct {
	FromDate      time.Time    `json:"from_date"`
	ToDate        time.Time    `json:"to_date"`
	IncomeLines   []ReportLine `json:"income_lines"`
	TotalIncome   float64      `json:"total_income"`
	ExpenseLines  []ReportLine `json:"expense_lines"`
	TotalExpenses float64      `json:"total_expenses"`
	NetProfit     float64      `json:"net_profit"` // TotalIncome - TotalExpenses
}

// ComputeTotals sums income and expenses to determine net profit/loss.
func (r *ProfitAndLossReport) ComputeTotals() {
	var totalIncome, totalExpenses float64
	for _, l := range r.IncomeLines {
		totalIncome += l.Amount
	}
	for _, l := range r.ExpenseLines {
		totalExpenses += l.Amount
	}
	r.TotalIncome = roundTo4(totalIncome)
	r.TotalExpenses = roundTo4(totalExpenses)
	r.NetProfit = roundTo4(r.TotalIncome - r.TotalExpenses)
}

// BalanceSheetReport contains snapshot balance sheet metrics.
type BalanceSheetReport struct {
	AsOfDate                  time.Time    `json:"as_of_date"`
	AssetLines                []ReportLine `json:"asset_lines"`
	TotalAssets               float64      `json:"total_assets"`
	LiabilityLines            []ReportLine `json:"liability_lines"`
	TotalLiabilities          float64      `json:"total_liabilities"`
	EquityLines               []ReportLine `json:"equity_lines"`
	RetainedEarnings          float64      `json:"retained_earnings"`
	TotalEquity               float64      `json:"total_equity"`
	TotalLiabilitiesAndEquity float64      `json:"total_liabilities_and_equity"`
	Difference                float64      `json:"difference"`
	IsBalanced                bool         `json:"is_balanced"`
}

// ComputeTotals calculates balance sheet aggregations and asserts Assets = Liabilities + Equity.
func (r *BalanceSheetReport) ComputeTotals() {
	var totalAssets, totalLiabilities, totalEquity float64
	for _, l := range r.AssetLines {
		totalAssets += l.Amount
	}
	for _, l := range r.LiabilityLines {
		totalLiabilities += l.Amount
	}
	for _, l := range r.EquityLines {
		totalEquity += l.Amount
	}
	r.TotalAssets = roundTo4(totalAssets)
	r.TotalLiabilities = roundTo4(totalLiabilities)
	r.TotalEquity = roundTo4(totalEquity)
	r.TotalLiabilitiesAndEquity = roundTo4(r.TotalLiabilities + r.TotalEquity + r.RetainedEarnings)
	r.Difference = roundTo4(math.Abs(r.TotalAssets - r.TotalLiabilitiesAndEquity))
	r.IsBalanced = r.Difference <= 0.01
}

// GeneralLedgerItem represents an individual posted movement in an account ledger.
type GeneralLedgerItem struct {
	Date           time.Time `json:"date"`
	MoveID         int64     `json:"move_id"`
	MoveName       string    `json:"move_name"`
	LineID         int64     `json:"line_id"`
	AccountID      int64     `json:"account_id"`
	AccountCode    string    `json:"account_code"`
	PartnerID      *int64    `json:"partner_id,omitempty"`
	PartnerName    string    `json:"partner_name,omitempty"`
	Label          string    `json:"label"`
	Debit          float64   `json:"debit"`
	Credit         float64   `json:"credit"`
	RunningBalance float64   `json:"running_balance"`
}
