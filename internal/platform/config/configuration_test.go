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

func TestAccountingGovernanceSettingsLoadFromEnvironment(t *testing.T) {
	values := map[string]string{
		"ACCOUNTING_MAKER_CHECKER_THRESHOLD": "12500.75",
		"ACCOUNTING_FUNCTIONAL_CURRENCY":     "USD",
	}
	cfg, err := NewEngine(WithEnvLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Accounting.MakerCheckerThreshold != "12500.75" || cfg.Accounting.FunctionalCurrency != "USD" {
		t.Fatalf("accounting settings = %+v", cfg.Accounting)
	}
}

func TestAccountingGovernanceSettingsValidation(t *testing.T) {
	for _, test := range []struct {
		name      string
		threshold string
		currency  string
	}{
		{name: "invalid amount", threshold: "not-a-number", currency: "SAR"},
		{name: "negative amount", threshold: "-1", currency: "SAR"},
		{name: "unsupported currency", threshold: "100", currency: "XYZ"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := DefaultConfiguration()
			cfg.Accounting.MakerCheckerThreshold = test.threshold
			cfg.Accounting.FunctionalCurrency = test.currency
			if err := NewValidationEngine().Validate(cfg); err == nil {
				t.Fatal("Validate() accepted invalid accounting settings")
			}
		})
	}
}
