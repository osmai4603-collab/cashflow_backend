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
	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	analytichttp "cashflow_backend/internal/adapters/http/analytic"
	attachmenthttp "cashflow_backend/internal/adapters/http/attachment"
	bankstatementhttp "cashflow_backend/internal/adapters/http/bankstatement"
	companyhttp "cashflow_backend/internal/adapters/http/company"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	currencyhttp "cashflow_backend/internal/adapters/http/currency"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	producthttp "cashflow_backend/internal/adapters/http/product"
	projecthttp "cashflow_backend/internal/adapters/http/project"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	sequencehttp "cashflow_backend/internal/adapters/http/sequence"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	userhttp "cashflow_backend/internal/adapters/http/user"
	activityhttp "cashflow_backend/internal/adapters/http/activity"
	mrphttp "cashflow_backend/internal/adapters/http/mrp"
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	analyticstorage "cashflow_backend/internal/adapters/storage/analytic"
	attachmentstorage "cashflow_backend/internal/adapters/storage/attachment"
	bankstatementstorage "cashflow_backend/internal/adapters/storage/bankstatement"
	companystorage "cashflow_backend/internal/adapters/storage/company"
	crmstorage "cashflow_backend/internal/adapters/storage/crm"
	currencystorage "cashflow_backend/internal/adapters/storage/currency"
	groupstorage "cashflow_backend/internal/adapters/storage/group"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	paymentstorage "cashflow_backend/internal/adapters/storage/payment"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	projectstorage "cashflow_backend/internal/adapters/storage/project"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	sequencestorage "cashflow_backend/internal/adapters/storage/sequence"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	userstorage "cashflow_backend/internal/adapters/storage/user"
	activitystorage "cashflow_backend/internal/adapters/storage/activity"
	mrpstorage "cashflow_backend/internal/adapters/storage/mrp"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/analytic"
	"cashflow_backend/internal/domain/attachment"
	bankstatement "cashflow_backend/internal/domain/bankstatement"
	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/currency"
	domaingroup "cashflow_backend/internal/domain/group"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/domain/mrp"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/payment"
	"cashflow_backend/internal/domain/product"
	projectdomain "cashflow_backend/internal/domain/project"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/sequence"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/domain/user"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/infrastructure/config"
	"cashflow_backend/internal/infrastructure/health"
	"cashflow_backend/internal/infrastructure/server"
	"cashflow_backend/internal/infrastructure/worker"
	platformaudit "cashflow_backend/internal/platform/audit"
	platformauth "cashflow_backend/internal/platform/auth"
	platformcurrency "cashflow_backend/internal/platform/currency"
	"cashflow_backend/internal/platform/database"
	"cashflow_backend/internal/platform/email"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	analyticusecase "cashflow_backend/internal/usecase/analytic"
	attachmentusecase "cashflow_backend/internal/usecase/attachment"
	bankstatementusecase "cashflow_backend/internal/usecase/bankstatement"
	companyusecase "cashflow_backend/internal/usecase/company"
	crmusecase "cashflow_backend/internal/usecase/crm"
	currencyusecase "cashflow_backend/internal/usecase/currency"
	hrusecase "cashflow_backend/internal/usecase/hr"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	paymentusecase "cashflow_backend/internal/usecase/payment"
	productusecase "cashflow_backend/internal/usecase/product"
	projectusecase "cashflow_backend/internal/usecase/project"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
	saleusecase "cashflow_backend/internal/usecase/sale"
	sequenceusecase "cashflow_backend/internal/usecase/sequence"
	stockusecase "cashflow_backend/internal/usecase/stock"
	userusecase "cashflow_backend/internal/usecase/user"
	activityusecase "cashflow_backend/internal/usecase/activity"
	mrpusecase "cashflow_backend/internal/usecase/mrp"
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
	logger.Info("initializing ERP backend service...")

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
	var (
		pinger         health.Pinger
		closer         io.Closer
		partnerRepo    partner.Repository
		productRepo    product.Repository
		accountingRepo accounting.Repository
		analyticRepo   analytic.Repository
		saleRepo       sale.Repository
		purchaseRepo   purchase.Repository
		stockRepo      stock.Repository
		crmRepo        crm.Repository
		paymentRepo    payment.Repository
		hrRepo         hr.Repository
		companyRepo    company.Repository
		userRepo       user.Repository
		currencyRepo   currency.Repository
		currencyRates  currency.RateRepository
		sequenceRepo   sequence.Repository
		attachmentRepo attachment.Repository
		bankstatementRepo bankstatement.Repository
		projectRepo    projectdomain.Repository
		permissionRepo domaingroup.PermissionRepository
		mrpRepo        mrp.Repository
		activityRepo   activity.ActivityRepository
		activityTypeRepo activity.ActivityTypeRepository
		activityMsgRepo  activity.MessageRepository
		activityNotifRepo activity.NotificationRepository
		emailQueueRepo activity.EmailQueueRepository
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
		analyticRepo = analyticstorage.NewPostgresRepo(pool)
		saleRepo = salestorage.NewPostgresRepo(pool)
		purchaseRepo = purchasestorage.NewPostgresRepo(pool)
		stockRepo = stockstorage.NewPostgresRepo(pool)
		crmRepo = crmstorage.NewPostgresRepo(pool)
		paymentRepo = paymentstorage.NewPostgresRepo(pool)
		hrRepo = hrstorage.NewPostgresRepo(pool)
		companyRepo = companystorage.NewPostgresRepo(pool)
		userRepo = userstorage.NewPostgresRepo(pool)
		currencyRepo = currencystorage.NewPostgresRepo(pool)
		currencyRates = currencystorage.NewPostgresRateRepo(pool)
		sequenceRepo = sequencestorage.NewPostgresRepo(pool)
		attachmentRepo = attachmentstorage.NewPostgresRepo(pool)
		bankstatementRepo = bankstatementstorage.NewPostgresRepo(pool)
		projectRepo = projectstorage.NewPostgresRepo(pool)
		permissionRepo = groupstorage.NewPostgresRepo(pool)
		mrpRepo = mrpstorage.NewPostgresRepo(pool)

		activityPostgres := activitystorage.NewPostgresRepo(pool)
		activityRepo = activityPostgres
		activityTypeRepo = activityPostgres
		activityMsgRepo = activityPostgres
		activityNotifRepo = activityPostgres
		emailQueueRepo = activityPostgres

	case "memory":
		logger.Info("using in-memory development driver")
		pinger = noopPinger{}
		closer = noopCloser{}
		partnerRepo = partnerstorage.NewMemoryRepo()
		productRepo = productstorage.NewMemoryRepo()
		accountingRepo = accountingstorage.NewMemoryRepo()
		analyticRepo = analyticstorage.NewMemoryRepo()
		saleRepo = salestorage.NewMemoryRepo()
		purchaseRepo = purchasestorage.NewMemoryRepo()
		stockRepo = stockstorage.NewMemoryRepo()
		crmRepo = crmstorage.NewMemoryRepo()
		paymentRepo = paymentstorage.NewMemoryRepo()
		hrRepo = hrstorage.NewMemoryRepo()
		companyRepo = companystorage.NewMemoryRepo()
		userRepo = userstorage.NewMemoryRepo()
		currencyRepo = currencystorage.NewMemoryRepo()
		currencyRates = currencystorage.NewMemoryRateRepo()
		sequenceRepo = sequencestorage.NewMemoryRepo()
		attachmentRepo = attachmentstorage.NewMemoryRepo()
		bankstatementRepo = bankstatementstorage.NewMemoryRepo()
		projectRepo = projectstorage.NewMemoryRepo()
		groupMemoryRepo := groupstorage.NewMemoryRepo()
		permissionRepo = groupMemoryRepo
		mrpRepo = mrpstorage.NewMemoryRepo()

		activityMemory := activitystorage.NewMemoryRepo()
		activityRepo = activityMemory
		activityTypeRepo = activityMemory
		activityMsgRepo = activityMemory
		activityNotifRepo = activityMemory
		emailQueueRepo = activityMemory

	default:
		logger.Error("unsupported storage driver", "driver", cfg.Database.StorageDriver)
		os.Exit(1)
	}

	// 1d. Base HTTP Handler and Domain Handlers
	handler := httpadapter.NewBaseHandler(cfg.App.Name, cfg.App.Version, logger)
	auditSink := platformaudit.NewSlogAuthorizationSink(logger)
	authorizer := platformauth.NewCachedAuthorizer(platformauth.NewACLAuthorizerWithSink(permissionRepo, auditSink))
	partnerUseCase := partnerusecase.New(partnerRepo, logger, authorizer)
	partnerHandler := partnerhttp.NewHandler(partnerUseCase, logger)
	productUseCase := productusecase.New(productRepo, logger)
	productHandler := producthttp.NewHandler(productUseCase, logger)
	accountingUseCase := accountingusecase.New(accountingRepo, logger)
	// Register EDI Processors
	accountingUseCase.RegisterEDIProcessor(accounting.EDIFormatZatcaPhase1, accountingusecase.NewZatcaProcessor())
	accountingUseCase.RegisterEDIProcessor(accounting.EDIFormatZatcaPhase2, accountingusecase.NewZatcaProcessor())

	accountingHandler := accountinghttp.NewHandler(accountingUseCase, logger)
	analyticUseCase := analyticusecase.New(analyticRepo, logger)
	analyticHandler := analytichttp.NewHandler(analyticUseCase, logger)
	saleUseCase := saleusecase.New(saleRepo, partnerRepo, productRepo, accountingRepo, accountingUseCase, logger)
	saleHandler := salehttp.NewHandler(saleUseCase, logger)
	purchaseUseCase := purchaseusecase.New(purchaseRepo, partnerRepo, productRepo, accountingRepo, accountingUseCase, logger)
	purchaseHandler := purchasehttp.NewHandler(purchaseUseCase, logger)
	stockUseCase := stockusecase.New(stockRepo, partnerRepo, productRepo, saleRepo, purchaseRepo, accountingUseCase, logger)
	stockHandler := stockhttp.NewHandler(stockUseCase, logger)
	crmUseCase := crmusecase.New(crmRepo, partnerUseCase, saleUseCase, logger)
	crmHandler := crmhttp.NewHandler(crmUseCase, logger)
	paymentUseCase := paymentusecase.New(paymentRepo, accountingUseCase, partnerRepo, logger)
	paymentHandler := paymenthttp.NewHandler(paymentUseCase, logger)
	bankstatementUseCase := bankstatementusecase.New(bankstatementRepo, accountingUseCase, statementNameProvider{}, logger)
	bankstatementHandler := bankstatementhttp.NewHandler(bankstatementUseCase, logger)
	attendanceUseCase := hrusecase.NewAttendanceUseCase(hrRepo, companyRepo, logger)
	hrUseCase := hrusecase.New(hrRepo, partnerRepo, logger)
	hrHandler := hrhttp.NewHandler(hrUseCase, attendanceUseCase, logger)

	// Core Infrastructure Modules
	companyUseCase := companyusecase.New(companyRepo, logger)
	companyHandler := companyhttp.NewHandler(companyUseCase, logger)

	userUseCase := userusecase.New(userRepo, partnerRepo, logger, cfg.Auth.JWTSecret, 24*time.Hour)
	userHandler := userhttp.NewHandler(userUseCase, logger)

	currencyConverter := platformcurrency.NewConverter(currencyRates)
	currencyUseCase := currencyusecase.New(currencyRepo, currencyRates, currencyConverter, logger)
	currencyHandler := currencyhttp.NewHandler(currencyUseCase, logger)

	sequenceUseCase := sequenceusecase.New(sequenceRepo, logger)
	sequenceHandler := sequencehttp.NewHandler(sequenceUseCase, logger)

	attachmentUseCase := attachmentusecase.New(attachmentRepo, logger)
	attachmentHandler := attachmenthttp.NewHandler(attachmentUseCase, logger)

	activityUseCase := activityusecase.NewUseCase(activityRepo, activityTypeRepo, activityMsgRepo, activityNotifRepo, emailQueueRepo, nil)
	activityHandler := activityhttp.NewHandler(activityUseCase, logger)

	projectUseCase := projectusecase.New(projectRepo)
	projectHandler := projecthttp.NewHandler(projectUseCase)

	mrpUseCase := mrpusecase.NewUsecase(mrpRepo, sequenceUseCase, stockRepo, accountingUseCase)
	mrpHandler := mrphttp.NewHandler(mrpUseCase, logger)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 2: Configuration
	// ─────────────────────────────────────────────────────────────────────

	// 2a. Health checker (Liveness & Readiness)
	healthChecker := health.NewHealthChecker(pinger)

	// 2b. Router configuration using Chi
	router := httpadapter.NewRouter(handler, healthChecker, partnerHandler, productHandler, accountingHandler, analyticHandler, saleHandler, purchaseHandler, stockHandler, crmHandler, paymentHandler, hrHandler, companyHandler, userHandler, currencyHandler, sequenceHandler, attachmentHandler, activityHandler, logger, cfg, cfg.Auth.JWTSecret, authorizer, projectHandler, bankstatementHandler, mrpHandler)

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
	wm.Start("email-queue-processor", worker.NewEmailWorker(emailQueueRepo, emailSender, logger))

	// Phase 13 — Auto Reorder: periodic reorder-rule evaluation → purchase proposals
	if cfg.Stock.ReorderEnabled {
		wm.Start("reorder-checker", worker.NewReorderWorker(stockUseCase, cfg.Stock.ReorderInterval, logger).Run())
	}

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
