package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	databasehttp "cashflow_backend/internal/adapters/http/database"
	storage "cashflow_backend/internal/adapters/storage"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/infrastructure/runtime/health"
	"cashflow_backend/internal/infrastructure/runtime/metrics"
	"cashflow_backend/internal/infrastructure/runtime/server"
	"cashflow_backend/internal/infrastructure/runtime/worker"
	platformaudit "cashflow_backend/internal/platform/audit"
	platformauth "cashflow_backend/internal/platform/auth"
	"cashflow_backend/internal/platform/config"
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

type pgxStatsProvider struct {
	pool *pgxpool.Pool
}

func (p *pgxStatsProvider) Stats() metrics.DBStats {
	s := p.pool.Stat()
	return metrics.DBStats{
		MaxConns:          s.MaxConns(),
		ActiveConns:       s.TotalConns() - s.IdleConns(),
		IdleConns:         s.IdleConns(),
		WaitCount:         s.AcquireCount(),
		EmptyAcquireCount: s.EmptyAcquireCount(),
		WaitDuration:      s.AcquireDuration(),
	}
}

type noopPinger struct{}

func (noopPinger) Ping(ctx context.Context) error { return nil }

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

type statementNameProvider struct{}

func (statementNameProvider) NextStatementName(_ context.Context, journalCode string, year int) (string, error) {
	return fmt.Sprintf("%s Statement %04d", journalCode, year), nil
}

type App struct {
	cfg               *config.Configuration
	logger            *slog.Logger
	repos             *storage.CashflowRepositories
	healthChecker     *health.HealthChecker
	router            http.Handler
	workerMgr         *worker.WorkerManager
	server            *server.Server
	closer            io.Closer
	logCloser         io.Closer
	notificationRelay *notificationbus.PostgresRelay
}

func New() *App {
	return &App{}
}

// prettyHandler is a custom slog.Handler for beautiful, colored terminal output.
type prettyHandler struct {
	w io.Writer
}

func (h *prettyHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	var levelColor string
	switch {
	case r.Level >= slog.LevelError:
		levelColor = "\033[31m" // Red
	case r.Level >= slog.LevelWarn:
		levelColor = "\033[33m" // Yellow
	case r.Level >= slog.LevelInfo:
		levelColor = "\033[32m" // Green
	case r.Level >= slog.LevelDebug:
		levelColor = "\033[36m" // Cyan
	}

	// 1. Time with color (Light Gray)
	timeStr := fmt.Sprintf("\033[90m%s\033[0m", r.Time.Format("2006-01-02 03:04:05 PM"))

	// 2. Level
	levelStr := fmt.Sprintf("%s%-5s\033[0m", levelColor, r.Level.String())

	// 3. Extract Method if present
	var methodStr string
	var otherAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "method" {
			val := strings.ToUpper(fmt.Sprint(a.Value.Any()))
			var mColor string
			switch val {
			case "GET":
				mColor = "\033[34m" // Blue
			case "POST":
				mColor = "\033[32m" // Green
			case "PUT", "PATCH":
				mColor = "\033[33m" // Yellow
			case "DELETE":
				mColor = "\033[31m" // Red
			default:
				mColor = "\033[35m" // Magenta
			}
			methodStr = fmt.Sprintf(" \033[1m%s%-4s\033[0m", mColor, val)
		} else {
			otherAttrs = append(otherAttrs, a)
		}
		return true
	})

	// Print: Time Level [Method] Message
	fmt.Fprintf(h.w, "%s %s%s %-30s", timeStr, levelStr, methodStr, r.Message)

	// 4. Other Attributes
	for _, a := range otherAttrs {
		if a.Key == "status" {
			var sCode int
			switch v := a.Value.Any().(type) {
			case int:
				sCode = v
			case int64:
				sCode = int(v)
			}
			if sCode > 0 {
				var sColor string
				switch {
				case sCode >= 500:
					sColor = "\033[1;37;41m" // Bold White on Red Background
				case sCode >= 400:
					sColor = "\033[1;33m"    // Bold Yellow
				case sCode >= 300:
					sColor = "\033[36m"      // Cyan
				default:
					sColor = "\033[32m"      // Green
				}
				fmt.Fprintf(h.w, " %sstatus=%d\033[0m", sColor, sCode)
				continue
			}
		}
		if a.Key == "path" && r.Level >= slog.LevelWarn {
			// Highlight path on warnings and errors
			fmt.Fprintf(h.w, " \033[1;37mpath=%v\033[0m", a.Value.Any())
			continue
		}
		fmt.Fprintf(h.w, " \033[90m%s=\033[0m%v", a.Key, a.Value.Any())
	}

	fmt.Fprint(h.w, "\n")
	return nil
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *prettyHandler) WithGroup(name string) slog.Handler       { return h }

// multiHandler fans out log records to multiple slog handlers concurrently.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// minLevelHandler filters records to only pass through if level >= minLevel.
type minLevelHandler struct {
	handler  slog.Handler
	minLevel slog.Level
}

func (h *minLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.minLevel && h.handler.Enabled(ctx, level)
}

