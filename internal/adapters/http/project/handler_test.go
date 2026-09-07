package projecthttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	projectstorage "cashflow_backend/internal/adapters/storage/project"
	"cashflow_backend/internal/platform/auth"
	projectusecase "cashflow_backend/internal/usecase/project"

	"github.com/go-chi/chi/v5"
)

func TestCreateProjectUsesAuthenticatedCompany(t *testing.T) {
	repo := projectstorage.NewMemoryRepo()
	handler := NewHandler(projectusecase.New(repo))
	router := chi.NewRouter()
	RegisterRoutes(router, handler)

	request := httptest.NewRequest(http.MethodPost, "/projects/", strings.NewReader(`{"name":"API project","allow_dependencies":true}`))
	claims := &auth.UserClaims{UserID: 7, CompanyID: 42}
	request = request.WithContext(auth.WithClaims(context.Background(), claims))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool `json:"success"`
		Data struct {
			CompanyID int64 `json:"company_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || envelope.Data.CompanyID != 42 {
		t.Fatalf("unexpected response: %+v", envelope)
	}
}
