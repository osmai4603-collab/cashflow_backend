package worker

import (
	"context"
	"log/slog"
	"time"

	stockusecase "cashflow_backend/internal/usecase/stock"
)

// ReorderWorker periodically evaluates every active reorder rule and creates
// purchase proposals for the ones that need replenishment (Odoo cron on
// stock.orderpoint).
type ReorderWorker struct {
	uc       *stockusecase.UseCase
	interval time.Duration
	logger   *slog.Logger
}

// NewReorderWorker builds a reorder-checker worker. interval <= 0 falls back to
// a sensible default so a misconfigured value cannot flood the scheduler.
func NewReorderWorker(uc *stockusecase.UseCase, interval time.Duration, logger *slog.Logger) *ReorderWorker {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ReorderWorker{
		uc:       uc,
		interval: interval,
		logger:   logger,
	}
}

// Run blocks until ctx is cancelled, running the replenishment pass immediately
// and then on every tick. Returns a runnable closure suitable for wm.Start.
func (w *ReorderWorker) Run() func(ctx context.Context) {
	return func(ctx context.Context) {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		w.pass(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.pass(ctx)
			}
		}
	}
}

func (w *ReorderWorker) pass(ctx context.Context) {
	start := time.Now()
	items, err := w.uc.RunReplenishment(ctx)
	if err != nil {
		w.logger.ErrorContext(ctx, "reorder check failed", "error", err)
		return
	}
	generated := 0
	for _, it := range items {
		if it.POGenerated {
			generated++
		}
	}
	w.logger.InfoContext(ctx, "reorder check completed",
		"evaluated", len(items), "po_generated", generated, "duration", time.Since(start))
}
