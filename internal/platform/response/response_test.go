package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
)

func TestJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	response.JSON(w, http.StatusOK, map[string]string{"name": "odoo_go"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !env.Success {
		t.Errorf("expected success to be true")
	}

	dataMap, ok := env.Data.(map[string]any)
	if !ok || dataMap["name"] != "odoo_go" {
		t.Errorf("unexpected data: %+v", env.Data)
	}
}

func TestPaginated_Success(t *testing.T) {
	w := httptest.NewRecorder()
	items := []string{"item1", "item2"}
	meta := map[string]any{"page": 1, "total": 2}

	response.Paginated(w, http.StatusOK, items, meta)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if !env.Success || env.Meta == nil {
		t.Errorf("expected success=true and non-nil meta")
	}
}

func TestError_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	appErr := platformerrors.NotFound("partner not found")

	response.Error(w, appErr)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if env.Success {
		t.Errorf("expected success=false")
	}
	if env.Error == nil || env.Error.Code != platformerrors.CodeNotFound {
		t.Errorf("expected code NOT_FOUND, got %+v", env.Error)
	}
}

func TestError_StandardError(t *testing.T) {
	w := httptest.NewRecorder()
	stdErr := errors.New("something crashed")

	response.Error(w, stdErr)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if env.Success {
		t.Errorf("expected success=false")
	}
	if env.Error == nil || env.Error.Code != platformerrors.CodeInternal {
		t.Errorf("expected code INTERNAL_ERROR, got %+v", env.Error)
	}
}
