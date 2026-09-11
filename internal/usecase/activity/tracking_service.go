package activityusecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"cashflow_backend/internal/domain/activity"
)

// TrackedField represents a field whose transition from OldValue to NewValue is to be audited.
type TrackedField struct {
	Name      string // Technical field name (e.g. "state", "stage_id")
	FieldDesc string // Human-readable description (e.g. "Status", "Stage")
	OldValue  string
	NewValue  string
}

// TrackingService detects and records field changes onto Chatter threads (mail.tracking.value in Odoo).
type TrackingService struct {
	threadService *ThreadService
	logger        *slog.Logger
}

// NewTrackingService creates a new TrackingService instance.
func NewTrackingService(threadService *ThreadService, logger *slog.Logger) *TrackingService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TrackingService{
		threadService: threadService,
		logger:        logger,
	}
}

// TrackChanges compares old and new values, and if any differences exist, generates a tracking
// notification message on the entity's Chatter thread.
func (s *TrackingService) TrackChanges(
	ctx context.Context,
	thread activity.Threadable,
	authorID int64,
	changes []TrackedField,
) error {
	if thread == nil || len(changes) == 0 {
		return nil
	}

	var trackingValues []activity.TrackingValue
	var descLines []string

	for _, c := range changes {
		if c.OldValue == c.NewValue {
			continue
		}

		fieldLabel := c.FieldDesc
		if fieldLabel == "" {
			fieldLabel = c.Name
		}

		trackingValues = append(trackingValues, activity.TrackingValue{
			Field:        c.Name,
			FieldName:    fieldLabel,
			OldValueText: c.OldValue,
			NewValueText: c.NewValue,
		})

		descLines = append(descLines, fmt.Sprintf("%s: %s → %s", fieldLabel, c.OldValue, c.NewValue))
	}

	if len(trackingValues) == 0 {
		return nil // Nothing changed
	}

	body := strings.Join(descLines, "\n")
	var authPtr *int64
	if authorID > 0 {
		authPtr = &authorID
	}

	_, err := s.threadService.PostMessage(ctx, thread, PostMessageInput{
		Subject:        "Field Tracking Update",
		Body:           body,
		MessageType:    activity.MessageTypeNotification,
		AuthorID:       authPtr,
		TrackingValues: trackingValues,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to post field tracking message",
			"res_model", thread.ThreadModel(),
			"res_id", thread.ThreadID(),
			"error", err,
		)
		return err
	}

	s.logger.InfoContext(ctx, "field changes tracked successfully",
		"res_model", thread.ThreadModel(),
		"res_id", thread.ThreadID(),
		"changes_count", len(trackingValues),
	)

	return nil
}
