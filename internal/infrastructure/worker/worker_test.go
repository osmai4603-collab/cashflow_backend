package worker_test

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"cashflow_backend/internal/infrastructure/worker"
)

func TestWorkerManager_Lifecycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	wm := worker.NewWorkerManager(logger)

	var counter atomic.Int32
	var stopped atomic.Bool

	wm.Start("test-worker", func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				stopped.Store(true)
				return
			case <-time.After(10 * time.Millisecond):
				counter.Add(1)
			}
		}
	})

	// Wait for at least one iteration
	time.Sleep(35 * time.Millisecond)

	if counter.Load() == 0 {
		t.Errorf("expected worker to have run, counter is 0")
	}

	wm.StopAll()

	if !stopped.Load() {
		t.Errorf("expected worker to be stopped cleanly via context cancel")
	}
}
