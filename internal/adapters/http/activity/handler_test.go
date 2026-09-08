package activityhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	activityhttp "cashflow_backend/internal/adapters/http/activity"
	activitystorage "cashflow_backend/internal/adapters/storage/activity"
	userstorage "cashflow_backend/internal/adapters/storage/user"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/platform/auth"
	activityusecase "cashflow_backend/internal/usecase/activity"

	"github.com/go-chi/chi/v5"
)

type mockBus struct{}

func (b *mockBus) NotifyUser(ctx context.Context, userID int64, channel string, payload any) error {
	return nil
}

func setupTestServer() (*chi.Mux, *activitystorage.MemoryRepo) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := activitystorage.NewMemoryRepo()
	userRepo := userstorage.NewMemoryRepo()
	bus := &mockBus{}

	uc := activityusecase.NewUseCase(repo, repo, repo, repo, repo, bus, userRepo)
	h := activityhttp.NewHandler(uc, logger)

	r := chi.NewRouter()
	activityhttp.RegisterRoutes(r, h)
	return r, repo
}

func withAuth(req *http.Request, userID, companyID int64) *http.Request {
	claims := &auth.UserClaims{
		UserID:    userID,
		CompanyID: companyID,
	}
	ctx := auth.WithClaims(req.Context(), claims)
	return req.WithContext(ctx)
}

func TestActivityHandler_EndToEnd(t *testing.T) {
	r, repo := setupTestServer()
	ctx := context.Background()

	// 1. Create Activity Type
	at := &activity.ActivityType{
		Name:    "Call",
		Summary: "Follow up call",
		Active:  true,
	}
	repo.CreateType(ctx, at)
	atID := at.ID

	// 2. Create Activity via HTTP
	createReq := activityhttp.CreateActivityRequest{
		ActivityTypeID: atID,
		Summary:        "Initial Sales Call",
		DateDeadline:   time.Now().Add(24 * time.Hour).UTC(),
		AssignedUserID: 100,
		ResModel:       "res.partner",
		ResID:          func() *int64 { id := int64(1); return &id }(),
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/activities", bytes.NewReader(body))
	req = withAuth(req, 1, 1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Data activityhttp.ActivityDTO `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&createResp)
	actID := createResp.Data.ID

	// 3. Get Activity
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/activities/%d", actID), nil)
	req = withAuth(req, 1, 1)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// 4. Complete Activity
	completeReq := activityhttp.CompleteActivityRequest{
		Feedback: "Customer was busy, scheduled for next week.",
	}
	body, _ = json.Marshal(completeReq)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/activities/%d/done", actID), bytes.NewReader(body))
	req = withAuth(req, 1, 1)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on complete, got %d", w.Code)
	}

	var completeResp struct {
		Data activityhttp.ActivityDTO `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&completeResp)
	if completeResp.Data.Active {
		t.Error("expected activity to be inactive after completion")
	}
	if completeResp.Data.Feedback != "Customer was busy, scheduled for next week." {
		t.Errorf("unexpected feedback: %s", completeResp.Data.Feedback)
	}

	// 5. List My Activities (should be empty if active=true is the default filter)
	req = httptest.NewRequest(http.MethodGet, "/activities/my", nil)
	req = withAuth(req, 100, 1) // Using assigned user
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on list, got %d", w.Code)
	}
}
