package i18n

import (
	"context"
	"testing"
)

func TestTranslationString_Get(t *testing.T) {
	LoadTranslations("ar_SA", map[string]string{
		"Apple": "تفاح",
	})

	ts := NewTranslation("Apple")

	tests := []struct {
		lang     string
		expected string
	}{
		{"en_US", "Apple"},
		{"ar_SA", "تفاح"},
		{"fr_FR", "Apple"}, // Fallback to en_US
		{"", "Apple"},      // Fallback to en_US
	}

	for _, tt := range tests {
		if got := ts.Get(tt.lang); got != tt.expected {
			t.Errorf("TranslationString.Get(%q) = %q, want %q", tt.lang, got, tt.expected)
		}
	}
}

func TestTranslationString_GetLocalized(t *testing.T) {
	LoadTranslations("ar_SA", map[string]string{
		"Hello": "مرحبا",
	})

	ts := NewTranslation("Hello")

	ctx := context.WithValue(context.Background(), LangKey, "ar_SA")
	if got := ts.GetLocalized(ctx); got != "مرحبا" {
		t.Errorf("GetLocalized with ar_SA = %q, want %q", got, "مرحبا")
	}

	ctx = context.Background() // No lang in context
	if got := ts.GetLocalized(ctx); got != "Hello" {
		t.Errorf("GetLocalized with no context = %q, want %q", got, "Hello")
	}
}

func TestTranslationString_Scan_AcceptsByteSliceAndString(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		var ts TranslationString
		if err := ts.Scan(nil); err != nil {
			t.Fatalf("scan nil: %v", err)
		}
		if ts != "" {
			t.Errorf("expected empty string, got %q", ts)
		}
	})

	t.Run("[]byte JSONB map", func(t *testing.T) {
		var ts TranslationString
		if err := ts.Scan([]byte(`{"en_US":"New","ar_SA":"جديد"}`)); err != nil {
			t.Fatalf("scan []byte: %v", err)
		}
		if string(ts) != "New" {
			t.Errorf("expected en_US default New, got %q", ts)
		}
	})

	t.Run("string JSONB map", func(t *testing.T) {
		var ts TranslationString
		if err := ts.Scan(`{"en_US":"Won"}`); err != nil {
			t.Fatalf("scan string: %v", err)
		}
		if string(ts) != "Won" {
			t.Errorf("expected Won, got %q", ts)
		}
	})

	t.Run("plain string", func(t *testing.T) {
		var ts TranslationString
		if err := ts.Scan("Qualified"); err != nil {
			t.Fatalf("scan plain string: %v", err)
		}
		if string(ts) != "Qualified" {
			t.Errorf("expected Qualified, got %q", ts)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		var ts TranslationString
		if err := ts.Scan(42); err == nil {
			t.Fatal("expected error for unsupported scan type")
		}
	})
}
