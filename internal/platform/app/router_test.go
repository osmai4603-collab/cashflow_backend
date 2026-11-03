package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestNewRouterRegistersAccountingRoutes(t *testing.T) {
	router := NewRouter(nil, nil, nil)
	routes := make(map[string]bool)
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk router: %v", err)
	}

	for _, route := range []string{
		"POST /api/v1/accounting/accounts",
		"GET /api/v1/accounting/accounts",
		"GET /api/v1/accounting/accounts/tree",
		"POST /api/v1/accounting/fiscal-years",
		"POST /api/v1/accounting/fiscal-periods/{id}/state",
		"POST /api/v1/accounting/moves",
		"GET /api/v1/accounting/moves/{id}",
		"POST /api/v1/accounting/moves/{id}/post",
		"POST /api/v1/accounting/moves/{id}/reverse",
		"GET /api/v1/accounting/reports/trial-balance",
		"POST /api/v1/accounting/subledger/events",
	} {
		if !routes[route] {
			t.Errorf("router is missing accounting route %q", route)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/accounting/accounts", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured accounting route status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
