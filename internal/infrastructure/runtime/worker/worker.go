package worker

import (
	"context"
	"log/slog"
	"sync"
)

// WorkerManager manages background goroutines using context cancellation and sync.WaitGroup.
// It ensures that all workers exit cleanly before the server shuts down.
type WorkerManager struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewWorkerManager creates a new WorkerManager with a cancelable context.
func NewWorkerManager(logger *slog.Logger) *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerManager{
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

// Start launches a named worker in a goroutine and registers it in the WaitGroup.
func (wm *WorkerManager) Start(name string, fn func(ctx context.Context)) {
	wm.wg.Add(1)
	go func() {
		defer wm.wg.Done()
		wm.logger.Info("background worker started", "name", name)
		fn(wm.ctx)
		wm.logger.Info("background worker stopped", "name", name)
	}()
}

// StopAll cancels the worker context and blocks until all goroutines have returned.
func (wm *WorkerManager) StopAll() {
	wm.logger.Info("stopping background workers...")
	wm.cancel()
	wm.wg.Wait()
	wm.logger.Info("all background workers stopped successfully")
}
