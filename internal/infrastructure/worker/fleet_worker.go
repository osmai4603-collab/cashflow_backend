package worker

import (
	"context"
	"log/slog"
	"time"

	fleetusecase "cashflow_backend/internal/usecase/fleet"
)

// NewContractWorker refreshes the state of vehicle contracts whose expiration
// date passed and schedules expiry reminders for contracts nearing their end.
func NewContractWorker(useCase *fleetusecase.Service, interval time.Duration, logger *slog.Logger) func(ctx context.Context) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				now = now.UTC()
				if refreshed, err := useCase.RefreshExpiredContracts(ctx, now); err != nil {
					logger.Error("contract refresh run failed", "error", err)
				} else if refreshed > 0 {
					logger.Info("expired fleet contracts refreshed", "count", refreshed)
				}
				if err := useCase.ScheduleContractReminders(ctx, now); err != nil {
					logger.Error("contract reminder run failed", "error", err)
				}
			}
		}
	}
}