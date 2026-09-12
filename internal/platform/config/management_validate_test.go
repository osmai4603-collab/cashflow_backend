package config

import "testing"

func TestValidate_ManagementDefaults(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("defaults must validate: %v", err)
	}
	if !cfg.Management.Enabled {
		t.Fatal("management listener should be enabled by default")
	}
	if cfg.Management.Interface != "127.0.0.1" {
		t.Fatalf("management must default to loopback, got %q", cfg.Management.Interface)
	}
}

func TestValidate_ManagementPortRange(t *testing.T) {
	cfg := Defaults()
	cfg.Management.Port = "0"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for management port 0")
	}

	cfg = Defaults()
	cfg.Management.Port = "99999"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for out-of-range management port")
	}

	cfg = Defaults()
	cfg.Management.Port = "not-a-port"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for non-numeric management port")
	}
}

func TestValidate_ManagementPortConflictsWithPublic(t *testing.T) {
	cfg := Defaults()
	cfg.Management.Port = "8070" // same as the public server port
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when management port equals public server port")
	}
}

func TestValidate_ManagementRequireAuthRequiresToken(t *testing.T) {
	cfg := Defaults()
	cfg.Management.RequireAuth = true
	cfg.Management.AuthToken = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when require_auth is set without a token")
	}
}

func TestValidate_ManagementWildcardInterfaceRequiresAuth(t *testing.T) {
	for _, iface := range []string{"0.0.0.0", "::", ""} {
		cfg := Defaults()
		cfg.Management.Interface = iface
		if err := cfg.Validate(); err == nil {
			t.Errorf("expected error binding management to %q without auth", iface)
		}
	}

	cfg := Defaults()
	cfg.Management.Interface = "0.0.0.0"
	cfg.Management.RequireAuth = true
	cfg.Management.AuthToken = "t0k3n"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("wildcard interface with auth should validate: %v", err)
	}
}

func TestValidate_ManagementDisabledSkipsChecks(t *testing.T) {
	cfg := Defaults()
	cfg.Management.Enabled = false
	cfg.Management.Port = "0"
	cfg.Management.RequireAuth = true
	cfg.Management.AuthToken = ""
	cfg.Management.Interface = "0.0.0.0"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled management must skip validation: %v", err)
	}
}
