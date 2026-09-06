package lifecycle

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
)

// =============================================================================
// Phase 1: Initialization
// =============================================================================
// Initialize all dependencies BEFORE starting the server.
// If any critical dependency fails, exit immediately.
// Order: config → logger → database → cache → services
// =============================================================================

// Config holds the application configuration loaded from environment or files.
type Config struct {
	Port            string
	DatabaseDSN     string
	ShutdownTimeout int // seconds
	DrainTimeout    int // seconds
}

// LoadConfig reads configuration from environment variables.
// In production, consider using a config file or secret manager.
func LoadConfig() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_DSN environment variable is required")
	}

	return &Config{
		Port:            port,
		DatabaseDSN:     dsn,
		ShutdownTimeout: 10,
		DrainTimeout:    5,
	}, nil
}

// Dependencies holds all initialized dependencies.
// Created in order, closed in REVERSE order during cleanup.
type Dependencies struct {
	Logger *slog.Logger
	DB     *sql.DB
	// Add more dependencies as needed:
	// Cache  *redis.Client
	// Queue  *amqp.Connection
}

// InitDependencies creates all dependencies in the correct order.
// If any critical dependency fails, return an error — do NOT start the server.
func InitDependencies(cfg *Config) (*Dependencies, error) {
	// 1. Logger (first — everything else logs through it)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// 2. Database connection
	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the connection is alive
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("all dependencies initialized successfully")

	return &Dependencies{
		Logger: logger,
		DB:     db,
	}, nil
}

// Close releases all dependencies in REVERSE order of creation.
func (d *Dependencies) Close() {
	// Close in reverse: DB → Logger
	if d.DB != nil {
		if err := d.DB.Close(); err != nil {
			d.Logger.Error("failed to close database", "error", err)
		} else {
			d.Logger.Info("database connection closed")
		}
	}

	d.Logger.Info("all dependencies closed")
}
