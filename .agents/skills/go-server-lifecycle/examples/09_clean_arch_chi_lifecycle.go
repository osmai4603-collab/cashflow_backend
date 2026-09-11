package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mattn/go-isatty"
)

// =============================================================================
// Clean Architecture & Chi Router Lifecycle with Observability
// =============================================================================

// ─── 1. DOMAIN LAYER ─────────────────────────────────────────────────────────

type ItemRepository interface {
	Ping(ctx context.Context) error
	Stats() DBStats
}

type DBStats struct {
	ActiveConns int32 `json:"active_conns"`
	MaxConns    int32 `json:"max_conns"`
}

// ─── 2. ADAPTERS LAYER ────────────────────────────────────────────────────────

type prettyHandler struct{ w io.Writer }

func (h *prettyHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := fmt.Sprintf("\033[90m%s\033[0m", r.Time.Format("2006-01-02 03:04:05 PM"))
	fmt.Fprintf(h.w, "%s [%s] %s\n", timeStr, r.Level, r.Message)
	return nil
}
func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *prettyHandler) WithGroup(name string) slog.Handler      { return h }

func initLogger() *slog.Logger {
	out := os.Stderr
	if isatty.IsTerminal(out.Fd()) {
		return slog.New(&prettyHandler{w: out})
	}
	return slog.New(slog.NewJSONHandler(out, nil))
}

// ─── 3. INFRASTRUCTURE LAYER ──────────────────────────────────────────────────

type HealthService struct {
	repo    ItemRepository
	isReady atomic.Bool
}

func (h *HealthService) Metrics(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"uptime": "ok",
		"db":     h.repo.Stats(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (h *HealthService) Readyz(w http.ResponseWriter, r *http.Request) {
	if !h.isReady.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func buildRouter(health *HealthService, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			// Suppress healthy probe logs
			if (r.URL.Path == "/metrics" || r.URL.Path == "/readyz") && ww.Status() < 400 {
				return
			}
			logger.Info("request", "path", r.URL.Path, "status", ww.Status())
		})
	})

	r.Get("/readyz", health.Readyz)
	r.Get("/metrics", health.Metrics)

	return r
}

func RunServer() error {
	logger := initLogger()
	// ... (rest of wiring)
	return nil
}
