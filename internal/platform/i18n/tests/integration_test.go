package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
	"cashflow_backend/internal/platform/response"
)

func TestI18nIntegration(t *testing.T) {
	// 1. Setup translations
	i18n.LoadTranslations("ar_SA", map[string]string{
		"Product name is required": "اسم المنتج مطلوب",
		"internal server error":    "خطأ داخلي في الخادم",
	})

	tests := []struct {
		name           string
		acceptLang     string
		queryLang      string
		errorMsg       string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "English Default",
			acceptLang:     "en-US",
			errorMsg:       "Product name is required",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Product name is required",
		},
		{
			name:           "Arabic via Header",
			acceptLang:     "ar-SA",
			errorMsg:       "Product name is required",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "اسم المنتج مطلوب",
		},
		{
			name:           "Arabic via Query (Overriding Header)",
			acceptLang:     "en-US",
			queryLang:      "ar_SA",
			errorMsg:       "Product name is required",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "اسم المنتج مطلوب",
		},
		{
			name:           "Fallback for unknown error",
			acceptLang:     "ar-SA",
			errorMsg:       "Something went wrong", // Not in map
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a handler that throws the error
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err := platformerrors.Validation(tt.errorMsg, nil)
				response.Error(w, r, err)
			})

			// Wrap with I18n Middleware
			wrappedHandler := i18n.Middleware(handler)

			url := "/"
			if tt.queryLang != "" {
				url += "?lang=" + tt.queryLang
			}
			req := httptest.NewRequest("GET", url, nil)
			if tt.acceptLang != "" {
				req.Header.Set("Accept-Language", tt.acceptLang)
			}
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp response.Envelope
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Error.Message != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, resp.Error.Message)
			}
		})
	}
}

func TestTranslationString_DBCompatibility(t *testing.T) {
	// Verify TranslationString stores a normal string value and survives DB round-trips.
	ts := i18n.NewTranslation("Table")

	val, err := ts.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	bytes := val.([]byte)

	var ts2 i18n.TranslationString
	if err := ts2.Scan(bytes); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if ts2.Get("en_US") != "Table" {
		t.Errorf("expected 'Table', got %q", ts2.Get("en_US"))
	}
}
