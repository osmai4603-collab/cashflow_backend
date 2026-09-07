package config

import "testing"

func TestValidate_CompanyMode(t *testing.T) {
	cfg := Defaults()
	cfg.App.CompanyMode = "single"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("single mode should validate: %v", err)
	}
	cfg.App.CompanyMode = "multi"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("multi mode should validate: %v", err)
	}
	cfg.App.CompanyMode = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid company mode")
	}
}

func TestValidate_BadPort(t *testing.T) {
	cfg := Defaults()
	cfg.Server.Port = "99999"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for out-of-range port")
	}
	cfg.Server.Port = "abc"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for non-numeric port")
	}
}

func TestValidate_SyslogFileConflict(t *testing.T) {
	cfg := Defaults()
	cfg.Log.Syslog = true
	cfg.Log.File = "/var/log/app.log"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for syslog+logfile conflict")
	}
}

func TestValidate_Security(t *testing.T) {
	cfg := Defaults()
	cfg.Auth.JWTSecret = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty JWT secret")
	}
}

func TestValidate_DriverAndLimits(t *testing.T) {
	cfg := Defaults()
	cfg.Database.StorageDriver = "mysql"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for unsupported storage driver")
	}

	cfg = Defaults()
	cfg.Limit.MemorySoft = 900
	cfg.Limit.MemoryHard = 100
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when soft > hard memory limit")
	}
}