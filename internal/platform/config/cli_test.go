package config_test

import (
	"testing"

	"cashflow_backend/internal/platform/config"
)

func TestParseFlags_AppliesOverrides(t *testing.T) {
	flags, err := config.ParseFlags([]string{
		"--config", "/tmp/cashflow.json",
		"--http-port", "9191",
		"--proxy-mode",
		"-d", "erp_prod",
		"--load", "base,web,rpc",
		"-i", "sale,purchase",
		"--log-level", "debug",
		"--workers", "4",
		"--dev", "all",
		"--stop-after-init",
	})
	if err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	cfg := config.Defaults()
	flags.Apply(cfg)

	if cfg.Server.Port != "9191" {
		t.Errorf("expected Port 9191, got %s", cfg.Server.Port)
	}
	if !cfg.Server.ProxyMode {
		t.Error("expected ProxyMode true")
	}
	if cfg.Database.Name != "erp_prod" {
		t.Errorf("expected DB name erp_prod, got %s", cfg.Database.Name)
	}
	if len(cfg.Runtime.InitModules) != 2 || cfg.Runtime.InitModules[0] != "sale" {
		t.Errorf("expected init modules [sale purchase], got %v", cfg.Runtime.InitModules)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.Log.Level)
	}
	if cfg.Worker.Workers != 4 {
		t.Errorf("expected workers 4, got %d", cfg.Worker.Workers)
	}
	if len(cfg.Runtime.DevMode) != 1 || cfg.Runtime.DevMode[0] != "all" {
		t.Errorf("expected dev mode [all], got %v", cfg.Runtime.DevMode)
	}
	if !cfg.Runtime.StopAfterInit {
		t.Error("expected StopAfterInit true")
	}
	if flags.Config != "/tmp/cashflow.json" {
		t.Errorf("expected config path /tmp/cashflow.json, got %s", flags.Config)
	}
}

func TestParseFlags_RejectsPositionalArgs(t *testing.T) {
	if _, err := config.ParseFlags([]string{"up"}); err == nil {
		t.Error("expected error for positional arguments")
	}
}

func TestParseFlags_ShortSeparateForm(t *testing.T) {
	flags, err := config.ParseFlags([]string{"-u", "stock"})
	if err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	cfg := config.Defaults()
	flags.Apply(cfg)
	if len(cfg.Runtime.UpdateModules) != 1 || cfg.Runtime.UpdateModules[0] != "stock" {
		t.Errorf("expected update modules [stock], got %v", cfg.Runtime.UpdateModules)
	}
}