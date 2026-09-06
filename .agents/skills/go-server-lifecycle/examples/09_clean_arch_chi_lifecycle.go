package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// =============================================================================
// Clean Architecture & Chi Router Lifecycle Example
// =============================================================================
// Demonstrates how the 7 lifecycle phases integrate cleanly with Clean
// Architecture, Chi router, and background worker management.
// =============================================================================

// ─── 1. DOMAIN LAYER (Pure Go, 0 external dependencies) ──────────────────────

type Item struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type ItemRepository interface {
	Save(ctx context.Context, item Item) error
	FindAll(ctx context.Context) ([]Item, error)
	Ping(ctx context.Context) error
	Close() error
}

// ─── 2. USE CASE LAYER (Application Business Logic) ──────────────────────────

type ItemUseCase struct {
	repo ItemRepository
}

func NewItemUseCase(repo ItemRepository) *ItemUseCase {
	return &ItemUseCase{repo: repo}
}

func (uc *ItemUseCase) CreateItem(ctx context.Context, title string) (Item, error) {
	if title == "" {
		return Item{}, errors.New("title is required")
	}
	item := Item{
		ID:        fmt.Sprintf("item-%d", time.Now().UnixNano()),
		Title:     title,
		CreatedAt: time.Now(),
	}
	if err := uc.repo.Save(ctx, item); err != nil {
		return Item{}, err
	}
	return item, nil
}

func (uc *ItemUseCase) ListItems(ctx context.Context) ([]Item, error) {
	return uc.repo.FindAll(ctx)
}

// ─── 3. ADAPTERS LAYER (Interface Adapters) ──────────────────────────────────

// 3a. Storage Adapter (In-Memory Repository with sync.RWMutex)
type MemoryItemRepo struct {
	mu    sync.RWMutex
	items map[string]Item
}

func NewMemoryItemRepo() *MemoryItemRepo {
	return &MemoryItemRepo{items: make(map[string]Item)}
}

func (r *MemoryItemRepo) Save(ctx context.Context, item Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ID] = item
	return nil
}

func (r *MemoryItemRepo) FindAll(ctx context.Context) ([]Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]Item, 0, len(r.items))
	for _, it := range r.items {
		res = append(res, it)
	}
	return res, nil
}

func (r *MemoryItemRepo) Ping(ctx context.Context) error {
	return nil
}

func (r *MemoryItemRepo) Close() error {
	return nil
}

// 3b. HTTP Adapter with Chi Router
type ItemHandler struct {
	uc     *ItemUseCase
	logger *slog.Logger
}

func NewItemHandler(uc *ItemUseCase, logger *slog.Logger) *ItemHandler {
	return &ItemHandler{uc: uc, logger: logger}
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	item, err := h.uc.CreateItem(r.Context(), body.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.uc.ListItems(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

// ─── 4. INFRASTRUCTURE LAYER (Lifecycle, Workers, Health, Server) ────────────

// 4a. Health Service (/livez and /readyz)
type AppHealthService struct {
	repo    ItemRepository
	isReady atomic.Bool
}

func NewAppHealthService(repo ItemRepository) *AppHealthService {
	h := &AppHealthService{repo: repo}
	h.isReady.Store(true)
	return h
}

func (h *AppHealthService) SetReady(ready bool) {
	h.isReady.Store(ready)
}

func (h *AppHealthService) Livez(w http.ResponseWriter, r *http.Request) {
	// Liveness only checks if the process is alive. Never check DB here.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"alive"}`))
}

func (h *AppHealthService) Readyz(w http.ResponseWriter, r *http.Request) {
	if !h.isReady.Load() {
		http.Error(w, `{"status":"not_ready"}`, http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.repo.Ping(ctx); err != nil {
		http.Error(w, `{"status":"database_unreachable"}`, http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

// 4b. Worker Manager for background goroutines
type AppWorkerManager struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

func NewAppWorkerManager(logger *slog.Logger) *AppWorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &AppWorkerManager{ctx: ctx, cancel: cancel, logger: logger}
}

func (wm *AppWorkerManager) Start(name string, fn func(ctx context.Context)) {
	wm.wg.Add(1)
	go func() {
		defer wm.wg.Done()
		wm.logger.Info("worker started", "name", name)
		fn(wm.ctx)
		wm.logger.Info("worker exited cleanly", "name", name)
	}()
}

func (wm *AppWorkerManager) StopAndWait(timeout time.Duration) error {
	wm.cancel()
	done := make(chan struct{})
	go func() {
		wm.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("timed out waiting for workers")
	}
}

// 4c. Setup Chi Router with Middlewares
func buildChiRouter(h *ItemHandler, health *AppHealthService) http.Handler {
	r := chi.NewRouter()

	// Chi Standard Middleware Stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second)) // Closes Handler Timeout Gap

	// Health probes
	r.Get("/livez", health.Livez)
	r.Get("/readyz", health.Readyz)

	// API Routes
	r.Route("/api/v1/items", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
	})

	return r
}

// 4d. Server orchestrating the 7 phases
type CleanArchServer struct {
	httpServer      *http.Server
	health          *AppHealthService
	workerManager   *AppWorkerManager
	repo            ItemRepository
	logger          *slog.Logger
	drainDuration   time.Duration
	shutdownTimeout time.Duration
}

func RunCleanArchServer() error {
	// PHASE 1: Initialization
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	repo := NewMemoryItemRepo()
	uc := NewItemUseCase(repo)
	handler := NewItemHandler(uc, logger)
	health := NewAppHealthService(repo)
	wm := NewAppWorkerManager(logger)

	router := buildChiRouter(handler, health)

	// PHASE 2: Configuration
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	appServer := &CleanArchServer{
		httpServer:      srv,
		health:          health,
		workerManager:   wm,
		repo:            repo,
		logger:          logger,
		drainDuration:   2 * time.Second,
		shutdownTimeout: 5 * time.Second,
	}

	// PHASE 3: Startup
	wm.Start("cache-refresher", func(ctx context.Context) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// background job
			}
		}
	})

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting http server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// PHASE 4: Serving & OS Signals
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-sigCtx.Done():
		logger.Info("shutdown trigger received")
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	// PHASE 5: Drain Phase
	logger.Info("drain phase: marking server not-ready")
	appServer.health.SetReady(false)
	time.Sleep(appServer.drainDuration)

	// PHASE 6: Graceful Shutdown
	logger.Info("graceful shutdown: stopping HTTP listeners")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), appServer.shutdownTimeout)
	defer cancel()

	if err := appServer.httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	// PHASE 7: Cleanup in Reverse Order
	logger.Info("cleanup phase: stopping workers and closing resources")
	if err := appServer.workerManager.StopAndWait(appServer.shutdownTimeout); err != nil {
		logger.Error("workers stop error", "error", err)
	}

	if err := appServer.repo.Close(); err != nil {
		logger.Error("repository close error", "error", err)
	}

	logger.Info("server exited cleanly")
	return nil
}
