package reportusecase

import (
	"context"
	"time"

	"cashflow_backend/internal/domain/report"
)

type DashboardUseCase struct {
	generator report.ReportGenerator
}

func NewDashboardUseCase(generator report.ReportGenerator) *DashboardUseCase {
	return &DashboardUseCase{generator: generator}
}

func (uc *DashboardUseCase) GetOverviewDashboard(ctx context.Context, companyID int64) (*report.DashboardData, error) {
	now := time.Now()
	opts := report.ReportOptions{
		CompanyID: companyID,
		DateTo:    &now,
	}

	data := &report.DashboardData{
		Title: "Executive Overview",
		KPIs:  []report.DashboardKPI{},
	}

	// Net Profit KPI from P&L
	plConfig := GetProfitAndLossConfig()
	plRes, err := uc.generator.Generate(ctx, plConfig, opts)
	if err == nil {
		for _, line := range plRes.Lines {
			if line.Code == "NET_PROFIT" {
				data.KPIs = append(data.KPIs, report.DashboardKPI{
					Key:   "net_profit",
					Label: "Net Profit",
					Value: line.Values["balance"],
					Unit:  "USD",
				})
			}
		}
	}

	// Total Sales KPI from Sales Analysis
	saConfig := GetSalesAnalysisConfig()
	saRes, err := uc.generator.Generate(ctx, saConfig, opts)
	if err == nil {
		for _, line := range saRes.Lines {
			if line.Code == "TOTAL_SALE" {
				data.KPIs = append(data.KPIs, report.DashboardKPI{
					Key:   "total_sales",
					Label: "Total Sales",
					Value: line.Values["balance"],
					Unit:  "USD",
				})
			}
		}
	}

	// Inventory Value KPI
	ivConfig := GetInventoryValuationConfig()
	ivRes, err := uc.generator.Generate(ctx, ivConfig, opts)
	if err == nil {
		for _, line := range ivRes.Lines {
			if line.Code == "INV_TOTAL" {
				data.KPIs = append(data.KPIs, report.DashboardKPI{
					Key:   "inventory_value",
					Label: "Inventory Value",
					Value: line.Values["balance"],
					Unit:  "USD",
				})
			}
		}
	}

	return data, nil
}
