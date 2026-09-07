package analyticusecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/analytic"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreatePlanInput struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	ParentID             *int64 `json:"parent_id"`
	Sequence             int    `json:"sequence"`
	Color                int    `json:"color"`
	DefaultApplicability string `json:"default_applicability"` // optional|mandatory|unavailable
}

type UpdatePlanInput struct {
	Name                 *string `json:"name"`
	Description          *string `json:"description"`
	ParentID             *int64  `json:"parent_id"`
	Sequence             *int    `json:"sequence"`
	Color                *int    `json:"color"`
	DefaultApplicability *string `json:"default_applicability"`
	Active               *bool   `json:"active"`
}

type SetApplicabilityInput struct {
	PlanID         int64  `json:"plan_id"`
	BusinessDomain string `json:"business_domain"`
	Applicability  string `json:"applicability"`
	CompanyID      *int64 `json:"company_id"`
	Sequence       int    `json:"sequence"`
}

type CreateAccountInput struct {
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	PlanID    int64   `json:"plan_id"`
	PartnerID *int64  `json:"partner_id"`
	Color     int     `json:"color"`
	CompanyID *int64  `json:"company_id"`
	Currency  string  `json:"currency"`
}

type UpdateAccountInput struct {
	Name      *string `json:"name"`
	Code      *string `json:"code"`
	PlanID    *int64  `json:"plan_id"`
	PartnerID *int64  `json:"partner_id"`
	Color     *int    `json:"color"`
	CompanyID *int64  `json:"company_id"`
	Active    *bool   `json:"active"`
}

type CreateLineInput struct {
	Name             string                       `json:"name"`
	Date             time.Time                    `json:"date"`
	Amount           float64                      `json:"amount"`
	UnitAmount       float64                      `json:"unit_amount"`
	ProductUoMID     *int64                       `json:"product_uom_id"`
	PartnerID        *int64                       `json:"partner_id"`
	UserID           int64                        `json:"user_id"`
	CompanyID        int64                        `json:"company_id"`
	Currency         string                       `json:"currency"`
	Category         string                       `json:"category"`
	AccountID        int64                        `json:"account_id"`
	MoveLineID       *int64                       `json:"move_line_id"`
	GeneralAccountID *int64                       `json:"general_account_id"`
	Source           string                       `json:"source"`
	Distribution     analytic.AnalyticDistribution `json:"distribution,omitempty"`
}

type UpdateLineInput struct {
	Name             *string                      `json:"name"`
	Date             *time.Time                   `json:"date"`
	Amount           *float64                     `json:"amount"`
	UnitAmount       *float64                     `json:"unit_amount"`
	Category         *string                      `json:"category"`
	AccountID        *int64                       `json:"account_id"`
	Distribution     analytic.AnalyticDistribution `json:"distribution,omitempty"`
}

type CreateDistributionModelInput struct {
	Sequence          int                         `json:"sequence"`
	PartnerID         *int64                      `json:"partner_id"`
	PartnerCategoryID *int64                      `json:"partner_category_id"`
	CompanyID         *int64                      `json:"company_id"`
	Distribution      analytic.AnalyticDistribution `json:"distribution"`
}

type UpdateDistributionModelInput struct {
	Sequence          *int                         `json:"sequence"`
	PartnerID         *int64                       `json:"partner_id"`
	PartnerCategoryID *int64                       `json:"partner_category_id"`
	CompanyID         *int64                       `json:"company_id"`
	Distribution      analytic.AnalyticDistribution `json:"distribution"`
	Active            *bool                        `json:"active"`
}

// MoveLineAnalyticInput carries the data needed to derive analytic lines from a
// posted accounting move line (Phase 3 integration - CreateLinesFromMoveLine).
type MoveLineAnalyticInput struct {
	MoveLineID       int64                         `json:"move_line_id"`
	GeneralAccountID int64                         `json:"general_account_id"`
	Name             string                        `json:"name"`
	Date             time.Time                     `json:"date"`
	PartnerID        *int64                        `json:"partner_id"`
	UserID           int64                         `json:"user_id"`
	CompanyID        int64                         `json:"company_id"`
	Currency         string                        `json:"currency"`
	Amount           float64                       `json:"amount"` // debit - credit (signed)
	UnitAmount       float64                       `json:"unit_amount"`
	Source           string                        `json:"source"`
	Distribution     analytic.AnalyticDistribution `json:"distribution"`
}

