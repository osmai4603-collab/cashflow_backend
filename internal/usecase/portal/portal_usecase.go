package portalusecase

import (
	"context"
	"cashflow_backend/internal/domain/portal"
)

type UseCase struct {
	repo portal.Repository
}

func New(repo portal.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Authenticate(ctx context.Context, email, password string) (*portal.User, string, error) {
	return nil, "", nil
}

func (u *UseCase) Register(ctx context.Context, email, password string, inviteToken string) error {
	return nil
}

func (u *UseCase) ResetPassword(ctx context.Context, email string) error {
	return nil
}

func (u *UseCase) GetMyAccount(ctx context.Context, userID int64) (*portal.User, error) {
	return nil, nil
}

func (u *UseCase) GetDashboardSummary(ctx context.Context, userID int64) (*portal.DashboardSummary, error) {
	return nil, nil
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
