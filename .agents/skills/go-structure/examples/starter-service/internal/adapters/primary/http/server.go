package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/example/starter-service/internal/platform/config"
)

// Server غلاف خادم HTTP يدعم الفحص التشغيلي والإغلاق الآمن
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// NewServer إعداد خادم HTTP مع الموجه والمهل الزمنية
func NewServer(cfg *config.Config, walletHandler *WalletHandler, logger *slog.Logger) *Server {
	mux := http.NewServeMux()

	// 1. مسارات الفحص التشغيلي (Kubernetes Liveness & Readiness Probes)
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// 2. مسارات الأعمال
	mux.HandleFunc("/api/v1/wallets/transfer", walletHandler.HandleTransfer)

	// تطبيق برمجية وسيطة بسيطة للتسجيل والتعافي من الانهيار
	handler := loggingRecoveryMiddleware(mux, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: srv,
		logger:     logger,
	}
}

// Start تشغيل الخادم
func (s *Server) Start() error {
	s.logger.Info("http server listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown إغلاق الخادم الآمن
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func loggingRecoveryMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("recovered from panic in http handler", "panic", rec)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()

		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}
