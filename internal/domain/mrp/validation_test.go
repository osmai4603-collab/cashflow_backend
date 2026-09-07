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
