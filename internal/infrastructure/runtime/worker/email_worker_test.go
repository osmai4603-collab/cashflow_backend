package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	activitystorage "cashflow_backend/internal/adapters/storage/activity"
	"cashflow_backend/internal/domain/activity"
)

type mockSender struct {
	sendFunc func(to string, subject, body string) error
}

func (m *mockSender) Send(to string, subject, body string) error {
	return m.sendFunc(to, subject, body)
}

func TestProcessQueue_Success(t *testing.T) {
	ctx := context.Background()
	repo := activitystorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Seed an email
	repo.PushEmail(ctx, &activity.EmailQueueItem{
		RecipientEmail: "test@example.com",
		Subject:        "Hello",
		Body:           "World",
		Status:         activity.EmailStatusQueued,
	})

	sender := &mockSender{
		sendFunc: func(to string, subject, body string) error {
			return nil
		},
	}

	processQueue(ctx, repo, sender, logger)

	items, _ := repo.PopEmails(ctx, 10)
	if len(items) != 0 {
		t.Error("expected queue to be empty after successful processing")
	}

	// Manual check of history/status would require more repo methods or accessing map
}

func TestProcessQueue_Retry(t *testing.T) {
	ctx := context.Background()
	repo := activitystorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo.PushEmail(ctx, &activity.EmailQueueItem{
		RecipientEmail: "test@example.com",
		Status:         activity.EmailStatusQueued,
	})

	sender := &mockSender{
		sendFunc: func(to string, subject, body string) error {
			return errors.New("smtp error")
		},
	}

	processQueue(ctx, repo, sender, logger)

	// In memory repo, PopEmails only returns items whose NextAttemptAt is before now.
	// processQueue updates Status to EmailStatusFailed and adds delay.
	// So PopEmails should return 0 items now.
	items, _ := repo.PopEmails(ctx, 10)
	if len(items) != 0 {
		t.Error("expected item to be in backoff, not immediately poppable")
	}
}
