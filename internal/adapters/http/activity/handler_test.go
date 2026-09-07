package activityhttp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	activitystorage "cashflow_backend/internal/adapters/storage/activity"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/platform/auth"
	"cashflow_backend/internal/platform/notificationbus"
	activityusecase "cashflow_backend/internal/usecase/activity"

	"github.com/go-chi/chi/v5"
)

type flushResponseWriter struct {
	header  http.Header
	mu      sync.Mutex
	body    strings.Builder
	flushed chan struct{}
}

func newFlushResponseWriter() *flushResponseWriter {
	return &flushResponseWriter{header: make(http.Header), flushed: make(chan struct{}, 4)}
}

func (w *flushResponseWriter) Header() http.Header { return w.header }

func (w *flushResponseWriter) WriteHeader(statusCode int) {}

func (w *flushResponseWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(data)
}

func (w *flushResponseWriter) Flush() {
	select {
	case w.flushed <- struct{}{}:
	default:
	}
}

func (w *flushResponseWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}

func TestStreamNotificationsRequiresAuthentication(t *testing.T) {
	handler := newActivityTestHandler(notificationbus.New())
	router := chi.NewRouter()
	router.Use(auth.Middleware("test-secret"))
	RegisterRoutes(router, handler)

	recording := httptest.NewRecorder()
	router.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/notifications/stream", nil))
	if recording.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated stream to return 401, got %d", recording.Code)
	}
}

func TestStreamNotificationsDeliversUserEvent(t *testing.T) {
	bus := notificationbus.New()
	handler := newActivityTestHandler(bus)
	router := chi.NewRouter()
	router.Use(auth.Middleware("test-secret"))
	RegisterRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/notifications/stream", nil)
	token, err := auth.GenerateToken(7, 9, []string{"user"}, "test-secret", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	ctx, cancel := context.WithCancel(request.Context())
	request = request.WithContext(ctx)

	writer := newFlushResponseWriter()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(writer, request)
		close(done)
	}()

	select {
	case <-writer.flushed:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("stream did not send initial connection event")
	}
	if !strings.Contains(writer.String(), ": connected") {
		cancel()
		t.Fatalf("expected initial SSE connection event, got %q", writer.String())
	}

	if err := bus.NotifyUser(context.Background(), 7, "mail.notification", activity.Notification{ID: 11}); err != nil {
		cancel()
		t.Fatalf("notify user: %v", err)
	}
	select {
	case <-writer.flushed:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("stream did not flush notification event")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream did not close after request cancellation")
	}
	body := writer.String()
	if !strings.Contains(body, "event: mail.notification") || !strings.Contains(body, `"id":11`) {
		t.Fatalf("expected notification event in SSE body, got %q", body)
	}
}

func newActivityTestHandler(bus *notificationbus.Bus) *Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := activitystorage.NewMemoryRepo()
	useCase := activityusecase.NewUseCase(repo, repo, repo, repo, repo, bus, nil)
	return NewHandler(useCase, logger, bus)
}
