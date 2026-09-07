package analytic

import (
	"context"
	"time"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// RelevantPlan is a root analytic plan relevant to a specific company + business
// domain context, together with the resolved applicability (get_relevant_plans).
type RelevantPlan struct {
	ID            int64         `json:"id"`
	Name          string        `json:"name"`
	Applicability Applicability `json:"applicability"`
	ColumnName    string        `json:"column_name"`
}

// Repository defines the persistent storage contract for the Analytic Accounting domain.
type Repository interface {
	// ─── Plans ───────────────────────────────────────────────────────────
	CreatePlan(ctx context.Context, p *AnalyticPlan) error
	GetPlanByID(ctx context.Context, id int64) (*AnalyticPlan, error)
	UpdatePlan(ctx context.Context, p *AnalyticPlan) error
	DeletePlan(ctx context.Context, id int64) error
	ListPlans(ctx context.Context, includeInactive bool) ([]AnalyticPlan, error)
	GetChildrenPlans(ctx context.Context, parentID int64) ([]AnalyticPlan, error)

	// ─── Applicabilities (G2) ────────────────────────────────────────────
	SetApplicability(ctx context.Context, a *AnalyticApplicability) error
	GetApplicabilities(ctx context.Context, planID int64) ([]AnalyticApplicability, error)
	// GetRelevantPlans returns the root plans relevant to a context (get_relevant_plans).
	GetRelevantPlans(ctx context.Context, companyID int64, businessDomain BusinessDomain) ([]RelevantPlan, error)

	// ─── Accounts ────────────────────────────────────────────────────────
	CreateAccount(ctx context.Context, a *AnalyticAccount) error
	GetAccountByID(ctx context.Context, id int64) (*AnalyticAccount, error)
	UpdateAccount(ctx context.Context, a *AnalyticAccount) error
	DeleteAccount(ctx context.Context, id int64) error
	ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[AnalyticAccount], error)
	GetAccountTotals(ctx context.Context, accountID int64, fromDate, toDate *time.Time) (DebitCreditBalance, error)

	// ─── Lines ───────────────────────────────────────────────────────────
	CreateLine(ctx context.Context, l *AnalyticLine) error
	CreateLines(ctx context.Context, lines []AnalyticLine) error
	GetLineByID(ctx context.Context, id int64) (*AnalyticLine, error)
	UpdateLine(ctx context.Context, l *AnalyticLine) error
	ListLines(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[AnalyticLine], error)
	ListLinesByMoveLine(ctx context.Context, moveLineID int64) ([]AnalyticLine, error)

	// ─── Distribution Models (G7) ────────────────────────────────────────
	CreateDistributionModel(ctx context.Context, m *DistributionModel) error
	GetDistributionModelByID(ctx context.Context, id int64) (*DistributionModel, error)
	UpdateDistributionModel(ctx context.Context, m *DistributionModel) error
	DeleteDistributionModel(ctx context.Context, id int64) error
	ListDistributionModels(ctx context.Context) ([]DistributionModel, error)
	// MatchDistribution returns the best matching rule for a context (get_distribution).
	MatchDistribution(ctx context.Context, partnerID, partnerCategoryID *int64, companyID int64) (AnalyticDistribution, error)

	// ─── Project Plan (G8) ───────────────────────────────────────────────
	GetProjectPlanID(ctx context.Context) (int64, error)
	SetProjectPlanID(ctx context.Context, planID int64) error
}