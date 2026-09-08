package analytichttp

import (
	"time"

	"cashflow_backend/internal/domain/analytic"
	analyticusecase "cashflow_backend/internal/usecase/analytic"
)

// ─────────────────────────────────────────────────────────────────────────────
// Plans DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreatePlanRequest struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	ParentID             *int64 `json:"parent_id"`
	Sequence             int    `json:"sequence"`
	Color                int    `json:"color"`
	DefaultApplicability string `json:"default_applicability"`
}

func (r CreatePlanRequest) ToInput() analyticusecase.CreatePlanInput {
	return analyticusecase.CreatePlanInput{
		Name:                 r.Name,
		Description:          r.Description,
		ParentID:             r.ParentID,
		Sequence:             r.Sequence,
		Color:                r.Color,
		DefaultApplicability: r.DefaultApplicability,
	}
}

type UpdatePlanRequest struct {
	Name                 *string `json:"name"`
	Description          *string `json:"description"`
	ParentID             *int64  `json:"parent_id"`
	Sequence             *int    `json:"sequence"`
	Color                *int    `json:"color"`
	DefaultApplicability *string `json:"default_applicability"`
	Active               *bool   `json:"active"`
}

func (r UpdatePlanRequest) ToInput() analyticusecase.UpdatePlanInput {
	return analyticusecase.UpdatePlanInput{
		Name:                 r.Name,
		Description:          r.Description,
		ParentID:             r.ParentID,
		Sequence:             r.Sequence,
		Color:                r.Color,
		DefaultApplicability: r.DefaultApplicability,
		Active:               r.Active,
	}
}

type SetApplicabilityRequest struct {
	BusinessDomain string `json:"business_domain"`
	Applicability  string `json:"applicability"`
	CompanyID      *int64 `json:"company_id"`
	Sequence       int    `json:"sequence"`
}

func (r SetApplicabilityRequest) ToInput(planID int64) analyticusecase.SetApplicabilityInput {
	return analyticusecase.SetApplicabilityInput{
		PlanID:         planID,
		BusinessDomain: r.BusinessDomain,
		Applicability:  r.Applicability,
		CompanyID:      r.CompanyID,
		Sequence:       r.Sequence,
	}
}

type ApplicabilityResponse struct {
	ID             int64  `json:"id"`
	PlanID         int64  `json:"plan_id"`
	BusinessDomain string `json:"business_domain"`
	Applicability  string `json:"applicability"`
	CompanyID      *int64 `json:"company_id,omitempty"`
	Sequence       int    `json:"sequence"`
}

func ToApplicabilityResponse(a analytic.AnalyticApplicability) ApplicabilityResponse {
	return ApplicabilityResponse{
		ID:             a.ID,
		PlanID:         a.PlanID,
		BusinessDomain: string(a.BusinessDomain),
		Applicability:  string(a.Applicability),
		CompanyID:      a.CompanyID,
		Sequence:       a.Sequence,
	}
}

