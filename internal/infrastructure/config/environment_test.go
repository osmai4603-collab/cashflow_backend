package config_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/infrastructure/config"
	platconfig "cashflow_backend/internal/platform/config"
)

func TestLoadEnvironment_CanonicalAndAliases(t *testing.T) {
	cfg := platconfig.Defaults()

	env := map[string]string{
		"PGHOST":               "db-node-1",
		"PGPORT":               "5999",
		"PORT":                 "8443",
		"DB_MAX_CONNS":         "12",
		"READ_TIMEOUT_SECONDS": "20",
	}

	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
	if err := config.LoadEnvironment(cfg, lookup); err != nil {
		t.Fatalf("LoadEnvironment: %v", err)
	}

	if cfg.Database.Host != "db-node-1" {
		t.Errorf("expected PGHOST alias -> Host db-node-1, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != "5999" {
		t.Errorf("expected PGPORT alias -> Port 5999, got %s", cfg.Database.Port)
	}
	if cfg.Server.Port != "8443" {
		t.Errorf("expected Port 8443, got %s", cfg.Server.Port)
	}
	if cfg.Database.MaxConns != 12 {
		t.Errorf("expected MaxConns 12, got %d", cfg.Database.MaxConns)
	}
	if cfg.Server.ReadTimeout != 20*time.Second {
		t.Errorf("expected ReadTimeout 20s, got %v", cfg.Server.ReadTimeout)
	}
}

func TestLoadEnvironment_InvalidValue(t *testing.T) {
	cfg := platconfig.Defaults()
	env := map[string]string{"DB_MAX_CONNS": "not-a-number"}
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
	if err := config.LoadEnvironment(cfg, lookup); err == nil {
		t.Error("expected error for invalid DB_MAX_CONNS value")
	}
}

func TestLoadEnvironment_ListSplitting(t *testing.T) {
	cfg := platconfig.Defaults()
	env := map[string]string{"SERVER_WIDE_MODULES": "base, web, rpc"}
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
	if err := config.LoadEnvironment(cfg, lookup); err != nil {
		t.Fatalf("LoadEnvironment: %v", err)
	}
	if len(cfg.Runtime.ServerWideModules) != 3 {
		t.Fatalf("expected 3 server-wide modules, got %v", cfg.Runtime.ServerWideModules)
	}
	want := []string{"base", "web", "rpc"}
	for i, m := range want {
		if cfg.Runtime.ServerWideModules[i] != m {
			t.Errorf("module[%d] = %s, want %s", i, cfg.Runtime.ServerWideModules[i], m)
		}
	}
}

func TestLoadEnvironment_CanonicalBeatsAlias(t *testing.T) {
	cfg := platconfig.Defaults()
	env := map[string]string{
		"DB_HOST": "pg-main",
		"PGHOST":  "pg-replica",
	}
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
	if err := config.LoadEnvironment(cfg, lookup); err != nil {
		t.Fatalf("LoadEnvironment: %v", err)
	}
	if cfg.Database.Host != "pg-main" {
		t.Errorf("expected canonical DB_HOST to win, got %s", cfg.Database.Host)
	}
}