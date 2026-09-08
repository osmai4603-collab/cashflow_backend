package reportusecase

import (
	"cashflow_backend/internal/domain/report"
)

// GetInventoryValuationConfig returns the configuration for inventory valuation report.
func GetInventoryValuationConfig() *report.Report {
	return &report.Report{
		Name: "Inventory Valuation",
		Code: "INV_VAL",
		Columns: []report.ReportColumn{
			{Name: "Current Value", ExpressionLabel: "balance", FigureType: report.FigureTypeMonetary},
		},
		Lines: []report.ReportLine{
			{
				Name: "Total Inventory",
				Code: "INV_TOTAL",
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "14", FigureType: report.FigureTypeMonetary},
				},
			},
		},
	}
}

// GetSalesAnalysisConfig returns the configuration for sales analysis report.
func GetSalesAnalysisConfig() *report.Report {
	return &report.Report{
		Name: "Sales Analysis",
		Code: "SALE_ANALYSIS",
		Columns: []report.ReportColumn{
			{Name: "Revenue", ExpressionLabel: "balance", FigureType: report.FigureTypeMonetary},
		},
		Lines: []report.ReportLine{
			{
				Name: "Product Sales",
				Code: "PROD_SALE",
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "40", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name: "Service Revenue",
				Code: "SERV_SALE",
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAccountCodes, Formula: "41", FigureType: report.FigureTypeMonetary},
				},
			},
			{
				Name:           "Total Sales",
				Code:           "TOTAL_SALE",
				HierarchyLevel: 0,
				Expressions: []report.ReportExpression{
					{Label: "balance", Engine: report.EngineAggregation, Formula: "PROD_SALE.balance + SERV_SALE.balance", FigureType: report.FigureTypeMonetary},
				},
			},
		},
	}
}
