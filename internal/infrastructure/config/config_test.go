package config_test

import (
	"os"
	"testing"
	"time"

	"cashflow_backend/internal/infrastructure/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any overrides
	os.Unsetenv("PORT")
	os.Unsetenv("READ_TIMEOUT_SECONDS")
	os.Unsetenv("READ_HEADER_TIMEOUT_SECONDS")
	os.Unsetenv("WRITE_TIMEOUT_SECONDS")
	os.Unsetenv("IDLE_TIMEOUT_SECONDS")
	os.Unsetenv("DRAIN_SECONDS")
	os.Unsetenv("SHUTDOWN_SECONDS")

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Port)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected default ReadTimeout 5s, got %v", cfg.ReadTimeout)
	}
	if cfg.ReadHeaderTimeout != 2*time.Second {
		t.Errorf("expected default ReadHeaderTimeout 2s, got %v", cfg.ReadHeaderTimeout)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Errorf("expected default WriteTimeout 10s, got %v", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 120*time.Second {
		t.Errorf("expected default IdleTimeout 120s, got %v", cfg.IdleTimeout)
	}
	if cfg.DrainDuration != 5*time.Second {
		t.Errorf("expected default DrainDuration 5s, got %v", cfg.DrainDuration)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected default ShutdownTimeout 10s, got %v", cfg.ShutdownTimeout)
	}
	if cfg.MaxHeaderBytes != 1<<20 {
		t.Errorf("expected MaxHeaderBytes 1MB, got %d", cfg.MaxHeaderBytes)
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("READ_TIMEOUT_SECONDS", "8")
	t.Setenv("DRAIN_SECONDS", "3")

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("expected Port 9090, got %s", cfg.Port)
	}
	if cfg.ReadTimeout != 8*time.Second {
		t.Errorf("expected ReadTimeout 8s, got %v", cfg.ReadTimeout)
	}
	if cfg.DrainDuration != 3*time.Second {
		t.Errorf("expected DrainDuration 3s, got %v", cfg.DrainDuration)
	}
}
