package activityhttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cashflow_backend/internal/infrastructure/runtime/metrics"
	"cashflow_backend/internal/platform/notificationbus"
)

// signalRecorder flushes and exposes each Flush() call on a channel so tests can
// deterministically observe that SSE frames reached the wire through the
// middleware's ResponseWriter wrappers.
type signalRecorder struct {
	*httptest.ResponseRecorder
	flushed chan struct{}
}

func newSignalRecorder() *signalRecorder {
	return &signalRecorder{ResponseRecorder: httptest.NewRecorder(), flushed: make(chan struct{}, 16)}
}

func (s *signalRecorder) Flush() {
	s.ResponseRecorder.Flush()
	select {
	case s.flushed <- struct{}{}:
	default:
	}
}

func (s *signalRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseRecorder
}

func waitFlush(t *testing.T, rec *signalRecorder) {
	t.Helper()
	select {
	case <-rec.flushed:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE flush frame")
	}
}

func TestStreamNotifications_ThroughMiddlewareWrapperChain(t *testing.T) {
	bus := notificationbus.New()
	handler := metrics.Middleware(newTestHandlerWithBus(t, bus))

	rec := newSignalRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/stream", nil).WithContext(ctx)
	req = withAuth(req, 100, 1)

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	// First flush confirms ": connected" reached the recorder.
	waitFlush(t, rec)

	notifyCtx, notifyCancel := context.WithTimeout(context.Background(), time.Second)
	defer notifyCancel()
	_ = bus.NotifyUser(notifyCtx, 100, "invoice", map[string]any{"id": 7})
	// Second flush confirms the event frame was written through the wrapper.
	waitFlush(t, rec)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not terminate after context cancellation")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, ": connected") {
		t.Errorf("expected connection frame in body:\n%s", body)
	}
	if !strings.Contains(body, "event: invoice") {
		t.Errorf("expected invoice event frame in body:\n%s", body)
	}
	if !strings.Contains(body, `"id":7`) {
		t.Errorf("expected payload in body:\n%s", body)
	}
}