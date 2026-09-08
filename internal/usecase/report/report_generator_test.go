package reportusecase

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/report"
)

type mockDataRepo struct {
	balances map[string]float64
}

func (m *mockDataRepo) GetBalancesByAccountPrefix(ctx context.Context, prefixes []string, options report.ReportOptions) (map[string]float64, error) {
	return m.balances, nil
}

func (m *mockDataRepo) GetAnalyticBalances(ctx context.Context, analyticIDs []int64, options report.ReportOptions) (map[int64]float64, error) {
	return nil, nil
}

func TestReportGenerator(t *testing.T) {
	mockData := &mockDataRepo{
		balances: map[string]float64{
			"40": 1000,
			"41": 500,
			"50": 800,
			"60": 200,
			"61": 100,
		},
	}
	gen := NewReportGenerator(nil, mockData)

	pl := GetProfitAndLossConfig()
	opts := report.ReportOptions{CompanyID: 1}

	res, err := gen.Generate(context.Background(), pl, opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check Income (INC)
	var incVal float64
	for _, l := range res.Lines {
		if l.Code == "INC" {
			if v, ok := l.Values["balance"].(float64); ok {
				incVal = v
			}
		}
	}
	if incVal != 1500 {
		t.Errorf("expected Income 1500, got %f", incVal)
	}

	// Check Gross Profit (INC - COGS)
	var gpVal float64
	for _, l := range res.Lines {
		if l.Code == "GROSS_PROFIT" {
			if v, ok := l.Values["balance"].(float64); ok {
				gpVal = v
			}
		}
	}
	if gpVal != 700 {
		t.Errorf("expected Gross Profit 700, got %f", gpVal)
	}

	// Check Net Profit (GP - EXP)
	var npVal float64
	for _, l := range res.Lines {
		if l.Code == "NET_PROFIT" {
			if v, ok := l.Values["balance"].(float64); ok {
				npVal = v
			}
		}
	}
	if npVal != 400 {
		t.Errorf("expected Net Profit 400, got %f", npVal)
	}
}
