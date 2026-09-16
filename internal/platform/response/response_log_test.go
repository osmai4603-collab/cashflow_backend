package response_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"

	"github.com/go-chi/chi/v5/middleware"
)

// captureLogger buffers structured records so tests can assert on them.
type captureLogger struct {
	buf *strings.Builder
}

func newCaptureLogger() (*captureLogger, *slog.Logger) {
	c := &captureLogger{buf: &strings.Builder{}}
	return c, slog.New(slog.NewTextHandler(c.buf, nil))
}

func TestError_LogsServerErrorDetailsWithRequestID(t *testing.T) {
	c, logger := newCaptureLogger()
	response.SetLogger(logger)
	defer response.SetLogger(slog.Default())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/crm/stages", nil)
	ctx := req.Context()
	ctx = contextWithReqID(ctx, "req-abc-123")
	req = req.WithContext(ctx)

	rootCause := platformerrors.Internal("failed to scan stage",
		&scanError{msg: "Scan error: converting NULL to string is unsupported"},
	)
	response.Error(w, req, rootCause)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	out := c.buf.String()
	for _, want := range []string{"internal server error response", "req-abc-123", "INTERNAL_ERROR", "failed to scan stage", "Scan error: converting NULL to string is unsupported", "request_id"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q:\n%s", want, out)
		}
	}
}

func TestError_TakeErrorCorrelatesRequest(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/moves/3/cancel", nil)
	ctx := req.Context()
	ctx = contextWithReqID(ctx, "req-correl-1")
	req = req.WithContext(ctx)

	appErr := platformerrors.Forbidden("insufficient permissions")
	// 4xx errors are intentionally not captured for the access log.
	response.Error(w, req, appErr)
	if got := response.TakeError(req); got != nil {
		t.Fatalf("expected no captured error for non-5xx, got %v", got)
	}

	w = httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/boom", nil)
	ctx = req2.Context()
	ctx = contextWithReqID(ctx, "req-correl-2")
	req2 = req2.WithContext(ctx)

	rootCause := platformerrors.Internal("kaboom", nil)
	response.Error(w, req2, rootCause)
	captured := response.TakeError(req2)
	if captured == nil {
		t.Fatal("expected captured error for 5xx request")
	}
	if captured.Error() != rootCause.Error() {
		t.Errorf("captured error = %v, want %v", captured, rootCause)
	}
	// Second read must be empty (single-take semantics).
	if again := response.TakeError(req2); again != nil {
		t.Errorf("expected capture to be consumed, got %v", again)
	}
}

// scanError is a plain error type used to simulate a wrapped DB scan failure.
type scanError struct {
	msg string
}

func (e *scanError) Error() string { return e.msg }

func contextWithReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, middleware.RequestIDKey, id)
}