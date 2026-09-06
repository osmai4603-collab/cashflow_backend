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
	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	producthttp "cashflow_backend/internal/adapters/http/product"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	crmstorage "cashflow_backend/internal/adapters/storage/crm"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	paymentstorage "cashflow_backend/internal/adapters/storage/payment"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/payment"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/infrastructure/config"
	"cashflow_backend/internal/infrastructure/health"
	"cashflow_backend/internal/infrastructure/server"
	"cashflow_backend/internal/infrastructure/worker"
	"cashflow_backend/internal/platform/database"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	crmusecase "cashflow_backend/internal/usecase/crm"
	hrusecase "cashflow_backend/internal/usecase/hr"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	paymentusecase "cashflow_backend/internal/usecase/payment"
	productusecase "cashflow_backend/internal/usecase/product"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
	saleusecase "cashflow_backend/internal/usecase/sale"
	stockusecase "cashflow_backend/internal/usecase/stock"
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
		pinger         health.Pinger
		closer         io.Closer
		partnerRepo    partner.Repository
		productRepo    product.Repository
		accountingRepo accounting.Repository
		saleRepo       sale.Repository
		purchaseRepo   purchase.Repository
		stockRepo      stock.Repository
		crmRepo        crm.Repository
		paymentRepo    payment.Repository
		hrRepo         hr.Repository
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
		accountingRepo = accountingstorage.NewPostgresRepo(pool)
		saleRepo = salestorage.NewPostgresRepo(pool)
		purchaseRepo = purchasestorage.NewPostgresRepo(pool)
		stockRepo = stockstorage.NewPostgresRepo(pool)
		crmRepo = crmstorage.NewPostgresRepo(pool)
		paymentRepo = paymentstorage.NewPostgresRepo(pool)
		hrRepo = hrstorage.NewPostgresRepo(pool)

	case "memory":
		logger.Info("using in-memory development driver")
		pinger = noopPinger{}
		closer = noopCloser{}
		partnerRepo = partnerstorage.NewMemoryRepo()
		productRepo = productstorage.NewMemoryRepo()
		accountingRepo = accountingstorage.NewMemoryRepo()
		saleRepo = salestorage.NewMemoryRepo()
		purchaseRepo = purchasestorage.NewMemoryRepo()
		stockRepo = stockstorage.NewMemoryRepo()
		crmRepo = crmstorage.NewMemoryRepo()
		paymentRepo = paymentstorage.NewMemoryRepo()
		hrRepo = hrstorage.NewMemoryRepo()

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
	accountingUseCase := accountingusecase.New(accountingRepo, logger)
	accountingHandler := accountinghttp.NewHandler(accountingUseCase, logger)
	saleUseCase := saleusecase.New(saleRepo, partnerRepo, productRepo, accountingRepo, accountingUseCase, logger)
	saleHandler := salehttp.NewHandler(saleUseCase, logger)
	purchaseUseCase := purchaseusecase.New(purchaseRepo, partnerRepo, productRepo, accountingRepo, accountingUseCase, logger)
	purchaseHandler := purchasehttp.NewHandler(purchaseUseCase, logger)
	stockUseCase := stockusecase.New(stockRepo, partnerRepo, productRepo, saleRepo, purchaseRepo, logger)
	stockHandler := stockhttp.NewHandler(stockUseCase, logger)
	crmUseCase := crmusecase.New(crmRepo, partnerUseCase, saleUseCase, logger)
	crmHandler := crmhttp.NewHandler(crmUseCase, logger)
	paymentUseCase := paymentusecase.New(paymentRepo, accountingUseCase, partnerRepo, logger)
	paymentHandler := paymenthttp.NewHandler(paymentUseCase, logger)
	hrUseCase := hrusecase.New(hrRepo, partnerRepo, logger)
	hrHandler := hrhttp.NewHandler(hrUseCase, logger)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 2: Configuration
	// ─────────────────────────────────────────────────────────────────────

	// 2a. Health checker (Liveness & Readiness)
	healthChecker := health.NewHealthChecker(pinger)

	// 2b. Router configuration using Chi
	router := httpadapter.NewRouter(handler, healthChecker, partnerHandler, productHandler, accountingHandler, saleHandler, purchaseHandler, stockHandler, crmHandler, paymentHandler, hrHandler, logger)

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
