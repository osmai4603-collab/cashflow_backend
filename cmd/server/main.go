package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	producthttp "cashflow_backend/internal/adapters/http/product"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/infrastructure/config"
	"cashflow_backend/internal/infrastructure/health"
	"cashflow_backend/internal/infrastructure/server"
	"cashflow_backend/internal/infrastructure/worker"
	"cashflow_backend/internal/platform/database"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	productusecase "cashflow_backend/internal/usecase/product"
	"cashflow_backend/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

type poolCloser struct {
	pool *pgxpool.Pool
}

func (c poolCloser) Close() error {
	c.pool.Close()
	return nil
}

type noopPinger struct{}

func (noopPinger) Ping(ctx context.Context) error { return nil }

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

func main() {
	// ─────────────────────────────────────────────────────────────────────
	// PHASE 1: Initialization
	// ─────────────────────────────────────────────────────────────────────

	// 1a. Structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	logger.Info("initializing ERP backend service...")

	// 1b. Load configuration
	cfg := config.Load()

	// 1c. Initialize storage and dependencies
	var (
		pinger      health.Pinger
		closer      io.Closer
		partnerRepo partner.Repository
		productRepo product.Repository
	)

	switch cfg.StorageDriver {
	case "postgres":
		logger.Info("connecting to postgresql...",
			"host", cfg.DBHost,
			"port", cfg.DBPort,
			"db", cfg.DBName,
			"max_conns", cfg.DBMaxConns,
		)

		poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
		if err != nil {
			logger.Error("invalid postgres dsn configuration", "error", err)
			os.Exit(1)
		}

		poolCfg.MaxConns = cfg.DBMaxConns
		poolCfg.MinConns = cfg.DBMinConns
		poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
		poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime

		initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err := pgxpool.NewWithConfig(initCtx, poolCfg)
		if err != nil {
			cancel()
			logger.Error("failed to create postgres connection pool", "error", err)
			os.Exit(1)
		}

		if err := pool.Ping(initCtx); err != nil {
			cancel()
			pool.Close()
			logger.Error("failed to ping postgres database — fail-fast", "error", err)
			os.Exit(1)
		}
		cancel()

		// Run database schema migrations
		migrator := database.NewMigrator(pool, migrations.FS, ".", logger)
		applied, err := migrator.Up(context.Background())
		if err != nil {
			pool.Close()
			logger.Error("failed to apply database migrations", "error", err)
			os.Exit(1)
		}
		if applied > 0 {
			logger.Info("database migrations applied successfully", "count", applied)
		} else {
			logger.Info("database schema is up to date")
		}

		pinger = pool
		closer = poolCloser{pool: pool}
		partnerRepo = partnerstorage.NewPostgresRepo(pool)
		productRepo = productstorage.NewPostgresRepo(pool)

	case "memory":
		logger.Info("using in-memory development driver")
		pinger = noopPinger{}
		closer = noopCloser{}
		partnerRepo = partnerstorage.NewMemoryRepo()
		productRepo = productstorage.NewMemoryRepo()

	default:
		logger.Error("unsupported storage driver", "driver", cfg.StorageDriver)
		os.Exit(1)
	}

	// 1d. Base HTTP Handler and Domain Handlers
	handler := httpadapter.NewBaseHandler(cfg.AppName, cfg.AppVersion, logger)
	partnerUseCase := partnerusecase.New(partnerRepo, logger)
	partnerHandler := partnerhttp.NewHandler(partnerUseCase, logger)
	productUseCase := productusecase.New(productRepo, logger)
	productHandler := producthttp.NewHandler(productUseCase, logger)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 2: Configuration
	// ─────────────────────────────────────────────────────────────────────

	// 2a. Health checker (Liveness & Readiness)
	healthChecker := health.NewHealthChecker(pinger)

	// 2b. Router configuration using Chi
	router := httpadapter.NewRouter(handler, healthChecker, partnerHandler, productHandler, logger)

	// 2c. Background worker manager
	wm := worker.NewWorkerManager(logger)

	// 2d. Create server with strict timeouts and register closer resource for Phase 7
	srv := server.NewServer(cfg, router, healthChecker, wm, logger, closer)

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
