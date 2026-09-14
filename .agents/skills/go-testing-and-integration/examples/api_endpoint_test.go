package examples

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ItemRequest represents the incoming payload.
type ItemRequest struct {
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
}

// ItemResponse represents the unified response envelope.
type ItemResponse struct {
	Data struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Amount int64  `json:"amount"`
	} `json:"data"`
}

// ItemHandler handles item creation.
func ItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"type":"about:blank","title":"Method Not Allowed","status":405}`, http.StatusMethodNotAllowed)
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401}`, http.StatusUnauthorized)
		return
	}

	var req ItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Amount <= 0 {
		http.Error(w, `{"type":"about:blank","title":"Unprocessable Entity","status":422}`, http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{
			"id":     "item_001",
			"name":   req.Name,
			"amount": req.Amount,
		},
	})
}

// TestItemHandler demonstrates an in-memory HTTP integration test.
func TestItemHandler(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		method         string
		authHeader     string
		body           string
		expectedStatus int
	}{
		{
			name:           "successful creation",
			method:         http.MethodPost,
			authHeader:     "Bearer test-token",
			body:           `{"name":"Widget","amount":1500}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing auth returns 401",
			method:         http.MethodPost,
			authHeader:     "",
			body:           `{"name":"Widget","amount":1500}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid validation returns 422",
			method:         http.MethodPost,
			authHeader:     "Bearer test-token",
			body:           `{"name":"","amount":0}`,
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, "/items", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			rec := httptest.NewRecorder()
			ItemHandler(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if tc.expectedStatus == http.StatusCreated {
				var resp ItemResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response JSON: %v", err)
				}
				if resp.Data.ID != "item_001" {
					t.Errorf("expected ID 'item_001', got %s", resp.Data.ID)
				}
			}
		})
	}
}
