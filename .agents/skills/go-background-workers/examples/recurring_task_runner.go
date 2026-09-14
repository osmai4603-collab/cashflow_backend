package workers

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

// =============================================================================
// Recurring Task Runner with Safe Ticker Management and Overlap Prevention
// =============================================================================

type RecurringTask func(ctx context.Context)

// RunRecurringTask runs a task periodically, ensuring no ticker leaks and clean cancellation.
func RunRecurringTask(ctx context.Context, name string, interval time.Duration, task RecurringTask, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop() // Prevents ticker leaks in long-running services

	var inFlight atomic.Bool

	logger.Info("recurring task scheduled", "task", name, "interval", interval.String())

	for {
		select {
		case <-ctx.Done():
			logger.Info("recurring task stopped", "task", name)
			return

		case <-ticker.C:
			// Prevent overlapping execution if previous cycle took longer than interval
			if inFlight.CompareAndSwap(false, true) {
				func() {
					defer inFlight.Store(false)
					task(ctx)
				}()
			} else {
				logger.Warn("skipping recurring task cycle: previous execution still in flight",
					"task", name,
				)
			}
		}
	}
}
