package accounting

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	platformerr "cashflow_backend/internal/platform/errors"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func newAccountingTestServer(repo *mockAccountingRepository) *httptest.Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := chi.NewRouter()
	RegisterRoutes(router, NewUseCases(repo, nil, logger), logger)
	return httptest.NewServer(router)
}

func postJSON(t *testing.T, client *http.Client, url string, value any) *http.Response {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	response, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return response
}

func TestAccountingHandlersCreateAccountAndValidatePayload(t *testing.T) {
	repo := newMockAccountingRepository()
	server := newAccountingTestServer(repo)
	defer server.Close()

	response := postJSON(t, server.Client(), server.URL+"/api/v1/accounting/accounts", CreateAccountRequest{
		Code: "1000", Name: "Cash", Type: "asset",
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("create account status = %d, want %d; body: %s", response.StatusCode, http.StatusCreated, body)
	}
	var account AccountResponse
	if err := json.NewDecoder(response.Body).Decode(&account); err != nil {
		t.Fatalf("decode account response: %v", err)
	}
	if account.Code != "1000" || account.ID == uuid.Nil {
		t.Fatalf("unexpected account response: %+v", account)
	}

	response, err := server.Client().Get(server.URL + "/api/v1/accounting/accounts")
	if err != nil {
		t.Fatalf("list accounts request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list accounts status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	var accounts []AccountResponse
	if err := json.NewDecoder(response.Body).Decode(&accounts); err != nil {
		t.Fatalf("decode accounts response: %v", err)
	}
	if len(accounts) != 1 || accounts[0].ID != account.ID {
		t.Fatalf("listed accounts = %+v, want the created account", accounts)
	}

	response = postJSON(t, server.Client(), server.URL+"/api/v1/accounting/accounts", CreateAccountRequest{
		Code: "1000", Name: "Duplicate cash", Type: "asset",
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate account status = %d, want %d", response.StatusCode, http.StatusConflict)
	}

	response = postJSON(t, server.Client(), server.URL+"/api/v1/accounting/accounts", CreateAccountRequest{})
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid account status = %d, want %d", response.StatusCode, http.StatusUnprocessableEntity)
	}
	var apiError platformerr.HTTPErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
		t.Fatalf("decode validation error: %v", err)
	}
	if apiError.Code != platformerr.CodeInvalid || apiError.Message == "" {
		t.Fatalf("unexpected validation response: %+v", apiError)
	}
}

func TestAccountingHandlersMapMoveErrorsAndValidateReportDates(t *testing.T) {
	repo := newMockAccountingRepository()
	repo.move = &AccountMove{ID: uuid.New(), State: MoveStatePosted}
	server := newAccountingTestServer(repo)
	defer server.Close()

	response, err := server.Client().Post(server.URL+"/api/v1/accounting/moves/"+repo.move.ID.String()+"/post", "application/json", nil)
	if err != nil {
		t.Fatalf("post move request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("posted move status = %d, want %d", response.StatusCode, http.StatusConflict)
	}
	var apiError platformerr.HTTPErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if apiError.Code != platformerr.CodeConflict {
		t.Fatalf("conflict response code = %q, want %q", apiError.Code, platformerr.CodeConflict)
	}

	response, err = server.Client().Get(server.URL + "/api/v1/accounting/reports/trial-balance?from=invalid&to=2026-01-31")
	if err != nil {
		t.Fatalf("get trial balance request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid report date status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

func TestAccountingHandlersMapClosedPeriodToConflict(t *testing.T) {
	repo := newMockAccountingRepository()
	journalID, accountID := uuid.New(), uuid.New()
	repo.journal = &Journal{ID: journalID, IsActive: true}
	repo.period = &FiscalPeriod{State: PeriodStateHardClosed}
	repo.accountsByID[accountID] = &Account{ID: accountID, IsActive: true}
	server := newAccountingTestServer(repo)
	defer server.Close()

	response := postJSON(t, server.Client(), server.URL+"/api/v1/accounting/moves", CreateMoveRequest{
		JournalID: journalID,
		Date:      "2026-01-15",
		Lines: []MoveLineRequest{
			{AccountID: accountID, Debit: decimal.NewFromInt(10)},
			{AccountID: accountID, Credit: decimal.NewFromInt(10)},
		},
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("closed-period move status = %d, want %d; body: %s", response.StatusCode, http.StatusConflict, body)
	}
}

func TestAccountingHandlersRejectMalformedJSONAndFindMissingMoves(t *testing.T) {
	server := newAccountingTestServer(newMockAccountingRepository())
	defer server.Close()

	response, err := server.Client().Post(
		server.URL+"/api/v1/accounting/fiscal-years",
		"application/json",
		bytes.NewBufferString(`{"year":`),
	)
	if err != nil {
		t.Fatalf("send malformed JSON request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed JSON status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}

	response, err = server.Client().Get(server.URL + "/api/v1/accounting/moves/" + uuid.NewString())
	if err != nil {
		t.Fatalf("get missing move request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("missing move status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}
