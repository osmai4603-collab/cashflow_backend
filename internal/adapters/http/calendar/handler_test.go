package calendarhttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	calendarstorage "cashflow_backend/internal/adapters/storage/calendar"
	calendarusecase "cashflow_backend/internal/usecase/calendar"

	"github.com/go-chi/chi/v5"
)

func TestCreateEventRoute(t *testing.T) {
	handler := NewHandler(calendarusecase.New(calendarstorage.NewMemoryRepo()))
	chiRouter := NewTestRouter(handler)
	body := []byte(`{"name":"Planning","start":"2026-09-12T09:00:00Z","stop":"2026-09-12T10:00:00Z","user_id":1,"company_id":1}`)
	request := httptest.NewRequest(http.MethodPost, "/calendar/events", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	chiRouter.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func NewTestRouter(handler *Handler) http.Handler {
	router := chi.NewRouter()
	RegisterRoutes(router, handler)
	return router
}
