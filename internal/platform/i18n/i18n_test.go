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
