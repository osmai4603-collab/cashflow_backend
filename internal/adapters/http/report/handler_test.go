package reporthttp_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	reporthttp "cashflow_backend/internal/adapters/http/report"
	reportstorage "cashflow_backend/internal/adapters/storage/report"
	"cashflow_backend/internal/domain/report"
	"cashflow_backend/internal/platform/response"
	reportusecase "cashflow_backend/internal/usecase/report"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(t *testing.T) chi.Router {
	_ = slog.New(slog.NewTextHandler(io.Discard, nil))

	repo := reportstorage.NewMemoryRepo()
	generator := reportusecase.NewReportGenerator(repo, repo)
	dashboard := reportusecase.NewDashboardUseCase(generator)
	handler := reporthttp.NewHandler(generator, dashboard)

	r := chi.NewRouter()
	reporthttp.RegisterRoutes(r, handler)
	return r
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) response.Envelope {
	t.Helper()
	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode envelope: %v", err)
	}
	return env
}

func TestReportHTTP_GetReport(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/reports/PL?company_id=1&date_from=2026-01-01&date_to=2026-09-09", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("failed to marshal report data: %v", err)
	}
	var result report.ReportResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal report result: %v", err)
	}
	if result.Name == "" {
		t.Fatalf("expected report name, got empty")
	}
	if len(result.Lines) == 0 {
		t.Fatalf("expected report lines, got none")
	}
}

func TestReportHTTP_GetDashboard(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/reports/dashboard?company_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	data, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("failed to marshal dashboard data: %v", err)
	}
	var dashboard report.DashboardData
	if err := json.Unmarshal(data, &dashboard); err != nil {
		t.Fatalf("failed to unmarshal dashboard: %v", err)
	}
	if dashboard.Title == "" {
		t.Fatalf("expected dashboard title, got empty")
	}
}

func TestReportHTTP_UnknownReport(t *testing.T) {
	r := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/reports/UNKNOWN", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}