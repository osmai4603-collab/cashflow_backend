package audit_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"cashflow_backend/internal/platform/audit"
)

// captureLogger returns a logger whose records land in a buffer with each
// rendered record prefixed by its level name so tests assert severity.
func captureLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler), &buf
}

func TestSlogAuthorizationSink_AllowedLoggedAtInfo(t *testing.T) {
	logger, buf := captureLogger()
	sink := audit.NewSlogAuthorizationSink(logger)

	sink.RecordAuthorizationDecision(audit.AuthorizationDecision{
		Event:     "authorization.allowed",
		UserID:    7,
		CompanyID: 1,
		Model:     "account.move",
		Action:    "read",
		Allowed:   true,
		Reason:    "group 2 grants read",
	})

	out := buf.String()
	if strings.Contains(out, "level=WARN") {
		t.Fatalf("allowed decision must not be logged as WARN, got:\n%s", out)
	}
	if !strings.Contains(out, "level=INFO") {
		t.Fatalf("allowed decision must be logged as INFO, got:\n%s", out)
	}
}

func TestSlogAuthorizationSink_DeniedLoggedAtWarn(t *testing.T) {
	logger, buf := captureLogger()
	sink := audit.NewSlogAuthorizationSink(logger)

	sink.RecordAuthorizationDecision(audit.AuthorizationDecision{
		Event:     "authorization.denied",
		UserID:    7,
		CompanyID: 1,
		Model:     "account.move",
		Action:    "write",
		Allowed:   false,
		Reason:    "group 2 lacks write",
	})

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Fatalf("denied decision must be logged at WARN level, got:\n%s", out)
	}
	if !strings.Contains(out, "authorization.denied") {
		t.Fatalf("expected event name in record, got:\n%s", out)
	}
}