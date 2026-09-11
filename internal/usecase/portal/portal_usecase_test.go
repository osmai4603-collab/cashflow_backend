package portalusecase

import (
	"context"
	"testing"
	"time"

	portalstorage "cashflow_backend/internal/adapters/storage/portal"
	"cashflow_backend/internal/domain/portal"
)

func TestPortalUseCase_AuthenticationAndDashboard(t *testing.T) {
	repo := portalstorage.NewMemoryRepo()
	ctx := context.Background()

	user := &portal.User{ID: 1, PartnerID: 12, Email: "customer@example.com", CompanyID: 1, IsActive: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	user.PasswordHash = ""
	if err := repo.SaveUser(ctx, user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	uc := New(repo)
	created, token, err := uc.Authenticate(ctx, "customer@example.com", "secret")
	if err == nil {
		t.Fatalf("expected auth error for empty password hash; got user=%v token=%q", created, token)
	}

	user.PasswordHash = "$2a$10$Itm9f6Y8m2sI3x9f6Y8m2uXy7T0Tvd7a3E8j0QpE9xJ4JrD0ZQ7K" // invalid bcrypt hash to validate guard path
	_ = repo.SaveUser(ctx, user)
	if _, _, err = uc.Authenticate(ctx, "customer@example.com", "secret"); err == nil {
		t.Fatal("expected invalid bcrypt hash to fail auth")
	}

	if _, err := uc.GetDashboardSummary(ctx, 12); err == nil {
		t.Fatal("expected dashboard summary to require stored summary state")
	}
}
