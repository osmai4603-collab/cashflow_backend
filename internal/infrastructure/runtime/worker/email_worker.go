package worker

import (
	"context"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/platform/email"
)

func NewEmailWorker(
	repo activity.EmailQueueRepository,
	sender email.Sender,
	logger *slog.Logger,
) func(ctx context.Context) {
	return func(ctx context.Context) {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processQueue(ctx, repo, sender, logger)
			}
		}
	}
}

func processQueue(ctx context.Context, repo activity.EmailQueueRepository, sender email.Sender, logger *slog.Logger) {
	items, err := repo.PopEmails(ctx, 10)
	if err != nil {
		logger.Error("failed to pop emails from queue", "error", err)
		return
	}

	for _, item := range items {
		err := sender.Send(item.RecipientEmail, item.Subject, item.Body)
		item.Attempts++

		if err != nil {
			logger.Error("failed to send email", "id", item.ID, "error", err)
			item.LastError = err.Error()
			if item.Attempts >= 5 {
				item.Status = activity.EmailStatusDeadLetter
			} else {
				item.Status = activity.EmailStatusFailed
				// Exponential backoff for retry
				item.NextAttemptAt = time.Now().UTC().Add(time.Duration(item.Attempts*item.Attempts) * time.Minute)
			}
		} else {
			item.Status = activity.EmailStatusSent
			now := time.Now().UTC()
			item.SentAt = &now
		}

		if err := repo.UpdateEmail(ctx, &item); err != nil {
			logger.Error("failed to update email status", "id", item.ID, "error", err)
		}
	}
}
