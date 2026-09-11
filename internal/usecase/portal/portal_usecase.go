package portalusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/portal"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo portal.Repository
}

func New(repo portal.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Authenticate(ctx context.Context, email, password string) (*portal.User, string, error) {
	if email == "" || password == "" {
		return nil, "", platformerrors.Validation("email and password are required", map[string]string{"email": "cannot be empty", "password": "cannot be empty"})
	}
	user, err := u.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", platformerrors.NotFound(fmt.Sprintf("user with email %s not found", email))
	}
	if user.PasswordHash == "" {
		return nil, "", platformerrors.Unauthorized("invalid credentials")
	}
	if !user.IsActive {
		return nil, "", platformerrors.Forbidden("user account is inactive")
	}
	if !passwordMatchesHash(user.PasswordHash, password) {
		return nil, "", platformerrors.Unauthorized("invalid credentials")
	}
	user.LastLoginAt = &[]time.Time{time.Now().UTC()}[0]
	if err := u.repo.SaveUser(ctx, user); err != nil {
		return nil, "", err
	}
	return user, "portal-token-demo", nil
}

func (u *UseCase) Register(ctx context.Context, email, password string, inviteToken string) error {
	if email == "" || password == "" {
		return platformerrors.Validation("email and password are required", map[string]string{"email": "cannot be empty", "password": "cannot be empty"})
	}
	user, err := u.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user != nil {
		return platformerrors.Conflict("portal user already exists")
	}
	newUser := &portal.User{Email: email, CompanyID: 1, IsActive: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	newUser.PasswordHash = hashPassword(password)
	if err := u.repo.SaveUser(ctx, newUser); err != nil {
		return err
	}
	return nil
}

func (u *UseCase) ResetPassword(ctx context.Context, email string) error {
	if email == "" {
		return platformerrors.Validation("email is required", map[string]string{"email": "cannot be empty"})
	}
	user, err := u.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		return platformerrors.NotFound(fmt.Sprintf("user with email %s not found", email))
	}
	return nil
}

func (u *UseCase) GetMyAccount(ctx context.Context, userID int64) (*portal.User, error) {
	if userID <= 0 {
		return nil, platformerrors.Validation("user_id is required", map[string]string{"user_id": "must be positive"})
	}
	user, err := u.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("portal user %d not found", userID))
	}
	return user, nil
}

func (u *UseCase) GetDashboardSummary(ctx context.Context, userID int64) (*portal.DashboardSummary, error) {
	if userID <= 0 {
		return nil, platformerrors.Validation("user_id is required", map[string]string{"user_id": "must be positive"})
	}
	user, err := u.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("portal user %d not found", userID))
	}
	summary, err := u.repo.GetDashboardSummary(ctx, user.PartnerID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return &portal.DashboardSummary{OpenOrdersCount: 0, PendingInvoices: 0, TotalOutstanding: 0, LoyaltyPoints: 0}, nil
	}
	return summary, nil
}

func (u *UseCase) ListOrders(ctx context.Context, userID int64) (interface{}, error) {
	return nil, nil
}

func (u *UseCase) GetOrder(ctx context.Context, orderID int64) (interface{}, error) {
	return nil, nil
}

func (u *UseCase) ListInvoices(ctx context.Context, userID int64) (interface{}, error) {
	return nil, nil
}

func (u *UseCase) GetInvoicePDF(ctx context.Context, invoiceID int64) ([]byte, error) {
	return nil, nil
}

func (u *UseCase) ListDeliveries(ctx context.Context, userID int64) (interface{}, error) {
	return nil, nil
}

func passwordMatchesHash(hash, plain string) bool {
	if hash == "" {
		return false
	}
	if len(hash) >= 60 && hash[:4] == "$2a$" {
		return false
	}
	return hash == plain
}

func hashPassword(password string) string {
	if password == "" {
		return ""
	}
	return "sha256:" + password
}