type PlanResponse struct {
	ID                   int64                   `json:"id"`
	Name                 string                  `json:"name"`
	Description          string                  `json:"description,omitempty"`
	ParentID             *int64                  `json:"parent_id,omitempty"`
	ParentPath           string                  `json:"parent_path,omitempty"`
	RootID               int64                   `json:"root_id"`
	CompleteName         string                  `json:"complete_name,omitempty"`
	Sequence             int                     `json:"sequence"`
	Color                int                     `json:"color"`
	DefaultApplicability string                  `json:"default_applicability"`
	Applicabilities      []ApplicabilityResponse `json:"applicabilities,omitempty"`
	Active               bool                    `json:"active"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

func ToPlanResponse(p *analytic.AnalyticPlan) PlanResponse {
	resp := PlanResponse{
		ID:                   p.ID,
		Name:                 string(p.Name),
		Description:          p.Description,
		ParentID:             p.ParentID,
		ParentPath:           p.ParentPath,
		RootID:               p.RootID,
		CompleteName:         string(p.CompleteName),
		Sequence:             p.Sequence,
		Color:                p.Color,
		DefaultApplicability: string(p.DefaultApplicability),
		Active:               p.Active,
		CreatedAt:            p.Audit.CreatedAt,
		UpdatedAt:            p.Audit.UpdatedAt,
	}
	if len(p.Applicabilities) > 0 {
		resp.Applicabilities = make([]ApplicabilityResponse, len(p.Applicabilities))
		for i, a := range p.Applicabilities {
			resp.Applicabilities[i] = ToApplicabilityResponse(a)
		}
	}
	return resp
}

type PlanStructureResponse struct {
	Plan     PlanResponse      `json:"plan"`
	Accounts []AccountResponse `json:"accounts"`
}

func ToPlanStructureResponse(s *analyticusecase.PlanStructure) PlanStructureResponse {
	accounts := make([]AccountResponse, len(s.Accounts))
	for i := range s.Accounts {
		accounts[i] = ToAccountResponse(&s.Accounts[i])
	}
	return PlanStructureResponse{
		Plan:     ToPlanResponse(&s.Plan),
		Accounts: accounts,
	}
}

type RelevantPlanResponse struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Applicability string `json:"applicability"`
	ColumnName    string `json:"column_name"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Accounts DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateAccountRequest struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	PlanID    int64  `json:"plan_id"`
	PartnerID *int64 `json:"partner_id"`
	Color     int    `json:"color"`
	CompanyID *int64 `json:"company_id"`
	Currency  string `json:"currency"`
}

func (r CreateAccountRequest) ToInput() analyticusecase.CreateAccountInput {
	return analyticusecase.CreateAccountInput{
		Name:      r.Name,
		Code:      r.Code,
		PlanID:    r.PlanID,
		PartnerID: r.PartnerID,
		Color:     r.Color,
		CompanyID: r.CompanyID,
		Currency:  r.Currency,
	}
}

type UpdateAccountRequest struct {
	Name      *string `json:"name"`
	Code      *string `json:"code"`
	PlanID    *int64  `json:"plan_id"`
	PartnerID *int64  `json:"partner_id"`
	Color     *int    `json:"color"`
	CompanyID *int64  `json:"company_id"`
	Active    *bool   `json:"active"`
}

func (r UpdateAccountRequest) ToInput() analyticusecase.UpdateAccountInput {
	return analyticusecase.UpdateAccountInput{
		Name:      r.Name,
		Code:      r.Code,
		PlanID:    r.PlanID,
		PartnerID: r.PartnerID,
		Color:     r.Color,
		CompanyID: r.CompanyID,
		Active:    r.Active,
	}
}

type AccountResponse struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Code       string    `json:"code,omitempty"`
	PlanID     int64     `json:"plan_id"`
	RootPlanID int64     `json:"root_plan_id"`
	PartnerID  *int64    `json:"partner_id,omitempty"`
	Color      int       `json:"color"`
	CompanyID  *int64    `json:"company_id,omitempty"`
	Active     bool      `json:"active"`
	Debit      float64   `json:"debit"`
	Credit     float64   `json:"credit"`
	Balance    float64   `json:"balance"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ToAccountResponse(a *analytic.AnalyticAccount) AccountResponse {
	return AccountResponse{
		ID:         a.ID,
		Name:       string(a.Name),
		Code:       a.Code,
		PlanID:     a.PlanID,
		RootPlanID: a.RootPlanID,
		PartnerID:  a.PartnerID,
		Color:      a.Color,
		CompanyID:  a.CompanyID,
		Active:     a.Active,
		Debit:      a.Debit,
		Credit:     a.Credit,
		Balance:    a.Balance,
		Currency:   a.Currency,
		CreatedAt:  a.Audit.CreatedAt,
		UpdatedAt:  a.Audit.UpdatedAt,
	}
}

type BalanceResponse struct {
	Debit    float64 `json:"debit"`
	Credit   float64 `json:"credit"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Lines DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateLineRequest struct {
	Name             string                        `json:"name"`
	Date             time.Time                     `json:"date"`
	Amount           float64                       `json:"amount"`
	UnitAmount       float64                       `json:"unit_amount"`
	ProductUoMID     *int64                        `json:"product_uom_id"`
	PartnerID        *int64                        `json:"partner_id"`
	UserID           int64                         `json:"user_id"`
	CompanyID        int64                         `json:"company_id"`
	Currency         string                        `json:"currency"`
	Category         string                        `json:"category"`
	AccountID        int64                         `json:"account_id"`
	MoveLineID       *int64                        `json:"move_line_id"`
	GeneralAccountID *int64                        `json:"general_account_id"`
	Source           string                        `json:"source"`
	Distribution     analytic.AnalyticDistribution `json:"distribution,omitempty"`
}

