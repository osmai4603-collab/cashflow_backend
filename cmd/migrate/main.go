package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"cashflow_backend/internal/platform/config"
	"cashflow_backend/internal/platform/database"
	"cashflow_backend/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [up|down]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]
	if command != "up" && command != "down" {
		fmt.Fprintf(os.Stderr, "Unknown command %q. Must be 'up' or 'down'\n", command)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	if cfg.Database.StorageDriver != "postgres" {
		logger.Error("migrations are only supported for postgres driver", "driver", cfg.Database.StorageDriver)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		logger.Error("invalid database DSN", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	migrator := database.NewMigrator(pool, migrations.FS, ".", logger)

	switch command {
	case "up":
		count, err := migrator.Up(ctx)
		if err != nil {
			logger.Error("failed to apply migrations", "error", err)
			os.Exit(1)
		}
		if count > 0 {
			logger.Info("migrations applied successfully", "count", count)
		} else {
			logger.Info("schema is already up to date")
		}

	case "down":
		count, err := migrator.Down(ctx)
		if err != nil {
			logger.Error("failed to rollback migration", "error", err)
			os.Exit(1)
		}
		if count > 0 {
			logger.Info("migration rolled back successfully")
		} else {
			logger.Info("no migrations were rolled back")
		}
	}
}
