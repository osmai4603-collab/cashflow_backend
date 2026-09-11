package mrp

import (
	"testing"
	"time"
)

func TestProductionOrderValidate(t *testing.T) {
	t.Run("rejects invalid product", func(t *testing.T) {
		po := &ProductionOrder{ProductQty: 2}
		if err := po.Validate(); err == nil {
			t.Fatal("expected validation error for missing product")
		}
	})

	t.Run("rejects unknown states", func(t *testing.T) {
		po := &ProductionOrder{ProductID: 10, State: "unknown"}
		if err := po.Validate(); err == nil {
			t.Fatal("expected invalid production state to fail validation")
		}
	})

	t.Run("fills defaults", func(t *testing.T) {
		po := &ProductionOrder{ProductID: 10, ProductQty: 0}
		if err := po.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
		if po.ProductQty != 1.0 {
			t.Fatalf("expected default product qty 1.0, got %v", po.ProductQty)
		}
		if po.State != ProductionStateDraft {
			t.Fatalf("expected draft state, got %q", po.State)
		}
		if po.ReservationState != ReservationStateWaiting {
			t.Fatalf("expected waiting reservation state, got %q", po.ReservationState)
		}
		if po.DateStart.IsZero() {
			t.Fatal("expected date start to be set")
		}
	})
}

func TestWorkcenterValidate(t *testing.T) {
	wc := &Workcenter{Name: "Cutting", CostPerHour: 25}
	if err := wc.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if wc.TimeEfficiency != 100 {
		t.Fatalf("expected default time efficiency 100, got %v", wc.TimeEfficiency)
	}
	if wc.Capacity != 1 {
		t.Fatalf("expected default capacity 1, got %v", wc.Capacity)
	}
}

func TestBoMValidate(t *testing.T) {
	bom := &BillOfMaterials{ProductID: 7, ProductQty: 0}
	if err := bom.Validate(); err == nil {
		t.Fatal("expected validation error for non-positive qty")
	}

	bom = &BillOfMaterials{ProductID: 7, ProductQty: 2}
	if err := bom.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if bom.Type != BomTypeNormal {
		t.Fatalf("expected normal BoM type, got %q", bom.Type)
	}
	if bom.ReadyToProduce == "" {
		t.Fatal("expected ready_to_produce default")
	}
}

func TestRoutingOperationValidate(t *testing.T) {
	op := &RoutingOperation{WorkcenterID: 1}
	if err := op.Validate(); err == nil {
		t.Fatal("expected validation error for missing operation name")
	}

	op = &RoutingOperation{Name: "Drill", WorkcenterID: 1, TimeMode: "manual", TimeCycleManual: 15}
	if err := op.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if op.TimeMode == "" {
		t.Fatal("expected time mode to remain set")
	}
}

func TestWorkorderValidate(t *testing.T) {
	wo := &Workorder{ProductionID: 21, WorkcenterID: 5, Name: "Assemble", Sequence: 1}
	if err := wo.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if wo.State != WorkorderStateReady {
		t.Fatalf("expected default state ready, got %q", wo.State)
	}
	if wo.DurationExpected < 0 {
		t.Fatal("expected non-negative duration default")
	}

	badState := &Workorder{ProductionID: 21, WorkcenterID: 5, Name: "Assemble", State: "unknown"}
	if err := badState.Validate(); err == nil {
		t.Fatal("expected invalid workorder state to fail validation")
	}

	bad := &Workorder{ProductionID: 0, WorkcenterID: 0, Name: " "}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected invalid workorder to fail validation")
	}

	if time.Now().IsZero() {
		t.Fatal("time must be valid")
	}
}

func TestWorkorderTimeTracking(t *testing.T) {
	workorder := &Workorder{ID: 3, ProductionID: 21, WorkcenterID: 5, Name: "Assemble", State: WorkorderStateReady}
	if err := workorder.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if workorder.State != WorkorderStateProgress || len(workorder.TimeLogs) != 1 {
		t.Fatalf("expected active work order and one time log, got state=%q logs=%d", workorder.State, len(workorder.TimeLogs))
	}
	if err := workorder.Pause(); err != nil {
		t.Fatalf("pause failed: %v", err)
	}
	if workorder.State != WorkorderStatePaused || workorder.Duration < 0 || workorder.TimeLogs[0].DateEnd == nil {
		t.Fatalf("expected closed paused interval, got state=%q duration=%v", workorder.State, workorder.Duration)
	}
	if err := workorder.Resume(); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if err := workorder.Finish(4); err != nil {
		t.Fatalf("finish failed: %v", err)
	}
	if workorder.State != WorkorderStateDone || workorder.QtyProduced != 4 || workorder.DateFinished == nil {
		t.Fatalf("expected completed work order, got state=%q qty=%v", workorder.State, workorder.QtyProduced)
	}
	if len(workorder.TimeLogs) != 2 || workorder.TimeLogs[1].DateEnd == nil || workorder.Duration < workorder.TimeLogs[0].Duration {
		t.Fatalf("expected two closed intervals, got logs=%d duration=%v", len(workorder.TimeLogs), workorder.Duration)
	}
}

func TestSchedulingEngineForwardAndBackward(t *testing.T) {
	start := time.Date(2026, time.March, 9, 8, 0, 0, 0, time.UTC)
	operations := []RoutingOperation{
		{ID: 2, WorkcenterID: 20, Sequence: 20, TimeCycleManual: 30},
		{ID: 1, WorkcenterID: 10, Sequence: 10, TimeCycleManual: 60},
	}
	engine := NewSchedulingEngine(nil)
	forward, err := engine.ForwardSchedule(nil, start, operations)
	if err != nil {
		t.Fatalf("forward schedule failed: %v", err)
	}
	if len(forward) != 2 || forward[0].OperationID != 1 || !forward[1].PlannedEnd.Equal(start.Add(90*time.Minute)) {
		t.Fatalf("unexpected forward schedule: %+v", forward)
	}
	backward, err := engine.BackwardSchedule(nil, start.Add(90*time.Minute), operations)
	if err != nil {
		t.Fatalf("backward schedule failed: %v", err)
	}
	if len(backward) != 2 || !backward[0].PlannedStart.Equal(start) || !backward[1].PlannedEnd.Equal(start.Add(90*time.Minute)) {
		t.Fatalf("unexpected backward schedule: %+v", backward)
	}
}

func TestCalculateOEE(t *testing.T) {
	from := time.Date(2026, time.March, 9, 8, 0, 0, 0, time.UTC)
	to := from.Add(8 * time.Hour)
	metrics, err := CalculateOEE(10, from, to, 480, 4, 100, 5, []WorkcenterProductivity{{WorkcenterID: 10, LossType: LossTypeProductive, Duration: 400}})
	if err != nil {
		t.Fatalf("calculate OEE failed: %v", err)
	}
	if metrics.Availability < 83.33 || metrics.Availability > 83.34 || metrics.Quality != 95 || metrics.OEE <= 0 {
		t.Fatalf("unexpected OEE metrics: %+v", metrics)
	}
}

func TestSubcontractingWorkflow(t *testing.T) {
	order := &SubcontractingOrder{ProductionID: 1, SubcontractorID: 2}
	if err := order.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, step := range []func() error{order.ConfirmPurchase, order.MarkSent, order.MarkReceived, order.Finish} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if order.State != SubcontractStateDone {
		t.Fatalf("expected done, got %q", order.State)
	}
}
