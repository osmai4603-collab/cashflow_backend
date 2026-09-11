package repairusecase

import (
	"context"
	"cashflow_backend/internal/domain/repair"
)

type UseCase struct {
	repo repair.Repository
}

func (u *UseCase) ListOrders(ctx context.Context) ([]repair.Order, error) {
	return nil, nil
}

func (u *UseCase) CreateOrder(ctx context.Context, o *repair.Order) (*repair.Order, error) {
	return nil, nil
}

func (u *UseCase) ConfirmOrder(ctx context.Context, id int64) error {
	return nil
}

func (u *UseCase) CompleteOrder(ctx context.Context, id int64) error {
	return nil
}

func (u *UseCase) CreateInvoice(ctx context.Context, id int64) (int64, error) {
	return 0, nil
}

func New(repo repair.Repository) *UseCase {
	return &UseCase{repo: repo}
}
