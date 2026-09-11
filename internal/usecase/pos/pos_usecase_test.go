package posusecase

import (
	"context"
	"testing"

	posstorage "cashflow_backend/internal/adapters/storage/pos"
	"cashflow_backend/internal/domain/pos"
)

func TestSyncOrdersIsIdempotent(t *testing.T) {
	repo := posstorage.NewMemoryRepo()
	useCase := New(repo)
	config := &pos.PosConfig{Name: "Counter", WarehouseID: 1, StockLocationID: 1, JournalID: 1, CompanyID: 1, ManualDiscountLimit: 100}
	if _, err := useCase.CreateConfig(context.Background(), config); err != nil {
		t.Fatalf("create config: %v", err)
	}
	session, err := useCase.OpenSession(context.Background(), config.ID, 1, 1, 100)
	if err != nil {
		t.Fatalf("open session: %v", err)
	}
	batch := &pos.SyncBatch{SessionID: session.ID, IdempotencyKey: "batch-1", Orders: []pos.PosOrder{{ClientUUID: "offline-1", UserID: 1, CompanyID: 1, Lines: []pos.PosOrderLine{{ProductID: 1, Qty: 1, PriceUnit: 10}}}}}
	first, err := useCase.SyncOrders(context.Background(), batch)
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	second, err := useCase.SyncOrders(context.Background(), batch)
	if err != nil {
		t.Fatalf("duplicate sync: %v", err)
	}
	if first.Accepted != 1 || second.Accepted != 1 || !second.Duplicate {
		t.Fatalf("unexpected sync results: first=%+v second=%+v", first, second)
	}
}
