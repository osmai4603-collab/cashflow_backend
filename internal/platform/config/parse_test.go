package config

import (
	"reflect"
	"testing"
	"time"
)

func TestSetField_String(t *testing.T) {
	var s string
	field := reflect.ValueOf(&s).Elem()
	if err := setField(field, "hello"); err != nil {
		t.Fatalf("setField(string): %v", err)
	}
	if s != "hello" {
		t.Errorf("expected hello, got %s", s)
	}
}

func TestSetField_Bool(t *testing.T) {
	var b bool
	field := reflect.ValueOf(&b).Elem()
	for _, raw := range []string{"1", "true", "yes", "on"} {
		if err := setField(field, raw); err != nil || !b {
			t.Errorf("setField(bool %q): err=%v value=%v", raw, err, b)
		}
	}
	for _, raw := range []string{"0", "false", "no", "off"} {
		if err := setField(field, raw); err != nil || b {
			t.Errorf("setField(bool %q): err=%v value=%v", raw, err, b)
		}
	}
	if err := setField(field, "not-a-bool"); err == nil {
		t.Error("expected error for invalid bool")
	}
}

func TestSetField_Duration(t *testing.T) {
	var d time.Duration
	field := reflect.ValueOf(&d).Elem()
	if err := setField(field, "90s"); err != nil {
		t.Fatalf("setField(duration): %v", err)
	}
	if d != 90*time.Second {
		t.Errorf("expected 90s, got %v", d)
	}
	// Bare numeric seconds (legacy *_SECONDS env vars).
	if err := setField(field, "30"); err != nil {
		t.Fatalf("setField(duration seconds): %v", err)
	}
	if d != 30*time.Second {
		t.Errorf("expected 30s from bare number, got %v", d)
	}
}

func TestSetField_Slice(t *testing.T) {
	var list []string
	field := reflect.ValueOf(&list).Elem()
	if err := setField(field, "base, rpc ,web"); err != nil {
		t.Fatalf("setField(slice): %v", err)
	}
	if !reflect.DeepEqual(list, []string{"base", "rpc", "web"}) {
		t.Errorf("expected [base rpc web], got %v", list)
	}
	if err := setField(field, ""); err != nil {
		t.Fatalf("setField(empty slice): %v", err)
	}
	if list != nil {
		t.Errorf("expected nil slice for empty input, got %v", list)
	}
}

func TestSetField_Int(t *testing.T) {
	var n int
	field := reflect.ValueOf(&n).Elem()
	if err := setField(field, "42"); err != nil {
		t.Fatalf("setField(int): %v", err)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
	if err := setField(field, "not-a-number"); err == nil {
		t.Error("expected error for invalid int")
	}
}

func TestParseSlice_Empty(t *testing.T) {
	if got := parseSlice(" , "); got != nil {
		t.Errorf("expected nil for whitespace-only, got %v", got)
	}
}