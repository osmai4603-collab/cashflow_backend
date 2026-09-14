package app

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestPrettyHandler_Highlighting(t *testing.T) {
	var buf bytes.Buffer
	handler := &prettyHandler{w: &buf}
	logger := slog.New(handler)

	ctx := context.Background()

	// Test 1: Normal 200 OK request
	logger.InfoContext(ctx, "http request",
		slog.String("method", "GET"),
		slog.String("path", "/api/v1/partners"),
		slog.Int("status", 200),
	)
	out1 := buf.String()
	buf.Reset()

	if !strings.Contains(out1, "\033[32mstatus=200\033[0m") {
		t.Errorf("expected green status=200, got: %q", out1)
	}
	if !strings.Contains(out1, "INFO") {
		t.Errorf("expected INFO level, got: %q", out1)
	}

	// Test 2: Client Error 404
	logger.WarnContext(ctx, "http request client error",
		slog.String("method", "GET"),
		slog.String("path", "/unknown"),
		slog.Int("status", 404),
	)
	out2 := buf.String()
	buf.Reset()

	if !strings.Contains(out2, "\033[1;33mstatus=404\033[0m") {
		t.Errorf("expected bold yellow status=404, got: %q", out2)
	}
	if !strings.Contains(out2, "\033[1;37mpath=/unknown\033[0m") {
		t.Errorf("expected bold white path for warning, got: %q", out2)
	}
	if !strings.Contains(out2, "WARN") {
		t.Errorf("expected WARN level, got: %q", out2)
	}

	// Test 3: Server Error 500
	logger.ErrorContext(ctx, "http request server error",
		slog.String("method", "POST"),
		slog.String("path", "/api/v1/payments"),
		slog.Int("status", 500),
	)
	out3 := buf.String()
	buf.Reset()

	if !strings.Contains(out3, "\033[1;37;41mstatus=500\033[0m") {
		t.Errorf("expected red background status=500, got: %q", out3)
	}
	if !strings.Contains(out3, "ERROR") {
		t.Errorf("expected ERROR level, got: %q", out3)
	}
}

func TestMinLevelHandler_Filter(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, nil)
	filtered := &minLevelHandler{handler: baseHandler, minLevel: slog.LevelWarn}
	logger := slog.New(filtered)
	ctx := context.Background()

	// Info should be ignored
	logger.InfoContext(ctx, "info message", slog.String("k", "v"))
	if buf.Len() > 0 {
		t.Fatalf("expected 0 bytes for info message, got %d: %s", buf.Len(), buf.String())
	}

	// Warn should be written
	logger.WarnContext(ctx, "warn message", slog.String("warn_key", "warn_val"))
	if !strings.Contains(buf.String(), "warn message") {
		t.Fatalf("expected warn message in output, got: %s", buf.String())
	}
	buf.Reset()

	// Error should be written
	logger.ErrorContext(ctx, "error message", slog.String("err_key", "err_val"))
	if !strings.Contains(buf.String(), "error message") {
		t.Fatalf("expected error message in output, got: %s", buf.String())
	}
}

func TestMultiHandler_FanOut(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewJSONHandler(&buf1, nil)
	h2 := slog.NewJSONHandler(&buf2, nil)

	multi := &multiHandler{handlers: []slog.Handler{h1, h2}}
	logger := slog.New(multi)
	ctx := context.Background()

	logger.InfoContext(ctx, "fanout test", slog.Int("count", 42))

	if !strings.Contains(buf1.String(), "fanout test") || !strings.Contains(buf1.String(), `"count":42`) {
		t.Fatalf("buf1 missing expected log: %s", buf1.String())
	}
	if !strings.Contains(buf2.String(), "fanout test") || !strings.Contains(buf2.String(), `"count":42`) {
		t.Fatalf("buf2 missing expected log: %s", buf2.String())
	}
}