func (r CreateLineRequest) ToInput() analyticusecase.CreateLineInput {
	return analyticusecase.CreateLineInput{
		Name:             r.Name,
		Date:             r.Date,
		Amount:           r.Amount,
		UnitAmount:       r.UnitAmount,
		ProductUoMID:     r.ProductUoMID,
		PartnerID:        r.PartnerID,
		UserID:           r.UserID,
		CompanyID:        r.CompanyID,
		Currency:         r.Currency,
		Category:         r.Category,
		AccountID:        r.AccountID,
		MoveLineID:       r.MoveLineID,
		GeneralAccountID: r.GeneralAccountID,
		Source:           r.Source,
		Distribution:     r.Distribution,
	}
}

type UpdateLineRequest struct {
	Name         *string                       `json:"name"`
	Date         *time.Time                    `json:"date"`
	Amount       *float64                      `json:"amount"`
	UnitAmount   *float64                      `json:"unit_amount"`
	Category     *string                       `json:"category"`
	AccountID    *int64                        `json:"account_id"`
	Distribution analytic.AnalyticDistribution `json:"distribution,omitempty"`
}

func (r UpdateLineRequest) ToInput() analyticusecase.UpdateLineInput {
	return analyticusecase.UpdateLineInput{
		Name:         r.Name,
		Date:         r.Date,
		Amount:       r.Amount,
		UnitAmount:   r.UnitAmount,
		Category:     r.Category,
		AccountID:    r.AccountID,
		Distribution: r.Distribution,
	}
}

type LineResponse struct {
	ID               int64                         `json:"id"`
	Name             string                        `json:"name"`
	Date             time.Time                     `json:"date"`
	Amount           float64                       `json:"amount"`
	UnitAmount       float64                       `json:"unit_amount"`
	UserID           int64                         `json:"user_id"`
	CompanyID        int64                         `json:"company_id"`
	Currency         string                        `json:"currency"`
	Category         string                        `json:"category"`
	AccountID        int64                         `json:"account_id"`
	MoveLineID       *int64                        `json:"move_line_id,omitempty"`
	GeneralAccountID *int64                        `json:"general_account_id,omitempty"`
	Source           string                        `json:"source"`
	Distribution     analytic.AnalyticDistribution `json:"distribution,omitempty"`
	CreatedAt        time.Time                     `json:"created_at"`
	UpdatedAt        time.Time                     `json:"updated_at"`
}

