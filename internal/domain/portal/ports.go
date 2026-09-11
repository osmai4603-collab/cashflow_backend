package portal

import (
	"context"
)

type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	SaveUser(ctx context.Context, user *User) error

	GetDashboardSummary(ctx context.Context, partnerID int64) (*DashboardSummary, error)
}

type Service interface {
	Authenticate(ctx context.Context, email, password string) (*User, string, error) // user, token, err
	Register(ctx context.Context, email, password string, inviteToken string) error
	ResetPassword(ctx context.Context, email string) error
	GetMyAccount(ctx context.Context, userID int64) (*User, error)
}
