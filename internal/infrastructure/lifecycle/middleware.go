package lifecycle

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// responseRecorder captures the HTTP status code and written byte count.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int64
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

// HTTPLoggingMiddleware provides HTTP status-aware dynamic log level routing
// and health probe log suppression.
//
// Routing Rules:
//   - Status >= 500: logger.Error("http request server error", ...)
//   - Status 400..499: logger.Warn("http request client error", ...)
//   - Status < 400: logger.Info("http request", ...)
//
// Suppression Rules:
//   - Probes (/livez, /readyz) with Status < 400 are suppressed from logs.
//   - Probes with Status >= 400 are logged immediately at WARN or ERROR.
func HTTPLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := newResponseRecorder(w)

			next.ServeHTTP(rec, r)

			duration := time.Since(start)
			path := r.URL.Path

			// Check probe suppression
			isProbe := path == "/livez" || path == "/readyz"
			if isProbe && rec.statusCode < 400 {
				return // Suppress noisy successful probe logs
			}

			// Route by status code
			attrs := []any{
				slog.Int("status", rec.statusCode),
				slog.String("method", r.Method),
				slog.String("path", path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Duration("duration", duration),
				slog.Int64("bytes", rec.bytes),
			}

			switch {
			case rec.statusCode >= 500:
				logger.Error("http request server error", attrs...)
			case rec.statusCode >= 400:
				logger.Warn("http request client error", attrs...)
			default:
				logger.Info("http request", attrs...)
			}
		})
	}
}

// PublicSecurityIsolationMiddleware enforces that diagnostic and operational
// endpoints are not reachable on the public router, returning a clean 404.
func PublicSecurityIsolationMiddleware(blockedPaths ...string) func(http.Handler) http.Handler {
	blockedMap := make(map[string]struct{}, len(blockedPaths))
	for _, p := range blockedPaths {
		blockedMap[p] = struct{}{}
	}

	// Default operational endpoints that must never be exposed publicly
	if len(blockedMap) == 0 {
		blockedMap["/debug/vars"] = struct{}{}
		blockedMap["/info"] = struct{}{}
		blockedMap["/metrics"] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			for blocked := range blockedMap {
				if path == blocked || strings.HasPrefix(path, blocked+"/") {
					http.NotFound(w, r)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
