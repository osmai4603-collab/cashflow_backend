package userhttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	userhttp "cashflow_backend/internal/adapters/http/user"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	userstorage "cashflow_backend/internal/adapters/storage/user"
	userusecase "cashflow_backend/internal/usecase/user"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*chi.Mux, *userstorage.MemoryRepo) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := userstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	// In a real environment, we'd mock the token generator
	uc := userusecase.New(repo, partnerRepo, logger, "secret", 0)
	h := userhttp.NewHandler(uc, logger)

	r := chi.NewRouter()
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/", h.List)
		r.Post("/login", h.Login)
	})

	return r, repo
}

func TestUserHandler_CRUD(t *testing.T) {
	r, _ := setupTestServer()

	// 1. Create User
	createReq := userhttp.CreateUserRequest{
		Name:      "Test User",
		Login:     "testuser",
		Password:  "securepassword",
		Email:     "test@example.com",
		CompanyID: 1,
		PartnerName: "Test Partner",
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Data userhttp.UserResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&createResp)
	userID := createResp.Data.ID

	// 2. Get User
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+strconvFormat(userID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 3. Update User
	updateReq := userhttp.UpdateUserRequest{
		Name: func() *string { value := "Updated User"; return &value }(),
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/"+strconvFormat(userID), bytes.NewReader(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 4. List Users
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?name=Updated", nil)
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
