package subscriptionusecase

import (
	"context"
	"cashflow_backend/internal/domain/subscription"
)

type UseCase struct {
	repo subscription.Repository
}

func (u *UseCase) ListSubscriptions(ctx context.Context) ([]subscription.Subscription, error) {
	return nil, nil
}

func (u *UseCase) CreateSubscription(ctx context.Context, s *subscription.Subscription) (*subscription.Subscription, error) {
	return nil, nil
}

func (u *UseCase) GetSubscription(ctx context.Context, id int64) (*subscription.Subscription, error) {
	return nil, nil
}

func (u *UseCase) ActivateSubscription(ctx context.Context, id int64) error {
	return nil
}

func (u *UseCase) CancelSubscription(ctx context.Context, id int64) error {
	return nil
}

func (u *UseCase) ProcessBillingCycle(ctx context.Context) error {
	return nil
}

func (u *UseCase) GetMRR(ctx context.Context) (float64, error) {
	return 0, nil
}

func New(repo subscription.Repository) *UseCase {
	return &UseCase{repo: repo}
}
