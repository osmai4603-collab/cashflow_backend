package mrpstorage

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/domain/mrp"
)

func TestMemoryRepoPersistsProductionWorkorderAndTimeLog(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()
	production := &mrp.ProductionOrder{ProductID: 10, ProductQty: 1, DateStart: time.Now(), State: mrp.ProductionStateDraft}
	if err := repo.CreateProduction(ctx, production); err != nil {
		t.Fatal(err)
	}
	workorder := &mrp.Workorder{ProductionID: production.ID, WorkcenterID: 2, Name: "Assembly", State: mrp.WorkorderStateReady}
	if err := repo.CreateWorkorder(ctx, workorder); err != nil {
		t.Fatal(err)
	}
	if err := workorder.Start(); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateWorkorder(ctx, workorder); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateWorkorderTimeLog(ctx, &workorder.TimeLogs[0]); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.GetWorkorderByID(ctx, workorder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != mrp.WorkorderStateProgress || len(loaded.TimeLogs) != 1 {
		t.Fatalf("unexpected workorder: %+v", loaded)
	}
	logs, err := repo.ListWorkorderTimeLogs(ctx, workorder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected one persisted log, got %d", len(logs))
	}
}