// PlanStructure is the plan plus every account belonging to its plan tree (G4).
type PlanStructure struct {
	Plan     analytic.AnalyticPlan     `json:"plan"`
	Accounts []analytic.AnalyticAccount `json:"accounts"`
}

// SplitResult is the outcome of splitting a line across a distribution.
type SplitResult struct {
	Lines []analytic.AnalyticLine `json:"lines"`
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Definition
// ─────────────────────────────────────────────────────────────────────────────

// UseCase orchestrates business rules for the Analytic Accounting domain.
type UseCase struct {
	repo   analytic.Repository
	logger *slog.Logger
}

// New creates an initialized analytic UseCase instance.
func New(repo analytic.Repository, logger *slog.Logger) *UseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &UseCase{
		repo:   repo,
		logger: logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Plans
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreatePlan(ctx context.Context, in CreatePlanInput) (*analytic.AnalyticPlan, error) {
	plan := &analytic.AnalyticPlan{
		Name:                strings.TrimSpace(in.Name),
		Description:         strings.TrimSpace(in.Description),
		ParentID:            in.ParentID,
		Sequence:            in.Sequence,
		Color:               in.Color,
		DefaultApplicability: analytic.Applicability(in.DefaultApplicability),
		Active:              true,
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreatePlan(ctx, plan); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "analytic plan created", "id", plan.ID, "name", plan.Name)
	return plan, nil
}

func (uc *UseCase) GetPlan(ctx context.Context, id int64) (*analytic.AnalyticPlan, error) {
	return uc.repo.GetPlanByID(ctx, id)
}

func (uc *UseCase) UpdatePlan(ctx context.Context, id int64, in UpdatePlanInput) (*analytic.AnalyticPlan, error) {
	plan, err := uc.repo.GetPlanByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		plan.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		plan.Description = strings.TrimSpace(*in.Description)
	}
	if in.ParentID != nil {
		plan.ParentID = in.ParentID
	}
	if in.Sequence != nil {
		plan.Sequence = *in.Sequence
	}
	if in.Color != nil {
		plan.Color = *in.Color
	}
	if in.DefaultApplicability != nil {
		plan.DefaultApplicability = analytic.Applicability(*in.DefaultApplicability)
	}
	if in.Active != nil {
		plan.Active = *in.Active
	}

	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdatePlan(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (uc *UseCase) DeletePlan(ctx context.Context, id int64) error {
	if err := uc.repo.DeletePlan(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "analytic plan deleted", "id", id)
	return nil
}

func (uc *UseCase) ListPlans(ctx context.Context, includeInactive bool) ([]analytic.AnalyticPlan, error) {
	return uc.repo.ListPlans(ctx, includeInactive)
}

func (uc *UseCase) GetChildrenPlans(ctx context.Context, parentID int64) ([]analytic.AnalyticPlan, error) {
	return uc.repo.GetChildrenPlans(ctx, parentID)
}

// GetPlanStructure returns a plan together with all accounts in its tree (root_plan_id).
func (uc *UseCase) GetPlanStructure(ctx context.Context, planID int64) (*PlanStructure, error) {
	plan, err := uc.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	page, err := uc.repo.ListAccounts(ctx, nil, pagination.PageRequest{Limit: 10000})
	if err != nil {
		return nil, err
	}
	accounts := make([]analytic.AnalyticAccount, 0, len(page.Items))
	for _, a := range page.Items {
		if a.RootPlanID == planID {
			accounts = append(accounts, a)
		}
	}
	return &PlanStructure{Plan: *plan, Accounts: accounts}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Applicabilities (G2)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) SetApplicability(ctx context.Context, in SetApplicabilityInput) (*analytic.AnalyticApplicability, error) {
	rule := &analytic.AnalyticApplicability{
		PlanID:         in.PlanID,
		BusinessDomain: analytic.BusinessDomain(in.BusinessDomain),
		Applicability:  analytic.Applicability(in.Applicability),
		CompanyID:      in.CompanyID,
		Sequence:       in.Sequence,
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.SetApplicability(ctx, rule); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "analytic applicability set",
		"plan_id", in.PlanID, "domain", in.BusinessDomain, "value", in.Applicability)
	return rule, nil
}

// GetRelevantPlans returns the root plans relevant to a company + business domain.
// Plans already used by the document may be passed as forcedPlanIDs; they are
// dropped so they are not proposed again (Odoo forced_plans behavior).
func (uc *UseCase) GetRelevantPlans(ctx context.Context, companyID int64, businessDomain analytic.BusinessDomain, forcedPlanIDs []int64) ([]analytic.RelevantPlan, error) {
	relevant, err := uc.repo.GetRelevantPlans(ctx, companyID, businessDomain)
	if err != nil {
		return nil, err
	}
	if len(forcedPlanIDs) == 0 {
		return relevant, nil
	}
	forced := make(map[int64]bool, len(forcedPlanIDs))
	for _, id := range forcedPlanIDs {
		forced[id] = true
	}
	filtered := make([]analytic.RelevantPlan, 0, len(relevant))
	for _, p := range relevant {
		if forced[p.ID] {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Accounts
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateAccount(ctx context.Context, in CreateAccountInput) (*analytic.AnalyticAccount, error) {
	if err := uc.ensurePlanExists(ctx, in.PlanID); err != nil {
		return nil, err
	}
	acc := &analytic.AnalyticAccount{
		Name:      strings.TrimSpace(in.Name),
		Code:      strings.TrimSpace(in.Code),
		PlanID:    in.PlanID,
		PartnerID: in.PartnerID,
		Color:     in.Color,
		CompanyID: in.CompanyID,
		Currency:  in.Currency,
		Active:    true,
	}
	if err := acc.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateAccount(ctx, acc); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "analytic account created", "id", acc.ID, "name", acc.Name, "plan_id", acc.PlanID)
	return acc, nil
}

func (uc *UseCase) GetAccount(ctx context.Context, id int64) (*analytic.AnalyticAccount, error) {
	return uc.repo.GetAccountByID(ctx, id)
}

func (uc *UseCase) UpdateAccount(ctx context.Context, id int64, in UpdateAccountInput) (*analytic.AnalyticAccount, error) {
	acc, err := uc.repo.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		acc.Name = strings.TrimSpace(*in.Name)
	}
	if in.Code != nil {
		acc.Code = strings.TrimSpace(*in.Code)
	}
	if in.PlanID != nil {
		acc.PlanID = *in.PlanID
	}
	if in.PartnerID != nil {
		acc.PartnerID = in.PartnerID
	}
	if in.Color != nil {
		acc.Color = *in.Color
	}
	if in.CompanyID != nil {
		acc.CompanyID = in.CompanyID
	}
	if in.Active != nil {
		acc.Active = *in.Active
	}
	if err := acc.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateAccount(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (uc *UseCase) DeleteAccount(ctx context.Context, id int64) error {
	if err := uc.repo.DeleteAccount(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "analytic account deleted", "id", id)
	return nil
}

func (uc *UseCase) ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[analytic.AnalyticAccount], error) {
	return uc.repo.ListAccounts(ctx, f, page)
}

// GetAccountBalance computes debit/credit/balance for an account (optionally filtered by date).
func (uc *UseCase) GetAccountBalance(ctx context.Context, accountID int64, fromDate, toDate *time.Time) (*analytic.DebitCreditBalance, error) {
	totals, err := uc.repo.GetAccountTotals(ctx, accountID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	totals.ComputeBalance()
	return &totals, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Lines
// ─────────────────────────────────────────────────────────────────────────────

// ReconcileLineDistribution splits a base line across a distribution, producing
// one analytic line per distribution key with a proportional amount
// (equivalent to Odoo's _split_amount_fname - G3).
func (uc *UseCase) ReconcileLineDistribution(line *analytic.AnalyticLine, distribution analytic.AnalyticDistribution) ([]analytic.AnalyticLine, error) {
	if len(distribution) == 0 || line.Amount == 0 {
		return []analytic.AnalyticLine{*line}, nil
	}

	keys := sortedKeys(distribution)
	lines := make([]analytic.AnalyticLine, 0, len(keys))
	for _, key := range keys {
		pct := distribution[key]
		accountID, err := firstAccountID(key)
		if err != nil {
			return nil, platformerrors.Validation("invalid analytic distribution key", map[string]string{
				key: "no valid account references",
			})
		}

		split := *line
		split.AccountID = accountID
		split.Amount = roundAmount(line.Amount * pct / 100)
		split.Distribution = analytic.AnalyticDistribution{key: pct}
		lines = append(lines, split)
	}
	return lines, nil
}

// ValidateDistribution enforces that every mandatory plan in the given context is
// covered by a distribution totalling 100% (equivalent to Odoo's _validate_distribution).
func (uc *UseCase) ValidateDistribution(ctx context.Context, companyID int64, businessDomain analytic.BusinessDomain, lines []analytic.AnalyticLine) error {
	relevant, err := uc.repo.GetRelevantPlans(ctx, companyID, businessDomain)
	if err != nil {
		return err
	}

	var mandatory []analytic.RelevantPlan
	for _, p := range relevant {
		if p.Applicability == analytic.AppMandatory {
			mandatory = append(mandatory, p)
		}
	}
	if len(mandatory) == 0 {
		return nil
	}

	combined := analytic.AnalyticDistribution{}
	for _, l := range lines {
		for key, pct := range l.Distribution {
			combined[key] = pct
		}
	}

	accountsByPlan := make(map[int64][]int64, len(mandatory))
	for _, p := range mandatory {
		page, err := uc.repo.ListAccounts(ctx,
			filter.NewFilter(filter.Criterion{Field: "plan_id", Operator: "eq", Value: p.ID}),
			pagination.PageRequest{Limit: 10000},
		)
		if err != nil {
			return err
		}
		for _, a := range page.Items {
			accountsByPlan[p.ID] = append(accountsByPlan[p.ID], a.ID)
		}
	}

	for _, p := range mandatory {
		ids := accountsByPlan[p.ID]
		covered := 0.0
		for key, pct := range combined {
			if keyIntersectsAnyAccount(key, ids) {
				covered += pct
			}
		}
		if math.Abs(covered-100) > analytic.DistributionTolerance {
			return platformerrors.Validation(
				"analytic distribution must total 100% for mandatory plans",
				map[string]any{
					"plan_id": p.ID,
					"plan":    p.Name,
					"total":   math.Round(covered*100) / 100,
				},
			)
		}
	}
	return nil
}

// AutoCompleteDistribution pulls the best matching distribution rule and merges it
// with an existing distribution (G7 _get_distribution + _merge_distribution).
func (uc *UseCase) AutoCompleteDistribution(ctx context.Context, partnerID, partnerCategoryID *int64, companyID int64, existing analytic.AnalyticDistribution) (analytic.AnalyticDistribution, bool, error) {
	matched, err := uc.repo.MatchDistribution(ctx, partnerID, partnerCategoryID, companyID)
	if err != nil {
		return nil, false, err
	}
	result := analytic.Merge(existing, matched)
	return result.Merged, result.Changed, nil
}

// CreateLinesFromMoveLine creates the analytic lines associated with a posted
// accounting move line carrying an analytic distribution (Phase 3 integration).
func (uc *UseCase) CreateLinesFromMoveLine(ctx context.Context, in MoveLineAnalyticInput) ([]analytic.AnalyticLine, error) {
	if in.Date.IsZero() {
		in.Date = time.Now().UTC()
	}
	if in.Source == "" {
		in.Source = string(analytic.SourceManual)
	}
	if in.Currency == "" {
		in.Currency = "USD"
	}

	base := &analytic.AnalyticLine{
		Name:             in.Name,
		Date:             in.Date,
		Amount:           in.Amount,
		UnitAmount:       in.UnitAmount,
		ProductUoMID:     nil,
		PartnerID:        in.PartnerID,
		UserID:           in.UserID,
		CompanyID:        in.CompanyID,
		Currency:         in.Currency,
		Category:         "other",
		MoveLineID:       &in.MoveLineID,
		GeneralAccountID: &in.GeneralAccountID,
		Source:           analytic.LineSource(in.Source),
	}

	if err := uc.ValidateDistribution(ctx, in.CompanyID, analytic.DomainGeneral,
		[]analytic.AnalyticLine{{Distribution: in.Distribution}}); err != nil {
		return nil, err
	}

	lines, err := uc.ReconcileLineDistribution(base, in.Distribution)
	if err != nil {
		return nil, err
	}
	for i := range lines {
		if err := lines[i].Validate(); err != nil {
			return nil, err
		}
	}
	if err := uc.repo.CreateLines(ctx, lines); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "created analytic lines from move line",
		"move_line_id", in.MoveLineID, "count", len(lines))
	return lines, nil
}

// RegisterManualLine creates a single analytic line from manual input.
func (uc *UseCase) RegisterManualLine(ctx context.Context, in CreateLineInput) (*analytic.AnalyticLine, error) {
	if in.Source == "" {
		in.Source = string(analytic.SourceManual)
	}
	line := &analytic.AnalyticLine{
		Name:             strings.TrimSpace(in.Name),
		Date:             in.Date,
		Amount:           in.Amount,
		UnitAmount:       in.UnitAmount,
		ProductUoMID:     in.ProductUoMID,
		PartnerID:        in.PartnerID,
		UserID:           in.UserID,
		CompanyID:        in.CompanyID,
		Currency:         in.Currency,
		Category:         in.Category,
		AccountID:        in.AccountID,
		MoveLineID:       in.MoveLineID,
		GeneralAccountID: in.GeneralAccountID,
		Source:           analytic.LineSource(in.Source),
		Distribution:     in.Distribution,
	}
	if err := line.Validate(); err != nil {
		return nil, err
	}
	if len(in.Distribution) > 0 {
		if err := uc.ValidateDistribution(ctx, line.CompanyID, analytic.DomainGeneral, []analytic.AnalyticLine{*line}); err != nil {
			return nil, err
		}
	}
	if err := uc.repo.CreateLine(ctx, line); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "registered manual analytic line", "id", line.ID, "account_id", line.AccountID)
	return line, nil
}

func (uc *UseCase) GetLine(ctx context.Context, id int64) (*analytic.AnalyticLine, error) {
	return uc.repo.GetLineByID(ctx, id)
}

func (uc *UseCase) UpdateLine(ctx context.Context, id int64, in UpdateLineInput) (*analytic.AnalyticLine, error) {
	line, err := uc.repo.GetLineByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		line.Name = strings.TrimSpace(*in.Name)
	}
	if in.Date != nil {
		line.Date = *in.Date
	}
	if in.Amount != nil {
		line.Amount = *in.Amount
	}
	if in.UnitAmount != nil {
		line.UnitAmount = *in.UnitAmount
	}
	if in.Category != nil {
		line.Category = *in.Category
	}
	if in.AccountID != nil {
		line.AccountID = *in.AccountID
	}
	if in.Distribution != nil {
		line.Distribution = in.Distribution
	}
	if err := line.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateLine(ctx, line); err != nil {
		return nil, err
	}
	return line, nil
}

func (uc *UseCase) ListLines(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[analytic.AnalyticLine], error) {
	return uc.repo.ListLines(ctx, f, page)
}

func (uc *UseCase) ListLinesByMoveLine(ctx context.Context, moveLineID int64) ([]analytic.AnalyticLine, error) {
	return uc.repo.ListLinesByMoveLine(ctx, moveLineID)
}

// ─────────────────────────────────────────────────────────────────────────────
// Distribution Models (G7)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateDistributionModel(ctx context.Context, in CreateDistributionModelInput) (*analytic.DistributionModel, error) {
	model := &analytic.DistributionModel{
		Sequence:          in.Sequence,
		PartnerID:         in.PartnerID,
		PartnerCategoryID: in.PartnerCategoryID,
		CompanyID:         in.CompanyID,
		Distribution:      in.Distribution,
		Active:            true,
	}
	if err := model.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateDistributionModel(ctx, model); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "analytic distribution model created", "id", model.ID)
	return model, nil
}

func (uc *UseCase) GetDistributionModel(ctx context.Context, id int64) (*analytic.DistributionModel, error) {
	return uc.repo.GetDistributionModelByID(ctx, id)
}

func (uc *UseCase) UpdateDistributionModel(ctx context.Context, id int64, in UpdateDistributionModelInput) (*analytic.DistributionModel, error) {
	model, err := uc.repo.GetDistributionModelByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Sequence != nil {
		model.Sequence = *in.Sequence
	}
	if in.PartnerID != nil {
		model.PartnerID = in.PartnerID
	}
	if in.PartnerCategoryID != nil {
		model.PartnerCategoryID = in.PartnerCategoryID
	}
	if in.CompanyID != nil {
		model.CompanyID = in.CompanyID
	}
	if in.Distribution != nil {
		model.Distribution = in.Distribution
	}
	if in.Active != nil {
		model.Active = *in.Active
	}
	if err := model.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateDistributionModel(ctx, model); err != nil {
		return nil, err
	}
	return model, nil
}

func (uc *UseCase) DeleteDistributionModel(ctx context.Context, id int64) error {
	if err := uc.repo.DeleteDistributionModel(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "analytic distribution model deleted", "id", id)
	return nil
}

func (uc *UseCase) ListDistributionModels(ctx context.Context) ([]analytic.DistributionModel, error) {
	return uc.repo.ListDistributionModels(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Project Plan (G8)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) GetProjectPlanID(ctx context.Context) (int64, error) {
	return uc.repo.GetProjectPlanID(ctx)
}

func (uc *UseCase) SetProjectPlanID(ctx context.Context, planID int64) error {
	if err := uc.repo.SetProjectPlanID(ctx, planID); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "analytic project plan configured", "plan_id", planID)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Integration helper for accounting move lines
// ─────────────────────────────────────────────────────────────────────────────

// MoveLineToAnalyticInput converts an accounting move line into the analytic input
// expected by CreateLinesFromMoveLine (used by the accounting module).
func MoveLineToAnalyticInput(move accounting.AccountMoveLine, userID int64, companyID int64, distribution analytic.AnalyticDistribution) MoveLineAnalyticInput {
	return MoveLineAnalyticInput{
		MoveLineID:       move.ID,
		GeneralAccountID: move.AccountID,
		Name:             move.Name,
		Date:             time.Now().UTC(),
		PartnerID:        move.PartnerID,
		UserID:           userID,
		CompanyID:        companyID,
		Currency:         "USD",
		Amount:           move.Balance,
		UnitAmount:       move.Quantity,
		Source:           string(analytic.SourceInvoice),
		Distribution:     distribution,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) ensurePlanExists(ctx context.Context, planID int64) error {
	if _, err := uc.repo.GetPlanByID(ctx, planID); err != nil {
		return platformerrors.Validation("analytic plan does not exist", map[string]string{
			"plan_id": fmt.Sprintf("%d", planID),
		})
	}
	return nil
}

func roundAmount(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func sortedKeys(d analytic.AnalyticDistribution) []string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// firstAccountID extracts the first account reference from a distribution key
// (a key may combine several account IDs as "1,5" - G3).
func firstAccountID(key string) (int64, error) {
	for _, part := range strings.Split(key, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		return id, nil
	}
	return 0, fmt.Errorf("no valid account id in key %q", key)
}

func keyIntersectsAnyAccount(key string, accountIDs []int64) bool {
	seen := make(map[int64]bool, len(accountIDs))
	for _, id := range accountIDs {
		seen[id] = true
	}
	for _, part := range strings.Split(key, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && seen[id] {
			return true
		}
	}
	return false
}