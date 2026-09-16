package group

import "testing"

func TestPermission_Allows_CRUD(t *testing.T) {
	p := &Permission{CanRead: true, CanCreate: true, CanUpdate: true, CanDelete: false}

	if !p.Allows("read") {
		t.Error("expected read to be allowed")
	}
	if !p.Allows("create") {
		t.Error("expected create to be allowed")
	}
	if !p.Allows("update") {
		t.Error("expected update to be allowed")
	}
	if p.Allows("delete") {
		t.Error("expected delete to be denied")
	}
	if p.Allows("unlink") {
		t.Error("expected unlink to be denied when delete is false")
	}
}

func TestPermission_Allows_OdooAliases(t *testing.T) {
	p := &Permission{CanRead: true, CanUpdate: true, CanDelete: true}

	// Odoo vocabulary must map onto the same privilege bits.
	if !p.Allows("write") {
		t.Error("expected write to be allowed")
	}
	if !p.Allows("WRITE") {
		t.Error("expected case-insensitive write to be allowed")
	}
	if !p.Allows("unlink") {
		t.Error("expected unlink to be allowed")
	}
	if !p.Allows("  write  ") {
		t.Error("expected trimmed write to be allowed")
	}
}

func TestPermission_Allows_UnknownAction(t *testing.T) {
	p := &Permission{CanRead: true}
	if p.Allows("export") {
		t.Error("expected unknown action to be denied")
	}
	if p.Allows("") {
		t.Error("expected empty action to be denied")
	}
}