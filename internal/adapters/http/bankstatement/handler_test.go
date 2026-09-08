package bankstatementhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bankstatementhttp "cashflow_backend/internal/adapters/http/bankstatement"
	bankstatementstorage "cashflow_backend/internal/adapters/storage/bankstatement"
	"cashflow_backend/internal/domain/accounting"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	bankstatementusecase "cashflow_backend/internal/usecase/bankstatement"

	"github.com/go-chi/chi/v5"
)

type mockAccountingSvc struct{}

func (m *mockAccountingSvc) CreateJournalEntry(ctx context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: 100}, nil
}
func (m *mockAccountingSvc) PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: id, State: accounting.MoveStatePosted}, nil
}
func (m *mockAccountingSvc) GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: id}, nil
}
func (m *mockAccountingSvc) GetJournal(ctx context.Context, id int64) (*accounting.Journal, error) {
	return &accounting.Journal{ID: id, Name: "Bank", Code: "BNK", Type: accounting.JournalTypeBank}, nil
}
func (m *mockAccountingSvc) GetAccount(ctx context.Context, id int64) (*accounting.Account, error) {
	return &accounting.Account{ID: id, Code: "101000"}, nil
}
func (m *mockAccountingSvc) GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error) {
	return &accounting.Account{ID: 1, Code: code}, nil
}
func (m *mockAccountingSvc) UpdatePaymentStatus(ctx context.Context, id int64, state accounting.PaymentState, residual float64) error {
	return nil
}
func (m *mockAccountingSvc) GetMoveLine(ctx context.Context, id int64) (*accounting.AccountMoveLine, error) {
	return &accounting.AccountMoveLine{ID: id}, nil
}
func (m *mockAccountingSvc) UpdateMoveLineReconcile(ctx context.Context, id int64, reconciled bool, residual float64, matchingNumber *string) error {
	return nil
}
func (m *mockAccountingSvc) ListReconcilableMoveLines(ctx context.Context, partnerID *int64, excludeLineIDs []int64, limit int) ([]accounting.AccountMoveLine, error) {
	return []accounting.AccountMoveLine{}, nil
}

type mockSequenceProvider struct{}

func (m *mockSequenceProvider) NextStatementName(ctx context.Context, journalCode string, year int) (string, error) {
	return "ST/2026/001", nil
}

func setupTestServer() (*chi.Mux, *bankstatementusecase.UseCase) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := bankstatementstorage.NewMemoryRepo()
	accSvc := &mockAccountingSvc{}
	seq := &mockSequenceProvider{}
	uc := bankstatementusecase.New(repo, accSvc, seq, logger)
	h := bankstatementhttp.NewHandler(uc, logger)

	r := chi.NewRouter()
	r.Post("/bank-statements", h.CreateStatement)
	r.Get("/bank-statements/{id}", h.GetStatement)
	return r, uc
}

func TestBankStatementHandler_CRUD(t *testing.T) {
	r, _ := setupTestServer()

	// 1. Create Statement
	reqBody := bankstatementhttp.CreateStatementRequest{
		JournalID: 3,
		Date:      func() *time.Time { value := time.Now(); return &value }(),
		Currency:  "USD",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/bank-statements", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data bankstatementhttp.StatementResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	stID := resp.Data.ID

	// 2. Get Statement
	req = httptest.NewRequest(http.MethodGet, "/bank-statements/"+strconvFormat(stID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func strconvFormat(id int64) string {
	var buf [20]byte
	i := len(buf)
	if id == 0 {
		return "0"
	}
	for id >= 10 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	i--
	buf[i] = byte('0' + id)
	return string(buf[i:])
}
