package userstorage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	userstorage "cashflow_backend/internal/adapters/storage/user"
	"cashflow_backend/internal/domain/user"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestUserMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := userstorage.NewMemoryRepo()

	now := time.Now().UTC()
	u := &user.User{
		Login:        "Admin",
		Email:        "admin@example.com",
		Name:         "Admin",
		PasswordHash: "hash",
		PartnerID:    1,
		CompanyID:    1,
		LastLoginAt:  &now,
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("unexpected error creating user: %v", err)
	}
	if u.ID <= 0 {
		t.Errorf("expected positive ID, got %d", u.ID)
	}

	fetched, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching user: %v", err)
	}
	if fetched.Login != "Admin" || fetched.LastLoginAt == nil {
		t.Errorf("fetch mismatch: login=%q last_login=%v", fetched.Login, fetched.LastLoginAt)
	}

	fetched.Name = "Super Admin"
	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("unexpected error updating user: %v", err)
	}

	// Delete
	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("unexpected error deleting user: %v", err)
	}
	_, err = repo.GetByID(ctx, u.ID)
	if err == nil {
		t.Errorf("expected error getting soft-deleted user, got nil")
	}
}

func TestUserMemoryRepo_GetByLoginAndLastLogin(t *testing.T) {
	ctx := context.Background()
	repo := userstorage.NewMemoryRepo()

	u := &user.User{Login: "sales.user", Name: "Sales", PartnerID: 1, CompanyID: 1}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	fetched, err := repo.GetByLogin(ctx, "SALES.USER")
	if err != nil {
		t.Fatalf("unexpected error getting user by login: %v", err)
	}
	if fetched.Login != "sales.user" {
		t.Errorf("expected sales.user, got %q", fetched.Login)
	}

	// LastLogin is nil initially
	if fetched.LastLoginAt != nil {
		t.Errorf("expected nil LastLoginAt initially")
	}

	if err := repo.UpdateLastLogin(ctx, u.ID); err != nil {
		t.Fatalf("unexpected error updating last login: %v", err)
	}
	fetched2, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("unexpected error re-fetching user: %v", err)
	}
	if fetched2.LastLoginAt == nil {
		t.Errorf("expected LastLoginAt to be set after UpdateLastLogin")
	}

	// Unknown login
	_, err = repo.GetByLogin(ctx, "ghost")
	if err == nil {
		t.Errorf("expected error for unknown login, got nil")
	}
}

func TestUserMemoryRepo_ListAndFilter(t *testing.T) {
	ctx := context.Background()
	repo := userstorage.NewMemoryRepo()

	for _, login := range []string{"alice", "bob", "carol"} {
		if err := repo.Create(ctx, &user.User{Login: login, Name: login, PartnerID: 1, CompanyID: 1}); err != nil {
			t.Fatalf("failed to seed user: %v", err)
		}
	}

	page := pagination.PageRequest{Page: 1, Limit: 10}
	res, err := repo.List(ctx, nil, page)
	if err != nil {
		t.Fatalf("unexpected error listing users: %v", err)
	}
	if res.TotalItems != 3 {
		t.Errorf("expected 3 users, got %d", res.TotalItems)
	}

	loginFilter := filter.NewFilter(filter.Criterion{Field: "login", Operator: filter.OpILike, Value: "ali"})
	filtered, err := repo.List(ctx, loginFilter, page)
	if err != nil {
		t.Fatalf("unexpected error filtering users: %v", err)
	}
	if filtered.TotalItems != 1 {
		t.Errorf("expected 1 user matching 'ali', got %d", filtered.TotalItems)
	}
}

func TestUserMemoryRepo_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := userstorage.NewMemoryRepo()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_ = repo.Create(ctx, &user.User{Login: fmt.Sprintf("user%d", i), Name: "u", PartnerID: 1, CompanyID: 1})
		}(i)
	}
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 25})
		}()
	}
	wg.Wait()

	res, err := repo.List(ctx, nil, pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error after concurrent access: %v", err)
	}
	if res.TotalItems != n {
		t.Errorf("expected %d users, got %d", n, res.TotalItems)
	}
}
