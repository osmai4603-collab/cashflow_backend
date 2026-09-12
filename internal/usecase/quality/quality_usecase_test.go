package qualityusecase

import (
	"context"
	"testing"

	qualitystorage "cashflow_backend/internal/adapters/storage/quality"
	"cashflow_backend/internal/domain/quality"
)

func TestQualityUseCase_ControlPointCheckAndAlertLifecycle(t *testing.T) {
	repo := qualitystorage.NewMemoryRepo()
	uc := New(repo)
	ctx := context.Background()

	min := 10.0
	max := 20.0

	point, err := uc.CreateControlPoint(ctx, &quality.QualityControlPoint{
		Name:      "Length check",
		Trigger:   quality.TriggerReceipt,
		TestType:  "measure",
		NormMin:   &min,
		NormMax:   &max,
		CompanyID: 1,
	})
	if err != nil {
		t.Fatalf("create control point: %v", err)
	}
	if point == nil || point.ID == 0 {
		t.Fatalf("control point id was not assigned: %#v", point)
	}

	points, err := uc.ListControlPoints(ctx)
	if err != nil {
		t.Fatalf("list control points: %v", err)
	}
	if len(points) != 1 || points[0].ID != point.ID {
		t.Fatalf("unexpected list result: %#v", points)
	}

	passValue := 15.0
	passCheck, err := uc.ExecuteCheck(ctx, &quality.QualityCheck{
		PointID:      point.ID,
		ProductID:    77,
		CompanyID:    1,
		MeasureValue: &passValue,
	})
	if err != nil {
		t.Fatalf("execute passing check: %v", err)
	}
	if passCheck.State != quality.CheckStatePass {
		t.Fatalf("passing check should be marked pass, got %q", passCheck.State)
	}

	alerts, err := uc.ListAlerts(ctx)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("pass check should not create alert: %#v", alerts)
	}

	failValue := 25.0
	failCheck, err := uc.ExecuteCheck(ctx, &quality.QualityCheck{
		PointID:      point.ID,
		ProductID:    77,
		CompanyID:    1,
		MeasureValue: &failValue,
	})
	if err != nil {
		t.Fatalf("execute failing check: %v", err)
	}
	if failCheck.State != quality.CheckStateFail {
		t.Fatalf("failing check should be marked fail, got %q", failCheck.State)
	}

	alerts, err = uc.ListAlerts(ctx)
	if err != nil {
		t.Fatalf("list alerts after fail: %v", err)
	}
	if len(alerts) != 1 || alerts[0].Stage != quality.StageNew {
		t.Fatalf("failing check should create a new alert: %#v", alerts)
	}
}
