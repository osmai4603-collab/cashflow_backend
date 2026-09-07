package config

import "testing"

func TestSetAndVerifyAdminPassword(t *testing.T) {
	cfg := Defaults()
	if cfg.VerifyAdminPassword("hunter2") {
		t.Error("expected empty hash to reject all passwords")
	}

	if err := cfg.SetAdminPassword("hunter2", DefaultPBKDF2Rounds); err != nil {
		t.Fatalf("SetAdminPassword: %v", err)
	}

	if !cfg.VerifyAdminPassword("hunter2") {
		t.Error("expected correct password to verify")
	}
	if cfg.VerifyAdminPassword("wrong-password") {
		t.Error("expected wrong password to be rejected")
	}

	// Re-set with a new value; old password must stop matching.
	if err := cfg.SetAdminPassword("new-pass", 1000); err != nil {
		t.Fatalf("SetAdminPassword(new): %v", err)
	}
	if cfg.VerifyAdminPassword("hunter2") {
		t.Error("expected old password to stop verifying after re-set")
	}
	if !cfg.VerifyAdminPassword("new-pass") {
		t.Error("expected new password to verify")
	}
}

func TestDecodePBKDF2_Invalid(t *testing.T) {
	for _, encoded := range []string{"", "not-a-hash", "$foo$1$aaaa$bbbb"} {
		if _, _, _, err := decodePBKDF2(encoded); err == nil {
			t.Errorf("expected error for invalid hash %q", encoded)
		}
	}
}