package reportusecase

import (
	"cashflow_backend/internal/domain/report"
)

// GetProfitAndLossConfig returns the default configuration for the P&L report.
func GetProfitAndLossConfig() *report.Report {
	return &report.Report{
		Name: "Profit and Loss",
		Code: "PL",
		Columns: []report.ReportColumn{
			{Name: "Balance", ExpressionLabel: "balance", FigureType: report.FigureTypeMonetary, Sequence: 1},
		},
		Lines: []report.ReportLine{
			{
				Name:           "Income",
				Code:           "INC",
				HierarchyLevel: 1,
				Sequence:       10,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "40, 41", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name:           "Cost of Revenue",
				Code:           "COGS",
				HierarchyLevel: 1,
				Sequence:       20,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "50", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name:           "Gross Profit",
				Code:           "GROSS_PROFIT",
				HierarchyLevel: 2,
				Sequence:       30,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAggregation, Formula: "INC.balance - COGS.balance", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name:           "Expenses",
				Code:           "EXP",
				HierarchyLevel: 1,
				Sequence:       40,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "60, 61", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name:           "Net Profit",
				Code:           "NET_PROFIT",
				HierarchyLevel: 0,
				Sequence:       50,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAggregation, Formula: "GROSS_PROFIT.balance - EXP.balance", FigureType: report.FigureTypeMonetary},
				},
			},
		},
	}
}

// GetBalanceSheetConfig returns the default configuration for the Balance Sheet report.
func GetBalanceSheetConfig() *report.Report {
	return &report.Report{
		Name: "Balance Sheet",
		Code: "BS",
		Columns: []report.ReportColumn{
			{Name: "Balance", ExpressionLabel: "balance", FigureType: report.FigureTypeMonetary, Sequence: 1},
		},
		Lines: []report.ReportLine{
			{
				Name:           "Assets",
				Code:           "ASSETS",
				HierarchyLevel: 1,
				Sequence:       10,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "1", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name:           "Liabilities",
				Code:           "LIABILITIES",
				HierarchyLevel: 1,
				Sequence:       20,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "2", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name:           "Equity",
				Code:           "EQUITY",
				HierarchyLevel: 1,
				Sequence:       30,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "3", FigureType: report.FigureTypeMonetary},
				},
			},
		},
	}
}

// GetTrialBalanceConfig returns the default configuration for the Trial Balance report.
func GetTrialBalanceConfig() *report.Report {
	return &report.Report{
		Name: "Trial Balance",
		Code: "TB",
		Columns: []report.ReportColumn{
			{Name: "Debit", ExpressionLabel: "debit", FigureType: report.FigureTypeMonetary, Sequence: 1},
			{Name: "Credit", ExpressionLabel: "credit", FigureType: report.FigureTypeMonetary, Sequence: 2},
			{Name: "Balance", ExpressionLabel: "balance", FigureType: report.FigureTypeMonetary, Sequence: 3},
		},
		Lines: []report.ReportLine{
			{
				Name:     "Total",
				Code:     "TOTAL",
				Sequence: 1,
				Expressions: []report.ReportExpression{
					{Label: "debit", Engine: report.EngineAccountCodes, Formula: "1, 2, 3, 4, 5, 6"},
					{Label: "credit", Engine: report.EngineAccountCodes, Formula: "1, 2, 3, 4, 5, 6"},
					{Label: "balance", Engine: report.EngineAggregation, Formula: "TOTAL.debit - TOTAL.credit"},
				},
			},
		},
	}
}
