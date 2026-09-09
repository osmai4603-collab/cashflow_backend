package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	databasehttp "cashflow_backend/internal/adapters/http/database"
	storage "cashflow_backend/internal/adapters/storage"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/infrastructure/config"
	"cashflow_backend/internal/infrastructure/health"
	"cashflow_backend/internal/infrastructure/server"
	"cashflow_backend/internal/infrastructure/worker"
	platformaudit "cashflow_backend/internal/platform/audit"
	platformauth "cashflow_backend/internal/platform/auth"
	"cashflow_backend/internal/platform/database"
	"cashflow_backend/internal/platform/email"
	"cashflow_backend/internal/platform/i18n"
	"cashflow_backend/internal/platform/notificationbus"
	usecases "cashflow_backend/internal/usecase"
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

// statementNameProvider implements bankstatement.StatementSequenceProvider.
type statementNameProvider struct{}

// NextStatementName derives a stable statement name for a journal and year.
func (statementNameProvider) NextStatementName(_ context.Context, journalCode string, year int) (string, error) {
	return fmt.Sprintf("%s Statement %04d", journalCode, year), nil
}

func main() {
	// ─────────────────────────────────────────────────────────────────────
	// PHASE 1: Initialization
	// ─────────────────────────────────────────────────────────────────────

	// 1a. Structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load default translations
	if err := i18n.LoadFromDirectory("i18n"); err != nil {
		logger.Warn("could not load translations from directory", "error", err)
	}

	logger.Info(i18n.T(context.Background(), "initializing ERP backend service..."))

	// 1b. Parse CLI flags (highest priority layer after runtime overrides)
	cliFlags, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	// 1c. Load configuration (defaults < file < env < CLI)
	loadOpts := []config.Option{}
	if cliFlags.Config != "" {
		loadOpts = append(loadOpts, config.WithFile(cliFlags.Config))
		// -save must tolerate a config file that does not exist yet.
		if cliFlags.SaveConfig {
			if _, statErr := os.Stat(cliFlags.Config); os.IsNotExist(statErr) {
				loadOpts = []config.Option{config.WithNoFile()}
			}
		}
	}
	cfg, err := config.Load(loadOpts...)
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}
	cliFlags.Apply(cfg)

	// --save: persist the effective configuration and exit (like Odoo's config save)
	if cliFlags.SaveConfig {
		savePath := cliFlags.Config
		if savePath == "" {
			if rc := os.Getenv("CASHFLOW_RC"); rc != "" {
				savePath = rc
			} else {
				savePath = "config/cashflow.json"
			}
		}
		store, err := config.NewFileStore(savePath, true)
		if err != nil {
			logger.Error("cannot open config store", "path", savePath, "error", err)
			os.Exit(1)
		}
		if err := store.Save(cfg); err != nil {
			logger.Error("cannot save configuration", "path", savePath, "error", err)
			os.Exit(1)
		}
		logger.Info("configuration saved", "path", savePath)
		os.Exit(0)
	}

	logger.Info("configuration loaded",
		"interface", cfg.Server.Interface,
		"port", cfg.Server.Port,
		"storage_driver", cfg.Database.StorageDriver,
	)

	// 1c. Initialize storage and dependencies
	var repos *storage.CashflowRepositories
	var (
		pinger               health.Pinger
		closer               io.Closer
		localNotificationBus = notificationbus.New()
		notificationRelay    *notificationbus.PostgresRelay
	)

	switch cfg.Database.StorageDriver {
	case "postgres":
		logger.Info("connecting to postgresql...",
			"host", cfg.Database.Host,
			"port", cfg.Database.Port,
			"db", cfg.Database.Name,
			"max_conns", cfg.Database.MaxConns,
		)

		poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
		if err != nil {
			logger.Error("invalid postgres dsn configuration", "error", err)
			os.Exit(1)
		}

		poolCfg.MaxConns = cfg.Database.MaxConns
		poolCfg.MinConns = cfg.Database.MinConns
		poolCfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
		poolCfg.MaxConnIdleTime = cfg.Database.MaxConnIdleTime

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
		notificationRelay = notificationbus.NewPostgresRelay(pool, localNotificationBus)
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
		repos = storage.NewFromPostgres(pool)

	case "memory":
		logger.Info("using in-memory development driver")
		pinger = noopPinger{}
		closer = noopCloser{}
		repos = storage.NewFromMemory()

	default:
		logger.Error("unsupported storage driver", "driver", cfg.Database.StorageDriver)
		os.Exit(1)
	}

	// 1d. Base HTTP Handler and Domain Handlers
	auditSink := platformaudit.NewSlogAuthorizationSink(logger)
	authorizer := platformauth.NewCachedAuthorizer(platformauth.NewACLAuthorizerWithSink(repos.Permission, auditSink))
	var activityBus activity.Bus = localNotificationBus
	if notificationRelay != nil {
		activityBus = notificationRelay
	}
	ucs := usecases.New(repos, logger, authorizer, cfg.Auth.JWTSecret, 24*time.Hour, activityBus, statementNameProvider{})
	handlers := httpadapter.NewHandlers(ucs, cfg.App.Name, cfg.App.Version, logger, localNotificationBus)
	handlers.Database = databasehttp.NewHandler(cfg.Database.Name)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 2: Configuration
	// ─────────────────────────────────────────────────────────────────────

	// 2a. Health checker (Liveness & Readiness)
	healthChecker := health.NewHealthChecker(pinger)

	// 2b. Router configuration using Chi
	router := httpadapter.NewRouterWithHandlers(handlers, healthChecker, logger, cfg, cfg.Auth.JWTSecret, authorizer)

	// 2c. Background worker manager
	wm := worker.NewWorkerManager(logger)

	// Start Email Worker
	emailSender := email.NewSMTPSender(email.Config{
		Host:     cfg.Email.SMTPHost,
		Port:     cfg.Email.SMTPPort,
		User:     cfg.Email.User,
		Password: cfg.Email.Password,
		SSL:      cfg.Email.SSL,
		From:     "no-reply@cashflow.com", // Should be from config
	})
	wm.Start("email-queue-processor", worker.NewEmailWorker(repos.EmailQueue, emailSender, logger))

	// Phase 13 — Auto Reorder: periodic reorder-rule evaluation → purchase proposals
	if cfg.Stock.ReorderEnabled {
		wm.Start("reorder-checker", worker.NewReorderWorker(ucs.Stock, cfg.Stock.ReorderInterval, logger).Run())
	}

	// Phase 24 — Preventive maintenance: recurring request reminders for technicians
	wm.Start("preventive-maintenance", worker.NewPreventiveMaintenanceWorker(ucs.Maintenance, 24*time.Hour, logger))

	// Phase 24 — Fleet contracts: auto-expire + expiry reminders
	wm.Start("fleet-contract-worker", worker.NewContractWorker(ucs.Fleet, 24*time.Hour, logger))

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
	if notificationRelay != nil {
		go func() {
			if err := notificationRelay.Start(sigCtx); err != nil {
				logger.Error("notification relay stopped", "error", err)
			}
		}()
	}

	// Run executes Phases 3 (Startup), 4 (Serving), 5 (Drain), 6 (Graceful Shutdown), and 7 (Cleanup)
	if err := srv.Run(sigCtx); err != nil {
		logger.Error("server encountered fatal error", "error", err)
		os.Exit(1)
	}

	logger.Info("service terminated successfully")
}
