package userusecase_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	userstorage "cashflow_backend/internal/adapters/storage/user"
	"cashflow_backend/internal/platform/pagination"
	userusecase "cashflow_backend/internal/usecase/user"
)

func setupTestUseCase() (*userusecase.UserUseCase, *userstorage.MemoryRepo) {
	userRepo := userstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := userusecase.New(userRepo, partnerRepo, logger, "", 0)
	return uc, userRepo
}

func TestUserUseCase_CreateUser(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	// 1. Success with auto-created partner
	u, err := uc.CreateUser(ctx, userusecase.CreateUserInput{
		Login:       "Ahmed",
		Email:       "ahmed@example.com",
		Name:        "Ahmed Al-Otaibi",
		Password:    "SecurePass123",
		PartnerName: "Ahmed Partner",
		CompanyID:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error creating user: %v", err)
	}
	if u.ID <= 0 {
		t.Errorf("expected positive user ID, got %d", u.ID)
	}
	if u.Login != "ahmed" {
		t.Errorf("expected normalized lowercase login 'ahmed', got %q", u.Login)
	}
	if u.PartnerID <= 0 {
		t.Errorf("expected auto-created partner, got partner_id=%d", u.PartnerID)
	}
	if !u.CheckPassword("SecurePass123") {
		t.Errorf("expected password hash to match plaintext")
	}
	if u.CheckPassword("WrongPass") {
		t.Errorf("expected wrong password to fail CheckPassword")
	}

	// 2. Validation: missing company
	_, err = uc.CreateUser(ctx, userusecase.CreateUserInput{
		Login:       "bob",
		Name:        "Bob",
		Password:    "x",
		PartnerName: "Bob Partner",
	})
	if err == nil {
		t.Fatalf("expected error for missing company, got nil")
	}

	// 3. Validation: missing password
	_, err = uc.CreateUser(ctx, userusecase.CreateUserInput{
		Login:       "bob",
		Name:        "Bob",
		PartnerName: "Bob Partner",
		CompanyID:   1,
	})
	if err == nil {
		t.Fatalf("expected error for missing password, got nil")
	}

	// 4. Validation: neither partner_id nor partner_name
	_, err = uc.CreateUser(ctx, userusecase.CreateUserInput{
		Login:     "bob",
		Name:      "Bob",
		Password:  "x",
		CompanyID: 1,
	})
	if err == nil {
		t.Fatalf("expected error for missing partner, got nil")
	}
}

func TestUserUseCase_Login(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	u, err := uc.CreateUser(ctx, userusecase.CreateUserInput{
		Login:       "admin",
		Email:       "admin@example.com",
		Name:        "Admin",
		Password:    "admin123",
		PartnerName: "Administrator",
		CompanyID:   1,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 1. Successful login
	res, err := uc.Login(ctx, "admin", "admin123")
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if res.Token == "" {
		t.Errorf("expected non-empty JWT token")
	}
	if res.User == nil || res.User.ID != u.ID {
		t.Errorf("expected login result user to match created user")
	}

	// 2. Wrong password
	_, err = uc.Login(ctx, "admin", "wrong-password")
	if err == nil {
		t.Fatalf("expected error for wrong password, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("expected unauthorized message, got %v", err)
	}

	// 3. Nonexistent user
	_, err = uc.Login(ctx, "ghost", "admin123")
	if err == nil {
		t.Fatalf("expected error for nonexistent user, got nil")
	}
}

func TestUserUseCase_UpdateAndDelete(t *testing.T) {
	uc, repo := setupTestUseCase()
	ctx := context.Background()

	u, err := uc.CreateUser(ctx, userusecase.CreateUserInput{
		Login:       "update_me",
		Name:        "Original",
		Password:    "oldpass",
		PartnerName: "Partner",
		CompanyID:   1,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Update
	newEmail := "new@example.com"
	updated, err := uc.UpdateUser(ctx, u.ID, userusecase.UpdateUserInput{
		Email: &newEmail,
		Name:  stringPtr("Renamed"),
	})
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}
	if updated.Name != "Renamed" || updated.Email != newEmail {
		t.Errorf("update fields mismatch")
	}

	// Verify stored update
	fetched, err := uc.GetUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if fetched.Name != "Renamed" {
		t.Errorf("expected persisted updated name, got %q", fetched.Name)
	}

	// Invalid id
	_, err = uc.GetUser(ctx, 0)
	if err == nil {
		t.Fatalf("expected error for id=0, got nil")
	}

	// Delete (soft) — user should no longer be found, login fails
	if err := uc.DeleteUser(ctx, u.ID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}
	_, err = uc.GetUser(ctx, u.ID)
	if err == nil {
		t.Fatalf("expected not found after soft delete, got nil")
	}
	if _, err := repo.GetByLogin(ctx, "update_me"); err == nil {
		t.Fatalf("expected GetByLogin to fail after delete, got nil")
	}
}

func TestUserUseCase_ListUsers(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	for _, login := range []string{"alice", "bob", "carol"} {
		_, err := uc.CreateUser(ctx, userusecase.CreateUserInput{
			Login:       login,
			Name:        login,
			Password:    "pass123",
			PartnerName: login + " Partner",
			CompanyID:   1,
		})
		if err != nil {
			t.Fatalf("failed to create user %q: %v", login, err)
		}
	}

	res, err := uc.ListUsers(ctx, nil, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing users: %v", err)
	}
	if res.TotalItems != 3 {
		t.Errorf("expected 3 users, got %d", res.TotalItems)
	}
}

func stringPtr(s string) *string {
	return &s
}
