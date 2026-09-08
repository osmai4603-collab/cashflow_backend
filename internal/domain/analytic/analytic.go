package analytic

import (
	"math"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// Applicability determines how an analytic plan is enforced for a given business domain.
type Applicability string

const (
	AppOptional    Applicability = "optional"    // Plan may be used, no enforcement
	AppMandatory   Applicability = "mandatory"   // Distribution must total 100% for this plan
	AppUnavailable Applicability = "unavailable" // Plan is hidden for this context
)

// AllValidApplicabilities lists every supported applicability value.
var AllValidApplicabilities = map[Applicability]bool{
	AppOptional:    true,
	AppMandatory:   true,
	AppUnavailable: true,
}

// BusinessDomain identifies which business document a rule applies to (extensible).
type BusinessDomain string

const (
	DomainGeneral       BusinessDomain = "general"        // Any document
	DomainSaleOrder     BusinessDomain = "sale_order"     // Added later (M4)
	DomainPurchaseOrder BusinessDomain = "purchase_order" // Added later (M5)
	DomainExpense       BusinessDomain = "expense"        // Added later (M19)
)

// AllValidBusinessDomains lists every supported business domain.
var AllValidBusinessDomains = map[BusinessDomain]bool{
	DomainGeneral:       true,
	DomainSaleOrder:     true,
	DomainPurchaseOrder: true,
	DomainExpense:       true,
}

// LineSource identifies the origin of an analytic line (account.analytic.line "source").
type LineSource string

const (
	SourceManual     LineSource = "manual"
	SourceInvoice    LineSource = "invoice"
	SourceVendorBill LineSource = "vendor_bill"
	SourceSaleOrder  LineSource = "sale_order"
	SourceEmployee   LineSource = "employee"
)

// AllValidLineSources lists every supported line source.
var AllValidLineSources = map[LineSource]bool{
	SourceManual:     true,
	SourceInvoice:    true,
	SourceVendorBill: true,
	SourceSaleOrder:  true,
	SourceEmployee:   true,
}

// AnalyticPlan represents an analytic accounting plan (account.analytic.plan in Odoo).
type AnalyticPlan struct {
	ID                  int64                   `json:"id"`
	Name                i18n.TranslationString                  `json:"name"`
	Description         string                  `json:"description,omitempty"`
	ParentID            *int64                  `json:"parent_id,omitempty"`
	ParentPath          string                  `json:"parent_path,omitempty"`
	RootID              int64                   `json:"root_id"`
	CompleteName        i18n.TranslationString                  `json:"complete_name,omitempty"` // computed "parent / name"
	Sequence            int                     `json:"sequence"`
	Color               int                     `json:"color"`
	DefaultApplicability Applicability          `json:"default_applicability"` // company-dependent by default (G2)
	Applicabilities     []AnalyticApplicability `json:"applicabilities,omitempty"`
	Active              bool                    `json:"active"`
	Audit               audit.Fields            `json:"audit"`
}

// IsRoot returns true when the plan is a top-level plan (no parent).
func (p *AnalyticPlan) IsRoot() bool {
	return p.ParentID == nil
}

// Validate checks AnalyticPlan constraints and invariants.
func (p *AnalyticPlan) Validate() error {
	if len(p.Name) == 0 {
		return platformerrors.Validation("analytic plan name is required", nil)
	}

	if p.ParentID != nil && p.ID > 0 && *p.ParentID == p.ID {
		return platformerrors.Validation("analytic plan cannot be its own parent", map[string]string{
			"parent_id": "circular reference detected",
		})
	}

	if p.DefaultApplicability == "" {
		p.DefaultApplicability = AppOptional
	}
	if !AllValidApplicabilities[p.DefaultApplicability] {
		return platformerrors.Validation("invalid default applicability", map[string]string{
			"default_applicability": string(p.DefaultApplicability),
		})
	}

	return nil
}

// BuildCompleteName recomputes the hierarchical display name "parent / name".
func (p *AnalyticPlan) BuildCompleteName() i18n.TranslationString {
	// This is tricky for TranslationString.
	// We'll return the name as complete_name for now,
	// or implement a merge logic if needed.
	return p.Name
}

// AnalyticApplicability represents an independent applicability rule scoped to a
// company and a business domain (account.analytic.applicability in Odoo - G2).
type AnalyticApplicability struct {
	ID             int64          `json:"id"`
	PlanID         int64          `json:"plan_id"`
	BusinessDomain BusinessDomain `json:"business_domain"`
	Applicability  Applicability  `json:"applicability"`
	CompanyID      *int64         `json:"company_id,omitempty"` // nil = applies to all companies
	Sequence       int            `json:"sequence"`
}

// Validate checks AnalyticApplicability constraints.
func (a *AnalyticApplicability) Validate() error {
	if a.PlanID <= 0 {
		return platformerrors.Validation("analytic applicability requires a plan", nil)
	}
	if a.BusinessDomain == "" {
		a.BusinessDomain = DomainGeneral
	}
	if !AllValidBusinessDomains[a.BusinessDomain] {
		return platformerrors.Validation("invalid business domain", map[string]string{
			"business_domain": string(a.BusinessDomain),
		})
	}
	if !AllValidApplicabilities[a.Applicability] {
		return platformerrors.Validation("invalid applicability", map[string]string{
			"applicability": string(a.Applicability),
		})
	}
	return nil
}

// AnalyticAccount represents an analytic account (account.analytic.account in Odoo).
type AnalyticAccount struct {
	ID         int64        `json:"id"`
	Name       i18n.TranslationString       `json:"name"`
	Code       string       `json:"code,omitempty"`
	PlanID     int64        `json:"plan_id"`
	RootPlanID int64        `json:"root_plan_id"`
	PartnerID  *int64       `json:"partner_id,omitempty"`
	Color      int          `json:"color"`
	CompanyID  *int64       `json:"company_id,omitempty"` // nil = shared across companies
	Active     bool         `json:"active"`
	Debit      float64      `json:"debit"`   // computed, in company currency
	Credit     float64      `json:"credit"`  // computed, in company currency
	Balance    float64      `json:"balance"` // computed = credit - debit
	Currency   string       `json:"currency"`
	Audit      audit.Fields `json:"audit"`
}

// Validate checks AnalyticAccount constraints.
func (a *AnalyticAccount) Validate() error {
	if len(a.Name) == 0 {
		return platformerrors.Validation("analytic account name is required", nil)
	}
	if a.PlanID <= 0 {
		return platformerrors.Validation("analytic account requires a plan", map[string]string{
			"plan_id": "must reference a valid analytic plan",
		})
	}
	if a.Currency == "" {
		a.Currency = "USD"
	}
	return nil
}

// AnalyticLine represents a single analytic accounting entry
// (account.analytic.line in Odoo, one row per distribution split).
type AnalyticLine struct {
	ID               int64        `json:"id"`
	Name             i18n.TranslationString       `json:"name"` // label / description (G5)
	Date             time.Time    `json:"date"`
	Amount           float64      `json:"amount"`
	UnitAmount       float64      `json:"unit_amount"` // quantity
	ProductUoMID     *int64       `json:"product_uom_id,omitempty"`
	PartnerID        *int64       `json:"partner_id,omitempty"`
	UserID           int64        `json:"user_id"`
	CompanyID        int64        `json:"company_id"`
	Currency         string       `json:"currency"`
	Category         string       `json:"category"`
	AccountID        int64        `json:"account_id"` // "Project plan" column (G4)
	MoveLineID       *int64       `json:"move_line_id,omitempty"`
	GeneralAccountID *int64       `json:"general_account_id,omitempty"`
	Source           LineSource   `json:"source"`
	Distribution     AnalyticDistribution `json:"distribution,omitempty"` // G3: searchable JSONB snapshot
	Audit            audit.Fields `json:"audit"`
}

// Validate checks AnalyticLine constraints and invariants.
func (l *AnalyticLine) Validate() error {
	if len(l.Name) == 0 {
		return platformerrors.Validation("analytic line name is required", nil)
	}
	if l.Date.IsZero() {
		return platformerrors.Validation("analytic line date is required", nil)
	}
	if l.AccountID <= 0 {
		return platformerrors.Validation("an analytic account must be assigned", map[string]string{
			"account_id": "at least one analytic account is required (G4)",
		})
	}
	if l.UserID <= 0 {
		return platformerrors.Validation("analytic line user is required", nil)
	}
	if l.CompanyID <= 0 {
		return platformerrors.Validation("analytic line company is required", nil)
	}
	if l.Source == "" {
		l.Source = SourceManual
	}
	if !AllValidLineSources[l.Source] {
		return platformerrors.Validation("invalid analytic line source", map[string]string{
			"source": string(l.Source),
		})
	}
	if l.Category == "" {
		l.Category = "other"
	}
	if l.Currency == "" {
		l.Currency = "USD"
	}
	return nil
}

// DebitCreditBalance aggregates the debit, credit, and net balance of an analytic account.
type DebitCreditBalance struct {
	Debit    float64 `json:"debit"`
	Credit   float64 `json:"credit"`
	Balance  float64 `json:"balance"` // credit - debit
	Currency string  `json:"currency"`
}

// ComputeBalance derives the net balance from debit/credit.
func (b *DebitCreditBalance) ComputeBalance() {
	b.Balance = roundAmount(b.Credit - b.Debit)
}

func roundAmount(v float64) float64 {
	return math.Round(v*10000) / 10000
}