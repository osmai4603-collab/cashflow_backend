package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cashflow_backend/internal/infrastructure/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load(config.WithNoFile())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "8070" {
		t.Errorf("expected default Port 8070, got %s", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("expected default ReadTimeout 5s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.ReadHeaderTimeout != 2*time.Second {
		t.Errorf("expected default ReadHeaderTimeout 2s, got %v", cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Server.WriteTimeout != 10*time.Second {
		t.Errorf("expected default WriteTimeout 10s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 120*time.Second {
		t.Errorf("expected default IdleTimeout 120s, got %v", cfg.Server.IdleTimeout)
	}
	if cfg.Server.DrainDuration != 5*time.Second {
		t.Errorf("expected default DrainDuration 5s, got %v", cfg.Server.DrainDuration)
	}
	if cfg.Server.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected default ShutdownTimeout 10s, got %v", cfg.Server.ShutdownTimeout)
	}
	if cfg.Server.MaxHeaderBytes != 1<<20 {
		t.Errorf("expected MaxHeaderBytes 1MB, got %d", cfg.Server.MaxHeaderBytes)
	}
	if cfg.Server.ProxyMode {
		t.Error("expected ProxyMode false by default")
	}
	if cfg.Limit.TimeReal != 120*time.Second {
		t.Errorf("expected default TimeReal 120s, got %v", cfg.Limit.TimeReal)
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("READ_TIMEOUT_SECONDS", "8")
	t.Setenv("DRAIN_SECONDS", "3")

	cfg, err := config.Load(config.WithNoFile())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("expected Port 9090, got %s", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 8*time.Second {
		t.Errorf("expected ReadTimeout 8s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.DrainDuration != 3*time.Second {
		t.Errorf("expected DrainDuration 3s, got %v", cfg.Server.DrainDuration)
	}
}

func TestLoad_PostgresDefaults(t *testing.T) {
	cfg, err := config.Load(config.WithNoFile())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Database.StorageDriver != "postgres" {
		t.Errorf("expected default StorageDriver postgres, got %s", cfg.Database.StorageDriver)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("expected default Host localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("expected default Port 5432, got %s", cfg.Database.Port)
	}
	if cfg.Database.Name != "cashflow" {
		t.Errorf("expected default Name cashflow, got %s", cfg.Database.Name)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("expected default MaxConns 25, got %d", cfg.Database.MaxConns)
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg, err := config.Load(config.WithNoFile())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	expected := "postgres://postgres:postgres@localhost:5432/cashflow?sslmode=disable"
	if cfg.DSN() != expected {
		t.Errorf("expected DSN %s, got %s", expected, cfg.DSN())
	}

	cfg.Database.DatabaseURL = "postgres://custom:pass@remote:5433/customdb?sslmode=require"
	if cfg.DSN() != cfg.Database.DatabaseURL {
		t.Errorf("expected DatabaseURL override %s, got %s", cfg.Database.DatabaseURL, cfg.DSN())
	}
}

func TestLoad_AppAndSecurityDefaults(t *testing.T) {
	cfg, err := config.Load(config.WithNoFile())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.App.Name != "cashflow_go_backend" {
		t.Errorf("expected default AppName cashflow_go_backend, got %s", cfg.App.Name)
	}
	if cfg.App.Version != "0.1.0" {
		t.Errorf("expected default AppVersion 0.1.0, got %s", cfg.App.Version)
	}
	if cfg.App.CompanyMode != "single" {
		t.Errorf("expected default CompanyMode single, got %s", cfg.App.CompanyMode)
	}
	if cfg.Cache.RedisHost != "localhost" {
		t.Errorf("expected default RedisHost localhost, got %s", cfg.Cache.RedisHost)
	}
	if cfg.Cache.RedisPort != "6379" {
		t.Errorf("expected default RedisPort 6379, got %s", cfg.Cache.RedisPort)
	}
	if cfg.RedisAddr() != "localhost:6379" {
		t.Errorf("expected RedisAddr localhost:6379, got %s", cfg.RedisAddr())
	}
	if cfg.Auth.JWTSecret == "" {
		t.Error("expected non-empty default JWTSecret")
	}
	if cfg.HTTPAddr() != "0.0.0.0:8070" {
		t.Errorf("expected HTTPAddr 0.0.0.0:8070, got %s", cfg.HTTPAddr())
	}
}

func TestLoad_CustomAppAndSecurity(t *testing.T) {
	t.Setenv("APP_NAME", "my_erp")
	t.Setenv("APP_VERSION", "2.0.0")
	t.Setenv("COMPANY_MODE", "multi")
	t.Setenv("REDIS_HOST", "redis-server")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("JWT_SECRET", "super-secret-jwt")
	t.Setenv("PROXY_MODE", "true")

	cfg, err := config.Load(config.WithNoFile())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.App.Name != "my_erp" {
		t.Errorf("expected AppName my_erp, got %s", cfg.App.Name)
	}
	if cfg.App.Version != "2.0.0" {
		t.Errorf("expected AppVersion 2.0.0, got %s", cfg.App.Version)
	}
	if cfg.App.CompanyMode != "multi" {
		t.Errorf("expected CompanyMode multi, got %s", cfg.App.CompanyMode)
	}
	if cfg.Cache.RedisHost != "redis-server" {
		t.Errorf("expected RedisHost redis-server, got %s", cfg.Cache.RedisHost)
	}
	if cfg.Cache.RedisPort != "6380" {
		t.Errorf("expected RedisPort 6380, got %s", cfg.Cache.RedisPort)
	}
	if cfg.RedisAddr() != "redis-server:6380" {
		t.Errorf("expected RedisAddr redis-server:6380, got %s", cfg.RedisAddr())
	}
	if cfg.Auth.JWTSecret != "super-secret-jwt" {
		t.Errorf("expected JWTSecret super-secret-jwt, got %s", cfg.Auth.JWTSecret)
	}
	if !cfg.Server.ProxyMode {
		t.Error("expected ProxyMode true from PROXY_MODE env")
	}
}

func TestLoad_FileLayer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cashflow.json")
	file := `{
  "server": { "port": "9999", "proxy_mode": true },
  "app": { "name": "file_app" }
}`
	if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("HTTP_INTERFACE", "10.0.0.1")
	cfg, err := config.Load(config.WithFile(path))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "9999" {
		t.Errorf("expected file Port 9999, got %s", cfg.Server.Port)
	}
	if !cfg.Server.ProxyMode {
		t.Error("expected file ProxyMode true")
	}
	if cfg.Server.Interface != "10.0.0.1" {
		t.Errorf("expected env Interface 10.0.0.1, got %s", cfg.Server.Interface)
	}
	// Non-file keys keep defaults.
	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("expected default ReadTimeout retained, got %v", cfg.Server.ReadTimeout)
	}
}

func TestLoad_ValidateFailsFast(t *testing.T) {
	t.Setenv("COMPANY_MODE", "bogus")

	_, err := config.Load(config.WithNoFile())
	if err == nil {
		t.Fatal("expected validation error for invalid COMPANY_MODE")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.json")

	cfg, err := config.Load(config.WithNoFile())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	cfg.Server.Port = "8123"

	store, err := config.NewFileStore(path, true)
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	reloaded, err := config.Load(config.WithFile(path))
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	if reloaded.Server.Port != "8123" {
		t.Errorf("expected reloaded Port 8123, got %s", reloaded.Server.Port)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected 0600 permissions, got %o", perm)
	}
}