package analytichttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	analytichttp "cashflow_backend/internal/adapters/http/analytic"
	analyticstorage "cashflow_backend/internal/adapters/storage/analytic"
	analyticusecase "cashflow_backend/internal/usecase/analytic"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*chi.Mux, *analyticstorage.MemoryRepo) {
	repo := analyticstorage.NewMemoryRepo()
	uc := analyticusecase.New(repo, nil)
	h := analytichttp.NewHandler(uc, nil)

	r := chi.NewRouter()
	analytichttp.RegisterRoutes(r, h)
	return r, repo
}

func doJSON(r *chi.Mux, method, path string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decodeData(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode envelope: %v (body: %s)", err, rec.Body.String())
	}
	if err := json.Unmarshal(envelope.Data, dst); err != nil {
		t.Fatalf("failed to decode data: %v (body: %s)", err, rec.Body.String())
	}
}

func TestHandler_PlanFlowAndRelevantPlans(t *testing.T) {
	r, _ := setupTestServer()

	rec := doJSON(r, http.MethodGet, "/analytic-plans", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 listing plans, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create a new plan under the seeded "Project" plan.
	rec = doJSON(r, http.MethodPost, "/analytic-plans", map[string]any{
		"name":                  "Marketing",
		"description":           "Marketing campaigns",
		"parent_id":             1,
		"sequence":              30,
		"default_applicability": "optional",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating plan, got %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		PlanID int64 `json:"id"`
	}
	decodeData(t, rec, &created)
	if created.PlanID == 0 {
		t.Fatal("expected non-zero plan ID")
	}

	// Structure should report the root plan + its accounts.
	rec = doJSON(r, http.MethodGet, "/analytic-plans/1/structure", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for structure, got %d: %s", rec.Code, rec.Body.String())
	}
	var structure struct {
		Plan     PlanResp `json:"plan"`
		Accounts []any    `json:"accounts"`
	}
	decodeData(t, rec, &structure)
	if structure.Plan.ID != 1 {
		t.Fatalf("expected plan 1 in structure, got %d", structure.Plan.ID)
	}

	// Seed plan 1 becomes mandatory for company 7 (G2).
	rec = doJSON(r, http.MethodPut, "/analytic-plans/1/applicability", map[string]any{
		"business_domain": "general",
		"applicability":   "mandatory",
		"company_id":      7,
		"sequence":        1,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 setting applicability, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(r, http.MethodGet, "/analytic/relevant-plans?company_id=7&business_domain=general", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 relevant plans, got %d: %s", rec.Code, rec.Body.String())
	}
	var relevant []RelevantPlanResp
	decodeData(t, rec, &relevant)
	foundPlan := false
	for _, p := range relevant {
		if p.ID == 1 {
			foundPlan = true
			if p.Applicability != "mandatory" {
				t.Errorf("expected relevant plan 1 to be mandatory, got %s", p.Applicability)
			}
			if p.ColumnName != "account_id" {
				t.Errorf("expected project plan column_name account_id, got %s", p.ColumnName)
			}
		}
	}
	if !foundPlan {
		t.Fatalf("expected relevant plan 1 for company 7, got %+v", relevant)
	}

	// Forced plans are dropped.
	rec = doJSON(r, http.MethodGet, "/analytic/relevant-plans?company_id=7&forced_plan_ids=1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 relevant plans with forced, got %d: %s", rec.Code, rec.Body.String())
	}
	var afterForced []RelevantPlanResp
	decodeData(t, rec, &afterForced)
	for _, p := range afterForced {
		if p.ID == 1 {
			t.Fatal("expected plan 1 to be dropped when forced")
		}
	}
}

func TestHandler_AccountsLinesAndBalance(t *testing.T) {
	r, _ := setupTestServer()

	// Create an account on plan 2 (Departments).
	rec := doJSON(r, http.MethodPost, "/analytic-accounts", map[string]any{
		"name":    "Finance",
		"code":    "FIN",
		"plan_id": 2,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating account, got %d: %s", rec.Code, rec.Body.String())
	}

	// Register a credit line on seeded account 1.
	rec = doJSON(r, http.MethodPost, "/analytic-lines", map[string]any{
		"name":         "Consulting revenue",
		"date":         "2026-09-07T00:00:00Z",
		"amount":       500.0,
		"user_id":      11,
		"company_id":   7,
		"account_id":   1,
		"distribution": map[string]float64{"1": 100},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 registering line, got %d: %s", rec.Code, rec.Body.String())
	}

	// Manual line without distribution (no mandate enforced).
	rec = doJSON(r, http.MethodPost, "/analytic-lines", map[string]any{
		"name":       "Stationery",
		"date":       "2026-09-06T00:00:00Z",
		"amount":     -200.0,
		"user_id":    11,
		"company_id": 7,
		"account_id": 1,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 registering manual line, got %d: %s", rec.Code, rec.Body.String())
	}

	// Balance: credit 500, debit 200, balance 300.
	rec = doJSON(r, http.MethodGet, "/analytic-accounts/1/balance", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for balance, got %d: %s", rec.Code, rec.Body.String())
	}
	var balance struct {
		Debit    float64 `json:"debit"`
		Credit   float64 `json:"credit"`
		Balance  float64 `json:"balance"`
		Currency string  `json:"currency"`
	}
	decodeData(t, rec, &balance)
	if balance.Credit != 500 || balance.Debit != 200 || balance.Balance != 300 {
		t.Fatalf("expected credit 500, debit 200, balance 300; got %+v", balance)
	}

	// List lines for account 1.
	rec = doJSON(r, http.MethodGet, "/analytic-accounts/1/lines", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for account lines, got %d: %s", rec.Code, rec.Body.String())
	}
	var lines []LineResp
	decodeData(t, rec, &lines)
	if len(lines) != 2 {
		t.Fatalf("expected 2 account lines, got %d", len(lines))
	}
}

type PlanResp struct {
	ID int64 `json:"id"`
}

type RelevantPlanResp struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Applicability string `json:"applicability"`
	ColumnName    string `json:"column_name"`
}

type LineResp struct {
	ID      int64   `json:"id"`
	Amount  float64 `json:"amount"`
	Account int64   `json:"account_id"`
}

func TestHandler_MoveLineDistributionAndModels(t *testing.T) {
	r, _ := setupTestServer()

	// Make plan 1 mandatory for company 7 so distribution is enforced.
	rec := doJSON(r, http.MethodPut, "/analytic-plans/1/applicability", map[string]any{
		"business_domain": "general",
		"applicability":   "mandatory",
		"company_id":      7,
		"sequence":        1,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 setting applicability, got %d: %s", rec.Code, rec.Body.String())
	}

	// Distribute a 1000-amount move line across two keys referencing the
	// mandatory plan's account (50/50 split = 100% coverage for plan 1).
	rec = doJSON(r, http.MethodPost, "/move-lines/42/analytic", map[string]any{
		"general_account_id": 1010,
		"name":               "Invoice line 1",
		"date":               "2026-09-07T00:00:00Z",
		"user_id":            11,
		"company_id":         7,
		"currency":           "USD",
		"amount":             1000.0,
		"source":             "invoice",
		"distribution":       map[string]float64{"1": 50, "1,2": 50},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 distributing move line, got %d: %s", rec.Code, rec.Body.String())
	}
	var split []LineResp
	decodeData(t, rec, &split)
	if len(split) != 2 {
		t.Fatalf("expected 2 split lines, got %d", len(split))
	}

	// Fetch analytic lines for move line 42.
	rec = doJSON(r, http.MethodGet, "/move-lines/42/analytic", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 listing move line analytic, got %d: %s", rec.Code, rec.Body.String())
	}
	var moveLines []LineResp
	decodeData(t, rec, &moveLines)
	if len(moveLines) != 2 {
		t.Fatalf("expected 2 move-line analytic entries, got %d", len(moveLines))
	}

	// From now on, plan 1 is unavailable for company 9, so partial distribution fails.
	doJSON(r, http.MethodPut, "/analytic-plans/1/applicability", map[string]any{
		"business_domain": "general",
		"applicability":   "unavailable",
		"company_id":      9,
		"sequence":        2,
	})

	// Distribution model creation + match (G7).
	rec = doJSON(r, http.MethodPost, "/analytic-distribution-models", map[string]any{
		"sequence":     10,
		"distribution": map[string]float64{"1": 100},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating distribution model, got %d: %s", rec.Code, rec.Body.String())
	}
	var model struct {
		ModelID int64 `json:"id"`
	}
	decodeData(t, rec, &model)
	if model.ModelID == 0 {
		t.Fatal("expected non-zero distribution model ID")
	}

	rec = doJSON(r, http.MethodPost, "/analytic-distribution-models/match", map[string]any{
		"company_id": 7,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 matching distribution model, got %d: %s", rec.Code, rec.Body.String())
	}
	var matched map[string]float64
	decodeData(t, rec, &matched)
	if matched["1"] != 100 {
		t.Fatalf("expected matched distribution {1:100}, got %+v", matched)
	}

	// Project plan config (G8): current default is 1.
	rec = doJSON(r, http.MethodGet, "/analytic/project-plan", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for project plan, got %d: %s", rec.Code, rec.Body.String())
	}
	var projectPlan struct {
		PlanID int64 `json:"plan_id"`
	}
	decodeData(t, rec, &projectPlan)
	if projectPlan.PlanID != 1 {
		t.Fatalf("expected default project plan 1, got %d", projectPlan.PlanID)
	}
}

func TestHandler_InvalidInputs(t *testing.T) {
	r, _ := setupTestServer()

	rec := doJSON(r, http.MethodPost, "/analytic-plans", map[string]any{"name": ""})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty plan name, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(r, http.MethodGet, "/analytic-plans/999/structure", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing plan, got %d: %s", rec.Code, rec.Body.String())
	}
}
