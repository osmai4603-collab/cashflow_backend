package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	"cashflow_backend/internal/adapters/storage"
	"cashflow_backend/internal/infrastructure/config"
	"cashflow_backend/internal/infrastructure/health"
	"cashflow_backend/internal/infrastructure/server"
	"cashflow_backend/internal/infrastructure/worker"
	"cashflow_backend/internal/usecase"
)

func main() {
	// ─────────────────────────────────────────────────────────────────────
	// PHASE 1: Initialization
	// ─────────────────────────────────────────────────────────────────────

	// 1a. Structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	logger.Info("initializing cashflow backend service...")

	// 1b. Load configuration
	cfg := config.Load()

	// 1c. Initialize storage and dependencies
	repo := storage.NewMemoryTransactionRepo()
	txUseCase := usecase.NewTransactionUseCase(repo)
	handler := httpadapter.NewTransactionHandler(txUseCase, logger)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 2: Configuration
	// ─────────────────────────────────────────────────────────────────────

	// 2a. Health checker (Liveness & Readiness)
	healthChecker := health.NewHealthChecker(repo)

	// 2b. Router configuration using Chi
	router := httpadapter.NewRouter(handler, healthChecker, logger)

	// 2c. Background worker manager
	wm := worker.NewWorkerManager(logger)

	// Example periodic background task (e.g. metrics reporter)
	wm.Start("cashflow-stats-reporter", func(ctx context.Context) {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				summary, err := txUseCase.GetSummary(ctx)
				if err == nil {
					logger.Info("periodic cashflow summary",
						"total_income", summary.TotalIncome,
						"total_expense", summary.TotalExpense,
						"net_cashflow", summary.NetCashflow,
						"count", summary.Count,
					)
				}
			}
		}
	})

	// 2d. Create server with strict timeouts
	srv := server.NewServer(cfg, router, healthChecker, wm, logger, repo)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 3-7: Server Lifecycle Execution
	// ─────────────────────────────────────────────────────────────────────

	// Register OS termination signals
	sigCtx, sigStop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // SIGTERM (Kubernetes, Docker, systemd)
	)
	defer sigStop()

	// Run executes Phases 3 (Startup), 4 (Serving), 5 (Drain), 6 (Graceful Shutdown), and 7 (Cleanup)
	if err := srv.Run(sigCtx); err != nil {
		logger.Error("server encountered fatal error", "error", err)
		os.Exit(1)
	}

	logger.Info("service terminated successfully")
}