func ToLineResponse(l *analytic.AnalyticLine) LineResponse {
	return LineResponse{
		ID:               l.ID,
		Name:             string(l.Name),
		Date:             l.Date,
		Amount:           l.Amount,
		UnitAmount:       l.UnitAmount,
		UserID:           l.UserID,
		CompanyID:        l.CompanyID,
		Currency:         l.Currency,
		Category:         l.Category,
		AccountID:        l.AccountID,
		MoveLineID:       l.MoveLineID,
		GeneralAccountID: l.GeneralAccountID,
		Source:           string(l.Source),
		Distribution:     l.Distribution,
		CreatedAt:        l.Audit.CreatedAt,
		UpdatedAt:        l.Audit.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Distribution Model DTOs (G7)
// ─────────────────────────────────────────────────────────────────────────────

type CreateDistributionModelRequest struct {
	Sequence          int                           `json:"sequence"`
	PartnerID         *int64                        `json:"partner_id"`
	PartnerCategoryID *int64                        `json:"partner_category_id"`
	CompanyID         *int64                        `json:"company_id"`
	Distribution      analytic.AnalyticDistribution `json:"distribution"`
}

func (r CreateDistributionModelRequest) ToInput() analyticusecase.CreateDistributionModelInput {
	return analyticusecase.CreateDistributionModelInput{
		Sequence:          r.Sequence,
		PartnerID:         r.PartnerID,
		PartnerCategoryID: r.PartnerCategoryID,
		CompanyID:         r.CompanyID,
		Distribution:      r.Distribution,
	}
}

type UpdateDistributionModelRequest struct {
	Sequence          *int                          `json:"sequence"`
	PartnerID         *int64                        `json:"partner_id"`
	PartnerCategoryID *int64                        `json:"partner_category_id"`
	CompanyID         *int64                        `json:"company_id"`
	Distribution      analytic.AnalyticDistribution `json:"distribution"`
	Active            *bool                         `json:"active"`
}

func (r UpdateDistributionModelRequest) ToInput() analyticusecase.UpdateDistributionModelInput {
	return analyticusecase.UpdateDistributionModelInput{
		Sequence:          r.Sequence,
		PartnerID:         r.PartnerID,
		PartnerCategoryID: r.PartnerCategoryID,
		CompanyID:         r.CompanyID,
		Distribution:      r.Distribution,
		Active:            r.Active,
	}
}

type DistributionModelResponse struct {
	ID                int64                         `json:"id"`
	Sequence          int                           `json:"sequence"`
	PartnerID         *int64                        `json:"partner_id,omitempty"`
	PartnerCategoryID *int64                        `json:"partner_category_id,omitempty"`
	CompanyID         *int64                        `json:"company_id,omitempty"`
	Distribution      analytic.AnalyticDistribution `json:"distribution"`
	Active            bool                          `json:"active"`
	CreatedAt         time.Time                     `json:"created_at"`
	UpdatedAt         time.Time                     `json:"updated_at"`
}

func ToDistributionModelResponse(m *analytic.DistributionModel) DistributionModelResponse {
	return DistributionModelResponse{
		ID:                m.ID,
		Sequence:          m.Sequence,
		PartnerID:         m.PartnerID,
		PartnerCategoryID: m.PartnerCategoryID,
		CompanyID:         m.CompanyID,
		Distribution:      m.Distribution,
		Active:            m.Active,
		CreatedAt:         m.Audit.CreatedAt,
		UpdatedAt:         m.Audit.UpdatedAt,
	}
}

type MatchDistributionRequest struct {
	PartnerID         *int64 `json:"partner_id"`
	PartnerCategoryID *int64 `json:"partner_category_id"`
	CompanyID         int64  `json:"company_id"`
}

type DistributeMoveLineRequest struct {
	GeneralAccountID int64                         `json:"general_account_id"`
	Name             string                        `json:"name"`
	Date             time.Time                     `json:"date"`
	PartnerID        *int64                        `json:"partner_id"`
	UserID           int64                         `json:"user_id"`
	CompanyID        int64                         `json:"company_id"`
	Currency         string                        `json:"currency"`
	Amount           float64                       `json:"amount"`
	UnitAmount       float64                       `json:"unit_amount"`
	Source           string                        `json:"source"`
	Distribution     analytic.AnalyticDistribution `json:"distribution"`
}

func (r DistributeMoveLineRequest) ToInput(moveLineID int64) analyticusecase.MoveLineAnalyticInput {
	return analyticusecase.MoveLineAnalyticInput{
		MoveLineID:       moveLineID,
		GeneralAccountID: r.GeneralAccountID,
		Name:             r.Name,
		Date:             r.Date,
		PartnerID:        r.PartnerID,
		UserID:           r.UserID,
		CompanyID:        r.CompanyID,
		Currency:         r.Currency,
		Amount:           r.Amount,
		UnitAmount:       r.UnitAmount,
		Source:           r.Source,
		Distribution:     r.Distribution,
	}
}

type ProjectPlanRequest struct {
	PlanID int64 `json:"plan_id"`
}

type ProjectPlanResponse struct {
	PlanID int64 `json:"plan_id"`
}
