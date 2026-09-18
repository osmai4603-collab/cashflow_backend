package examples

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"golang.org/x/text/language"
)

func TestI18nNegotiationMatrix(t *testing.T) {
	supported := []language.Tag{
		language.MustParse("ar-SA"),
		language.Arabic,
		language.AmericanEnglish,
		language.French,
	}
	fallback := language.MustParse("ar-SA")
	mgr := NewI18nManager(supported, fallback)

	tests := []struct {
		name             string
		queryLang        string
		customHeader     string
		acceptHeader     string
		expectedLangCode string
	}{
		{
			name:             "Query parameter takes precedence over all",
			queryLang:        "fr",
			customHeader:     "ar-SA",
			acceptHeader:     "en-US,en;q=0.9",
			expectedLangCode: "fr",
		},
		{
			name:             "Custom header takes precedence over Accept-Language",
			queryLang:        "",
			customHeader:     "en-US",
			acceptHeader:     "ar-SA,ar;q=0.9",
			expectedLangCode: "en-US",
		},
		{
			name:             "Standard Accept-Language exact match",
			queryLang:        "",
			customHeader:     "",
			acceptHeader:     "en-US,en;q=0.8",
			expectedLangCode: "en-US",
		},
		{
			name:             "Dialect fallback to parent language (ar-EG -> ar)",
			queryLang:        "",
			customHeader:     "",
			acceptHeader:     "ar-EG,ar;q=0.9",
			expectedLangCode: "ar",
		},
		{
			name:             "Unsupported language falls back to default",
			queryLang:        "",
			customHeader:     "",
			acceptHeader:     "ja-JP,ja;q=0.9",
			expectedLangCode: "ar-SA",
		},
		{
			name:             "Empty headers falls back to default",
			queryLang:        "",
			customHeader:     "",
			acceptHeader:     "",
			expectedLangCode: "ar-SA",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url := "/test"
			if tc.queryLang != "" {
				url += "?lang=" + tc.queryLang
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tc.customHeader != "" {
				req.Header.Set("X-App-Language", tc.customHeader)
			}
			if tc.acceptHeader != "" {
				req.Header.Set("Accept-Language", tc.acceptHeader)
			}

			w := httptest.NewRecorder()

			handler := mgr.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				locale := GetLocale(r.Context(), fallback)
				if locale.String() != tc.expectedLangCode {
					t.Errorf("expected locale %q, got %q", tc.expectedLangCode, locale.String())
				}
			}))

			handler.ServeHTTP(w, req)
			if contentLang := w.Header().Get("Content-Language"); contentLang != tc.expectedLangCode {
				t.Errorf("expected Content-Language %q, got %q", tc.expectedLangCode, contentLang)
			}
		})
	}
}

func TestI18nConcurrentSafety(t *testing.T) {
	supported := []language.Tag{
		language.MustParse("ar-SA"),
		language.English,
		language.French,
	}
	fallback := language.MustParse("ar-SA")
	mgr := NewI18nManager(supported, fallback)

	handler := mgr.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		printer := GetPrinter(r.Context(), fallback)
		_ = printer.Sprintf("test message %d", 42)
	}))

	const concurrency = 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			var lang string
			switch idx % 3 {
			case 0:
				lang = "ar-SA"
			case 1:
				lang = "en"
			case 2:
				lang = "fr"
			}
			req := httptest.NewRequest(http.MethodGet, "/test?lang="+lang, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}(i)
	}

	wg.Wait()
}
