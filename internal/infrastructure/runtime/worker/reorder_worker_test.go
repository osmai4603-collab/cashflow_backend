package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

func TestReorderWorker_Pass(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := stockstorage.NewMemoryRepo()
	uc := stockusecase.New(repo, nil, nil, nil, nil, nil, logger)

	w := NewReorderWorker(uc, 1*time.Minute, logger)

	// Since RunReplenishment is a complex method in stock UC,
	// we just ensure it doesn't crash and completes.
	w.pass(ctx)
}