func (h *minLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < h.minLevel {
		return nil
	}
	return h.handler.Handle(ctx, r)
}

func (h *minLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &minLevelHandler{handler: h.handler.WithAttrs(attrs), minLevel: h.minLevel}
}

func (h *minLevelHandler) WithGroup(name string) slog.Handler {
	return &minLevelHandler{handler: h.handler.WithGroup(name), minLevel: h.minLevel}
}

type logCloser struct {
	files []*os.File
}

func (lc *logCloser) Close() error {
	var firstErr error
	for _, f := range lc.files {
		if f != nil {
			_ = f.Sync()
			if err := f.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (a *App) initLogger() (*slog.Logger, io.Closer) {
	out := os.Stderr

	var isTerminal bool
	if stat, err := out.Stat(); err == nil {
		isTerminal = (stat.Mode() & os.ModeCharDevice) != 0
	}

	forceColor := strings.ToLower(os.Getenv("CASHFLOW_LOG_COLOR")) == "true"

	jsonOpts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, attr.Value.Time().Format(time.RFC3339))
			}
			return attr
		},
	}

	var primaryHandler slog.Handler
	if isTerminal || forceColor {
		primaryHandler = &prettyHandler{w: out}
	} else {
		primaryHandler = slog.NewJSONHandler(out, jsonOpts)
	}

	handlers := []slog.Handler{primaryHandler}
	var files []*os.File

	// File logging is enabled by default unless explicitly disabled
	logToFile := strings.ToLower(os.Getenv("CASHFLOW_LOG_TO_FILE")) != "false"
	logDir := os.Getenv("CASHFLOW_LOG_DIR")
	if logDir == "" {
		logDir = "logs"
	}

	if logToFile {
		if err := os.MkdirAll(logDir, 0755); err == nil {
			// 1. All logs (app.log in JSON format)
			appLogPath := filepath.Join(logDir, "app.log")
			if appFile, err := os.OpenFile(appLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				files = append(files, appFile)
				handlers = append(handlers, slog.NewJSONHandler(appFile, jsonOpts))
			}

			// 2. Error & Warning logs only (error.log in JSON format)
			errLogPath := filepath.Join(logDir, "error.log")
			if errFile, err := os.OpenFile(errLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				files = append(files, errFile)
				errHandler := slog.NewJSONHandler(errFile, jsonOpts)
				handlers = append(handlers, &minLevelHandler{handler: errHandler, minLevel: slog.LevelWarn})
			}
		}
	}

	closer := &logCloser{files: files}
	if len(handlers) == 1 {
		return slog.New(primaryHandler), closer
	}
	return slog.New(&multiHandler{handlers: handlers}), closer
}

