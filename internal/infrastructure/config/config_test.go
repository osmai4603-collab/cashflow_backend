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

	if cfg.Port != "8070" {
		t.Errorf("expected default Port 8070, got %s", cfg.Port)
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

func TestLoad_PostgresDefaults(t *testing.T) {
	cfg := config.Load()

	if cfg.StorageDriver != "postgres" {
		t.Errorf("expected default StorageDriver postgres, got %s", cfg.StorageDriver)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("expected default DBHost localhost, got %s", cfg.DBHost)
	}
	if cfg.DBPort != "5432" {
		t.Errorf("expected default DBPort 5432, got %s", cfg.DBPort)
	}
	if cfg.DBName != "cashflow" {
		t.Errorf("expected default DBName cashflow, got %s", cfg.DBName)
	}
	if cfg.DBMaxConns != 25 {
		t.Errorf("expected default DBMaxConns 25, got %d", cfg.DBMaxConns)
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg := config.Load()
	expected := "postgres://postgres:postgres@localhost:5432/cashflow?sslmode=disable"
	if cfg.DSN() != expected {
		t.Errorf("expected DSN %s, got %s", expected, cfg.DSN())
	}

	cfg.DatabaseURL = "postgres://custom:pass@remote:5433/customdb?sslmode=require"
	if cfg.DSN() != cfg.DatabaseURL {
		t.Errorf("expected DatabaseURL override %s, got %s", cfg.DatabaseURL, cfg.DSN())
	}
}

func TestLoad_AppAndSecurityDefaults(t *testing.T) {
	cfg := config.Load()

	if cfg.AppName != "odoo_go_backend" {
		t.Errorf("expected default AppName odoo_go_backend, got %s", cfg.AppName)
	}
	if cfg.AppVersion != "0.1.0" {
		t.Errorf("expected default AppVersion 0.1.0, got %s", cfg.AppVersion)
	}
	if cfg.CompanyMode != "single" {
		t.Errorf("expected default CompanyMode single, got %s", cfg.CompanyMode)
	}
	if cfg.RedisHost != "localhost" {
		t.Errorf("expected default RedisHost localhost, got %s", cfg.RedisHost)
	}
	if cfg.RedisPort != "6379" {
		t.Errorf("expected default RedisPort 6379, got %s", cfg.RedisPort)
	}
	if cfg.RedisAddr() != "localhost:6379" {
		t.Errorf("expected RedisAddr localhost:6379, got %s", cfg.RedisAddr())
	}
	if cfg.JWTSecret == "" {
		t.Error("expected non-empty default JWTSecret")
	}
}

func TestLoad_CustomAppAndSecurity(t *testing.T) {
	t.Setenv("APP_NAME", "my_erp")
	t.Setenv("APP_VERSION", "2.0.0")
	t.Setenv("COMPANY_MODE", "multi")
	t.Setenv("REDIS_HOST", "redis-server")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("JWT_SECRET", "super-secret-jwt")

	cfg := config.Load()

	if cfg.AppName != "my_erp" {
		t.Errorf("expected AppName my_erp, got %s", cfg.AppName)
	}
	if cfg.AppVersion != "2.0.0" {
		t.Errorf("expected AppVersion 2.0.0, got %s", cfg.AppVersion)
	}
	if cfg.CompanyMode != "multi" {
		t.Errorf("expected CompanyMode multi, got %s", cfg.CompanyMode)
	}
	if cfg.RedisHost != "redis-server" {
		t.Errorf("expected RedisHost redis-server, got %s", cfg.RedisHost)
	}
	if cfg.RedisPort != "6380" {
		t.Errorf("expected RedisPort 6380, got %s", cfg.RedisPort)
	}
	if cfg.RedisAddr() != "redis-server:6380" {
		t.Errorf("expected RedisAddr redis-server:6380, got %s", cfg.RedisAddr())
	}
	if cfg.JWTSecret != "super-secret-jwt" {
		t.Errorf("expected JWTSecret super-secret-jwt, got %s", cfg.JWTSecret)
	}
}


