package workers

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// =============================================================================
// Supervised Background Worker with Panic Recovery and Teardown
// =============================================================================

// WorkerTask represents the unit of work executed by the background worker.
type WorkerTask func(ctx context.Context) error

// Supervisor manages and coordinates background workers.
type Supervisor struct {
	wg     sync.WaitGroup
	logger *slog.Logger
}

func NewSupervisor(logger *slog.Logger) *Supervisor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Supervisor{logger: logger}
}

// Start launches a supervised worker goroutine tracked by sync.WaitGroup.
func (s *Supervisor) Start(ctx context.Context, name string, task WorkerTask) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				s.logger.Error("background worker crashed with panic",
					"worker", name,
					"panic", fmt.Sprint(r),
					"stack", string(stack),
				)
			}
		}()

		s.logger.Info("background worker started", "worker", name)

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("background worker stopping", "worker", name, "reason", ctx.Err())
				return
			default:
				if err := task(ctx); err != nil {
					s.logger.Warn("background worker task execution returned error",
						"worker", name,
						"error", err,
					)
					// Small cooldown to prevent tight busy loops on continuous failure
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()
}

// Wait blocks until all supervised workers have completed their shutdown.
func (s *Supervisor) Wait() {
	s.wg.Wait()
	s.logger.Info("all background workers stopped cleanly")
}
