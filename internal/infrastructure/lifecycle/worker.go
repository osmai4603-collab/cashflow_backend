package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// WorkerManager manages background workers and ensures they are safely
// coordinated with the application lifecycle.
type WorkerManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	logger *slog.Logger
	mu     sync.Mutex
	active map[string]struct{}
}

// NewWorkerManager constructs a WorkerManager.
func NewWorkerManager(logger *slog.Logger) *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerManager{
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
		active: make(map[string]struct{}),
	}
}

// Start launches a supervised background worker.
// The worker receives a cancellable context and MUST exit when ctx.Done() is closed.
// It also includes panic recovery to prevent a worker panic from crashing the host process.
func (wm *WorkerManager) Start(name string, fn func(ctx context.Context)) {
	wm.mu.Lock()
	wm.active[name] = struct{}{}
	wm.mu.Unlock()

	wm.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				wm.logger.Error("background worker panicked",
					"worker", name,
					"panic", fmt.Sprintf("%v", r),
				)
			}
			wm.mu.Lock()
			delete(wm.active, name)
			wm.mu.Unlock()
			wm.wg.Done()
		}()

		wm.logger.Info("background worker started", "worker", name)
		fn(wm.ctx)
		wm.logger.Info("background worker terminated cleanly", "worker", name)
	}()
}

// Context returns the worker supervisor context.
func (wm *WorkerManager) Context() context.Context {
	return wm.ctx
}

// StopAll cancels the supervisor context, signaling all background workers to stop,
// and waits synchronously for all workers to finish.
func (wm *WorkerManager) StopAll() {
	wm.logger.Info("stopping all background workers...")
	wm.cancel()
	wm.wg.Wait()
	wm.logger.Info("all background workers stopped successfully")
}
