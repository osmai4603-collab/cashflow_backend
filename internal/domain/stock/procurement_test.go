package stock

import (
	"context"
	"testing"
	"time"
)

type testRuleRepo struct{ rules []StockRule }

func (r testRuleRepo) ListRulesByRoute(context.Context, int64) ([]StockRule, error) {
	return r.rules, nil
}

func TestProcurementEngineSelectsActiveRule(t *testing.T) {
	rule := StockRule{ID: 7, RouteID: 3, Action: ActionBuy, Active: true}
	called := false
	engine := &DefaultProcurementEngine{Rules: testRuleRepo{rules: []StockRule{{ID: 6, RouteID: 3, Active: false}, rule}}, Execute: func(_ context.Context, _ *ProcurementRequest, selected *StockRule) error {
		called = selected.ID == 7
		return nil
	}}
	request := &ProcurementRequest{ProductID: 10, Quantity: 2, LocationID: 8, CompanyID: 1, RouteIDs: []int64{3}, DatePlanned: time.Now()}
	if err := engine.RunProcurement(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected active rule to be executed")
	}
}
