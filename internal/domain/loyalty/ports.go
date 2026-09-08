package loyalty

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the persistence contract for the Loyalty & Rewards subsystem.
type Repository interface {
	// Programs
	CreateProgram(ctx context.Context, p *LoyaltyProgram) error
	UpdateProgram(ctx context.Context, p *LoyaltyProgram) error
	GetProgramByID(ctx context.Context, id int64) (*LoyaltyProgram, error)
	ListPrograms(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[LoyaltyProgram], error)
	DeleteProgram(ctx context.Context, id int64) error

	// Rules
	CreateRule(ctx context.Context, r *LoyaltyRule) error
	UpdateRule(ctx context.Context, r *LoyaltyRule) error
	DeleteRule(ctx context.Context, id int64) error

	// Rewards
	CreateReward(ctx context.Context, r *LoyaltyReward) error
	UpdateReward(ctx context.Context, r *LoyaltyReward) error
	DeleteReward(ctx context.Context, id int64) error
	GetRewardByID(ctx context.Context, id int64) (*LoyaltyReward, error)

	// Mails (config only)
	CreateMail(ctx context.Context, m *LoyaltyMail) error
	UpdateMail(ctx context.Context, m *LoyaltyMail) error
	DeleteMail(ctx context.Context, id int64) error

	// Cards
	CreateCard(ctx context.Context, c *LoyaltyCard) error
	UpdateCard(ctx context.Context, c *LoyaltyCard) error
	GetCardByID(ctx context.Context, id int64) (*LoyaltyCard, error)
	GetCardByCode(ctx context.Context, code string) (*LoyaltyCard, error)
	ListCards(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[LoyaltyCard], error)

	// History
	AddHistory(ctx context.Context, h *LoyaltyHistory) error
	ListHistoryByCard(ctx context.Context, cardID int64) ([]LoyaltyHistory, error)

	// Sale Order Coupon Points
	UpsertCouponPoints(ctx context.Context, orderID, couponID int64, points float64) error
	RemoveCouponPointsByOrder(ctx context.Context, orderID int64) error
	ListCouponPointsByOrder(ctx context.Context, orderID int64) ([]OrderCouponPoints, error)
}