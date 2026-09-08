package worker

import (
	"context"
	"log/slog"
	"time"

	maintenanceusecase "cashflow_backend/internal/usecase/maintenance"
)

// NewPreventiveMaintenanceWorker runs recurring preventive maintenance
// reminders. Open recurring requests whose schedule date has arrived get a
// follow-up activity for the assigned technician.
func NewPreventiveMaintenanceWorker(useCase *maintenanceusecase.Service, interval time.Duration, logger *slog.Logger) func(ctx context.Context) {
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
				if err := useCase.RemindRecurringDue(ctx, now.UTC()); err != nil {
					logger.Error("preventive maintenance reminder run failed", "error", err)
					continue
				}
			}
		}
	}
}