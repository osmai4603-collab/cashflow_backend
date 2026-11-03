package config

import (
	"net/url"
	"testing"
)

func TestConfigurationDSN(t *testing.T) {
	cfg := &Configuration{
		Database: DatabaseSettings{
			Host:     "db.example",
			Port:     "5432",
			Name:     "cashflow",
			User:     "cashflow",
			Password: "p@ss:/word",
			SSLMode:  "verify-ca",
		},
	}

	parsed, err := url.Parse(cfg.DSN())
	if err != nil {
		t.Fatalf("DSN is not a valid URL: %v", err)
	}
	if parsed.Scheme != "postgres" {
		t.Errorf("expected postgres scheme, got %q", parsed.Scheme)
	}
	if parsed.Host != "db.example:5432" {
		t.Errorf("expected host db.example:5432, got %q", parsed.Host)
	}
	if parsed.Path != "/cashflow" {
		t.Errorf("expected database path /cashflow, got %q", parsed.Path)
	}
	if parsed.Query().Get("sslmode") != "verify-ca" {
		t.Errorf("expected sslmode verify-ca, got %q", parsed.Query().Get("sslmode"))
	}
	if parsed.User.Username() != "cashflow" {
		t.Errorf("expected user cashflow, got %q", parsed.User.Username())
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss:/word" {
		t.Errorf("password was not safely encoded in DSN")
	}
}

func TestConfigurationDSNUsesDatabaseURL(t *testing.T) {
	const databaseURL = "postgres://custom-host:5432/custom_db?sslmode=require"
	cfg := &Configuration{
		Database: DatabaseSettings{DatabaseURL: databaseURL},
	}

	if got := cfg.DSN(); got != databaseURL {
		t.Errorf("expected configured database URL %q, got %q", databaseURL, got)
	}
}