func (a *App) Bootstrap(args []string) error {
	// ─────────────────────────────────────────────────────────────────────
	// PHASE 1: Initialization
	// ─────────────────────────────────────────────────────────────────────

	// 1a. Structured logger with environment-aware formatting and file persistence
	logger, logCloser := a.initLogger()
	a.logger = logger
	a.logCloser = logCloser

	// Load default translations
	if err := i18n.LoadFromDirectory("i18n"); err != nil {
		a.logger.Warn("could not load translations from directory", "error", err)
	}

	a.logger.Info(i18n.T(context.Background(), "initializing ERP backend service..."))

	// 1b. Parse CLI flags
	cliFlags, err := config.ParseFlags(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		return fmt.Errorf("parse flags: %w", err)
	}

	// 1c. Load configuration
	loadOpts := []config.Option{}
	if cliFlags.Config != "" {
		loadOpts = append(loadOpts, config.WithFile(cliFlags.Config))
		if cliFlags.SaveConfig {
			if _, statErr := os.Stat(cliFlags.Config); os.IsNotExist(statErr) {
				loadOpts = []config.Option{config.WithNoFile()}
			}
		}
	}
	a.cfg, err = config.Load(loadOpts...)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	cliFlags.Apply(a.cfg)

	// --save
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
			return fmt.Errorf("cannot open config store: %w", err)
		}
		if err := store.Save(a.cfg); err != nil {
			return fmt.Errorf("cannot save configuration: %w", err)
		}
		a.logger.Info("configuration saved", "path", savePath)
		os.Exit(0)
	}

	a.logger.Info("configuration loaded",
		"interface", a.cfg.Server.Interface,
		"port", a.cfg.Server.Port,
		"storage_driver", a.cfg.Database.StorageDriver,
	)

	// 1c. Initialize storage and dependencies
	var localNotificationBus = notificationbus.New()

	switch a.cfg.Database.StorageDriver {
	case "postgres":
		a.logger.Info("connecting to postgresql...",
			"host", a.cfg.Database.Host,
			"port", a.cfg.Database.Port,
			"db", a.cfg.Database.Name,
		)

		poolCfg, err := pgxpool.ParseConfig(a.cfg.DSN())
		if err != nil {
			return fmt.Errorf("invalid postgres dsn: %w", err)
		}

		poolCfg.MaxConns = a.cfg.Database.MaxConns
		poolCfg.MinConns = a.cfg.Database.MinConns
		poolCfg.MaxConnLifetime = a.cfg.Database.MaxConnLifetime
		poolCfg.MaxConnIdleTime = a.cfg.Database.MaxConnIdleTime

		initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err := pgxpool.NewWithConfig(initCtx, poolCfg)
		if err != nil {
			cancel()
			return fmt.Errorf("failed to create postgres pool: %w", err)
		}

		if err := pool.Ping(initCtx); err != nil {
			cancel()
			pool.Close()
			return fmt.Errorf("failed to ping postgres: %w", err)
		}
		a.notificationRelay = notificationbus.NewPostgresRelay(pool, localNotificationBus)
		cancel()

		// Register database for metrics
		metrics.RegisterDB(&pgxStatsProvider{pool: pool})
		metrics.RegisterServerURL(a.cfg.BaseURL())

		migrator := database.NewMigrator(pool, migrations.FS, ".", a.logger)
		applied, err := migrator.Up(context.Background())
		if err != nil {
			pool.Close()
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
		if applied > 0 {
			a.logger.Info("database migrations applied successfully", "count", applied)
		}

		a.closer = poolCloser{pool: pool}
		a.repos = storage.NewFromPostgres(pool)
		a.healthChecker = health.NewHealthChecker(pool)

	case "memory":
		a.logger.Info("using in-memory development driver")
		metrics.RegisterServerURL(a.cfg.BaseURL())
		a.closer = noopCloser{}
		a.repos = storage.NewFromMemory()
		a.healthChecker = health.NewHealthChecker(noopPinger{})

	default:
		return fmt.Errorf("unsupported storage driver: %s", a.cfg.Database.StorageDriver)
	}

	// 1d. Use cases and handlers
	auditSink := platformaudit.NewSlogAuthorizationSink(a.logger)
	authorizer := platformauth.NewCachedAuthorizer(platformauth.NewACLAuthorizerWithSink(a.repos.Permission, auditSink))
	var activityBus activity.Bus = localNotificationBus
	if a.notificationRelay != nil {
		activityBus = a.notificationRelay
	}
	ucs := usecases.New(a.repos, a.logger, authorizer, a.cfg.Auth.JWTSecret, 24*time.Hour, activityBus, statementNameProvider{})
	handlers := httpadapter.NewHandlers(ucs, a.cfg.App.Name, a.cfg.App.Version, a.logger, localNotificationBus)
	handlers.Database = databasehttp.NewHandler(a.cfg.Database.Name)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 2: Configuration
	// ─────────────────────────────────────────────────────────────────────

	a.router = httpadapter.NewRouterWithHandlers(handlers, a.healthChecker, a.logger, a.cfg, a.cfg.Auth.JWTSecret, authorizer)
	a.workerMgr = worker.NewWorkerManager(a.logger)

	emailSender := email.NewSMTPSender(email.Config{
		Host:     a.cfg.Email.SMTPHost,
		Port:     a.cfg.Email.SMTPPort,
		User:     a.cfg.Email.User,
		Password: a.cfg.Email.Password,
		SSL:      a.cfg.Email.SSL,
		From:     "no-reply@cashflow.com",
	})
	a.workerMgr.Start("email-queue-processor", worker.NewEmailWorker(a.repos.EmailQueue, emailSender, a.logger))

	if a.cfg.Stock.ReorderEnabled {
		a.workerMgr.Start("reorder-checker", worker.NewReorderWorker(ucs.Stock, a.cfg.Stock.ReorderInterval, a.logger).Run())
	}
	a.workerMgr.Start("preventive-maintenance", worker.NewPreventiveMaintenanceWorker(ucs.Maintenance, 24*time.Hour, a.logger))
	a.workerMgr.Start("fleet-contract-worker", worker.NewContractWorker(ucs.Fleet, 24*time.Hour, a.logger))

	a.server = server.NewServer(a.cfg, a.router, a.healthChecker, a.workerMgr, a.logger, a.closer)

	if a.cfg.Management.Enabled {
		a.server.SetManagementHandler(httpadapter.NewManagementRouter(a.cfg, a.healthChecker, a.logger))
	}

	return nil
}

func (a *App) Run() error {
	// ─────────────────────────────────────────────────────────────────────
	// PHASE 3-7: Server Lifecycle Execution
	// ─────────────────────────────────────────────────────────────────────

	sigCtx, sigStop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer sigStop()

	if a.notificationRelay != nil {
		go func() {
			if err := a.notificationRelay.Start(sigCtx); err != nil {
				a.logger.Error("notification relay stopped", "error", err)
			}
		}()
	}

	if err := a.server.Run(sigCtx); err != nil {
		return fmt.Errorf("server fatal error: %w", err)
	}

	a.logger.Info("service terminated successfully")

	if a.logCloser != nil {
		_ = a.logCloser.Close()
	}
	return nil
}

// Close flushes and closes application-level resources such as log files.
func (a *App) Close() error {
	if a.logCloser != nil {
		return a.logCloser.Close()
	}
	return nil
}
