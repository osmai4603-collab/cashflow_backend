package crmusecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/sale"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/i18n"
	"cashflow_backend/internal/platform/pagination"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	saleusecase "cashflow_backend/internal/usecase/sale"
)

// PartnerService abstracts partner operations required for automatic partner conversion.
type PartnerService interface {
	CreatePartner(ctx context.Context, in partnerusecase.CreatePartnerInput) (*partner.Partner, error)
	GetPartner(ctx context.Context, id int64) (*partner.Partner, error)
}

// SaleService abstracts sale order creation operations when an opportunity is won.
type SaleService interface {
	CreateOrder(ctx context.Context, in saleusecase.CreateSaleOrderInput) (*sale.SaleOrder, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateLeadInput struct {
	Name            string     `json:"name"`
	Type            string     `json:"type"` // "lead" or "opportunity"
	PartnerID       *int64     `json:"partner_id"`
	PartnerName     string     `json:"partner_name"`
	ContactName     string     `json:"contact_name"`
	EmailFrom       string     `json:"email_from"`
	Phone           string     `json:"phone"`
	StageID         *int64     `json:"stage_id"`
	SalespersonID   *int64     `json:"salesperson_id"`
	ExpectedRevenue float64    `json:"expected_revenue"`
	Probability     *float64   `json:"probability"`
	Source          string     `json:"source"`
	Priority        string     `json:"priority"` // "0", "1", "2", "3"
	DateDeadline    *time.Time `json:"date_deadline"`
	Notes           string     `json:"notes"`
	CompanyID       *int64     `json:"company_id"`
	TagIDs          []int64    `json:"tag_ids"`
}

type UpdateLeadInput struct {
	Name            *string    `json:"name"`
	Type            *string    `json:"type"`
	PartnerID       *int64     `json:"partner_id"`
	PartnerName     *string    `json:"partner_name"`
	ContactName     *string    `json:"contact_name"`
	EmailFrom       *string    `json:"email_from"`
	Phone           *string    `json:"phone"`
	StageID         *int64     `json:"stage_id"`
	SalespersonID   *int64     `json:"salesperson_id"`
	ExpectedRevenue *float64   `json:"expected_revenue"`
	Probability     *float64   `json:"probability"`
	Source          *string    `json:"source"`
	Priority        *string    `json:"priority"`
	DateDeadline    *time.Time `json:"date_deadline"`
	Notes           *string    `json:"notes"`
	CompanyID       *int64     `json:"company_id"`
	Active          *bool      `json:"active"`
	TagIDs          []int64    `json:"tag_ids"`
}

type ConvertLeadInput struct {
	StageID       *int64 `json:"stage_id"`
	PartnerID     *int64 `json:"partner_id"`
	CreatePartner bool   `json:"create_partner"` // Default true if no partner_id
}

type MarkWonInput struct {
	StageID         *int64 `json:"stage_id"`
	CreateSaleOrder bool   `json:"create_sale_order"`
}

type MarkLostInput struct {
	LostReasonID int64  `json:"lost_reason_id"`
	LostFeedback string `json:"lost_feedback"`
}

type CreateStageInput struct {
	Name         string `json:"name"`
	Sequence     int    `json:"sequence"`
	IsWon        bool   `json:"is_won"`
	IsClosed     bool   `json:"is_closed"`
	Fold         bool   `json:"fold"`
	Requirements string `json:"requirements"`
	CompanyID    *int64 `json:"company_id"`
}

type UpdateStageInput struct {
	Name         *string `json:"name"`
	Sequence     *int    `json:"sequence"`
	IsWon        *bool   `json:"is_won"`
	IsClosed     *bool   `json:"is_closed"`
	Fold         *bool   `json:"fold"`
	Requirements *string `json:"requirements"`
	CompanyID    *int64  `json:"company_id"`
	Active       *bool   `json:"active"`
}

type CreateLostReasonInput struct {
	Name string `json:"name"`
}

type UpdateLostReasonInput struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

type CreateTagInput struct {
	Name  string `json:"name"`
	Color int    `json:"color"`
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Implementation
// ─────────────────────────────────────────────────────────────────────────────

type UseCase struct {
	repo      crm.Repository
	partnerUC PartnerService
	saleUC    SaleService
	logger    *slog.Logger
}

// New initializes a CRM UseCase instance with injected dependencies.
func New(repo crm.Repository, partnerUC PartnerService, saleUC SaleService, logger *slog.Logger) *UseCase {
	return &UseCase{
		repo:      repo,
		partnerUC: partnerUC,
		saleUC:    saleUC,
		logger:    logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Lead & Opportunity Operations
// ─────────────────────────────────────────────────────────────────────────────

func (u *UseCase) CreateLead(ctx context.Context, in CreateLeadInput) (*crm.Lead, error) {
	leadType := crm.LeadTypeLead
	if strings.EqualFold(in.Type, string(crm.LeadTypeOpportunity)) {
		leadType = crm.LeadTypeOpportunity
	}

	stageID := int64(0)
	if in.StageID != nil && *in.StageID > 0 {
		stageID = *in.StageID
	} else {
		initStage, err := u.repo.GetInitialStage(ctx)
		if err != nil {
			return nil, platformerrors.Internal("failed to resolve initial stage", err)
		}
		stageID = initStage.ID
	}

	priority := crm.PriorityNormal
	if in.Priority != "" {
		priority = crm.Priority(in.Priority)
	}

	probability := 0.0
	if in.Probability != nil {
		probability = *in.Probability
	} else {
		// Default probabilities based on standard Odoo stages
		stage, err := u.repo.GetStageByID(ctx, stageID)
		if err == nil && stage != nil {
			switch {
			case stage.IsWon:
				probability = 100.0
			case strings.EqualFold(string(stage.Name), "Proposition"):
				probability = 70.0
			case strings.EqualFold(string(stage.Name), "Qualified"):
				probability = 30.0
			case strings.EqualFold(string(stage.Name), "New"):
				probability = 10.0
			}
		}
	}

	lead := &crm.Lead{
		Name:            in.Name,
		Type:            leadType,
		PartnerID:       in.PartnerID,
		PartnerName:     in.PartnerName,
		ContactName:     in.ContactName,
		EmailFrom:       in.EmailFrom,
		Phone:           in.Phone,
		StageID:         stageID,
		SalespersonID:   in.SalespersonID,
		ExpectedRevenue: in.ExpectedRevenue,
		Probability:     probability,
		Source:          in.Source,
		Priority:        priority,
		DateDeadline:    in.DateDeadline,
		Notes:           in.Notes,
		CompanyID:       in.CompanyID,
		Active:          true,
		TagIDs:          in.TagIDs,
	}

	if err := lead.Validate(); err != nil {
		return nil, err
	}

	if err := u.repo.CreateLead(ctx, lead); err != nil {
		return nil, err
	}

	u.logger.Info("created CRM record", "id", lead.ID, "name", lead.Name, "type", lead.Type)
	return lead, nil
}

func (u *UseCase) GetLead(ctx context.Context, id int64) (*crm.Lead, error) {
	return u.repo.GetLeadByID(ctx, id)
}

func (u *UseCase) UpdateLead(ctx context.Context, id int64, in UpdateLeadInput) (*crm.Lead, error) {
	lead, err := u.repo.GetLeadByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		lead.Name = *in.Name
	}
	if in.Type != nil {
		lead.Type = crm.LeadType(*in.Type)
	}
	if in.PartnerID != nil {
		lead.PartnerID = in.PartnerID
	}
	if in.PartnerName != nil {
		lead.PartnerName = *in.PartnerName
	}
	if in.ContactName != nil {
		lead.ContactName = *in.ContactName
	}
	if in.EmailFrom != nil {
		lead.EmailFrom = *in.EmailFrom
	}
	if in.Phone != nil {
		lead.Phone = *in.Phone
	}
	if in.StageID != nil && *in.StageID > 0 {
		lead.StageID = *in.StageID
	}
	if in.SalespersonID != nil {
		lead.SalespersonID = in.SalespersonID
	}
	if in.ExpectedRevenue != nil {
		lead.ExpectedRevenue = *in.ExpectedRevenue
	}
	if in.Probability != nil {
		lead.Probability = *in.Probability
	}
	if in.Source != nil {
		lead.Source = *in.Source
	}
	if in.Priority != nil {
		lead.Priority = crm.Priority(*in.Priority)
	}
	if in.DateDeadline != nil {
		lead.DateDeadline = in.DateDeadline
	}
	if in.Notes != nil {
		lead.Notes = *in.Notes
	}
	if in.CompanyID != nil {
		lead.CompanyID = in.CompanyID
	}
	if in.Active != nil {
		lead.Active = *in.Active
	}
	if in.TagIDs != nil {
		lead.TagIDs = in.TagIDs
	}

	if err := lead.Validate(); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateLead(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

func (u *UseCase) DeleteLead(ctx context.Context, id int64) error {
	return u.repo.DeleteLead(ctx, id)
}

func (u *UseCase) ListLeads(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[crm.Lead], error) {
	return u.repo.ListLeads(ctx, f, page)
}

// ConvertLead converts a lead into an opportunity and optionally creates a linked partner.
func (u *UseCase) ConvertLead(ctx context.Context, id int64, in ConvertLeadInput) (*crm.Lead, error) {
	lead, err := u.repo.GetLeadByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Determine destination stage
	stageID := lead.StageID
	if in.StageID != nil && *in.StageID > 0 {
		stageID = *in.StageID
	} else {
		stages, err := u.repo.ListStages(ctx)
		if err == nil && len(stages) > 0 {
			stageID = stages[0].ID
		}
	}

	// Resolve Partner
	partnerID := lead.PartnerID
	if in.PartnerID != nil && *in.PartnerID > 0 {
		partnerID = in.PartnerID
	} else if partnerID == nil && in.CreatePartner && u.partnerUC != nil {
		// Auto-create partner from lead details
		partnerName := strings.TrimSpace(lead.PartnerName)
		if partnerName == "" {
			partnerName = strings.TrimSpace(lead.ContactName)
		}
		if partnerName == "" {
			partnerName = lead.Name
		}

		isCustomer := true
		isSupplier := false
		pType := partner.PartnerTypeCompany
		if lead.PartnerName == "" && lead.ContactName != "" {
			pType = partner.PartnerTypeIndividual
		}

		newPartner, err := u.partnerUC.CreatePartner(ctx, partnerusecase.CreatePartnerInput{
			Name:       partnerName,
			Email:      lead.EmailFrom,
			Phone:      lead.Phone,
			Type:       pType,
			IsCustomer: &isCustomer,
			IsSupplier: &isSupplier,
			CompanyID:  lead.CompanyID,
		})

		if err == nil && newPartner != nil {
			partnerID = &newPartner.ID
			u.logger.Info("auto-created partner during CRM conversion", "partner_id", newPartner.ID, "name", newPartner.Name)
		} else if err != nil {
			u.logger.Warn("could not auto-create partner during conversion", "error", err)
		}
	}

	if err := lead.ActionConvert(stageID, partnerID); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateLead(ctx, lead); err != nil {
		return nil, err
	}

	u.logger.Info("converted lead to opportunity", "id", lead.ID, "partner_id", lead.PartnerID, "stage_id", lead.StageID)
	return lead, nil
}

// MarkLeadWon marks the opportunity as Won and optionally spawns a draft quotation.
func (u *UseCase) MarkLeadWon(ctx context.Context, id int64, in MarkWonInput) (*crm.Lead, *sale.SaleOrder, error) {
	lead, err := u.repo.GetLeadByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	wonStageID := int64(0)
	if in.StageID != nil && *in.StageID > 0 {
		wonStageID = *in.StageID
	} else {
		wonStage, err := u.repo.GetWonStage(ctx)
		if err != nil {
			return nil, nil, err
		}
		wonStageID = wonStage.ID
	}

	if err := lead.ActionMarkWon(wonStageID); err != nil {
		return nil, nil, err
	}

	if err := u.repo.UpdateLead(ctx, lead); err != nil {
		return nil, nil, err
	}

	var generatedOrder *sale.SaleOrder
	if in.CreateSaleOrder && lead.PartnerID != nil && u.saleUC != nil {
		saleInput := saleusecase.CreateSaleOrderInput{
			PartnerID: *lead.PartnerID,
			DateOrder: time.Now().UTC(),
			CompanyID: lead.CompanyID,
			Currency:  "USD",
			Note:      fmt.Sprintf("Created automatically from won CRM Opportunity: %s (ID: %d)", lead.Name, lead.ID),
		}

		order, err := u.saleUC.CreateOrder(ctx, saleInput)
		if err != nil {
			u.logger.Warn("opportunity marked won, but sale order generation failed", "error", err)
		} else {
			generatedOrder = order
			u.logger.Info("generated draft quotation for won opportunity", "order_id", order.ID, "order_name", order.Name)
		}
	}

	u.logger.Info("marked opportunity as WON", "id", lead.ID, "revenue", lead.ExpectedRevenue)
	return lead, generatedOrder, nil
}

// MarkLeadLost marks the opportunity as Lost with explanation.
func (u *UseCase) MarkLeadLost(ctx context.Context, id int64, in MarkLostInput) (*crm.Lead, error) {
	lead, err := u.repo.GetLeadByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate reason exists
	if in.LostReasonID <= 0 {
		return nil, platformerrors.Validation("lost_reason_id is required", nil)
	}
	if _, err := u.repo.GetLostReasonByID(ctx, in.LostReasonID); err != nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("lost reason with ID %d does not exist", in.LostReasonID))
	}

	if err := lead.ActionMarkLost(in.LostReasonID, in.LostFeedback); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateLead(ctx, lead); err != nil {
		return nil, err
	}

	u.logger.Info("marked opportunity as LOST", "id", lead.ID, "reason_id", in.LostReasonID)
	return lead, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pipeline & Analytics
// ─────────────────────────────────────────────────────────────────────────────

func (u *UseCase) GetPipelineView(ctx context.Context, salespersonID *int64) ([]crm.PipelineStageData, error) {
	return u.repo.GetPipeline(ctx, salespersonID)
}

func (u *UseCase) GetCRMStats(ctx context.Context) (*crm.CRMStats, error) {
	return u.repo.GetStats(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Stages Management
// ─────────────────────────────────────────────────────────────────────────────

func (u *UseCase) CreateStage(ctx context.Context, in CreateStageInput) (*crm.Stage, error) {
	stage := &crm.Stage{
		Name:         i18n.NewTranslation(in.Name),
		Sequence:     in.Sequence,
		IsWon:        in.IsWon,
		IsClosed:     in.IsClosed,
		Fold:         in.Fold,
		Requirements: in.Requirements,
		CompanyID:    in.CompanyID,
		Active:       true,
	}
	if err := stage.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.CreateStage(ctx, stage); err != nil {
		return nil, err
	}
	return stage, nil
}

func (u *UseCase) GetStage(ctx context.Context, id int64) (*crm.Stage, error) {
	return u.repo.GetStageByID(ctx, id)
}

func (u *UseCase) UpdateStage(ctx context.Context, id int64, in UpdateStageInput) (*crm.Stage, error) {
	stage, err := u.repo.GetStageByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		stage.Name = i18n.NewTranslation(*in.Name)
	}
	if in.Sequence != nil {
		stage.Sequence = *in.Sequence
	}
	if in.IsWon != nil {
		stage.IsWon = *in.IsWon
	}
	if in.IsClosed != nil {
		stage.IsClosed = *in.IsClosed
	}
	if in.Fold != nil {
		stage.Fold = *in.Fold
	}
	if in.Requirements != nil {
		stage.Requirements = *in.Requirements
	}
	if in.CompanyID != nil {
		stage.CompanyID = in.CompanyID
	}
	if in.Active != nil {
		stage.Active = *in.Active
	}

	if err := stage.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.UpdateStage(ctx, stage); err != nil {
		return nil, err
	}
	return stage, nil
}

func (u *UseCase) DeleteStage(ctx context.Context, id int64) error {
	return u.repo.DeleteStage(ctx, id)
}

func (u *UseCase) ListStages(ctx context.Context) ([]crm.Stage, error) {
	return u.repo.ListStages(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Lost Reasons Management
// ─────────────────────────────────────────────────────────────────────────────

func (u *UseCase) CreateLostReason(ctx context.Context, in CreateLostReasonInput) (*crm.LostReason, error) {
	reason := &crm.LostReason{
		Name:   i18n.NewTranslation(in.Name),
		Active: true,
	}
	if err := reason.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.CreateLostReason(ctx, reason); err != nil {
		return nil, err
	}
	return reason, nil
}

func (u *UseCase) GetLostReason(ctx context.Context, id int64) (*crm.LostReason, error) {
	return u.repo.GetLostReasonByID(ctx, id)
}

func (u *UseCase) UpdateLostReason(ctx context.Context, id int64, in UpdateLostReasonInput) (*crm.LostReason, error) {
	reason, err := u.repo.GetLostReasonByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		reason.Name = i18n.NewTranslation(*in.Name)
	}
	if in.Active != nil {
		reason.Active = *in.Active
	}
	if err := reason.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.UpdateLostReason(ctx, reason); err != nil {
		return nil, err
	}
	return reason, nil
}

func (u *UseCase) DeleteLostReason(ctx context.Context, id int64) error {
	return u.repo.DeleteLostReason(ctx, id)
}

func (u *UseCase) ListLostReasons(ctx context.Context) ([]crm.LostReason, error) {
	return u.repo.ListLostReasons(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Tags Management
// ─────────────────────────────────────────────────────────────────────────────

func (u *UseCase) CreateTag(ctx context.Context, in CreateTagInput) (*crm.Tag, error) {
	tag := &crm.Tag{
		Name:   i18n.NewTranslation(in.Name),
		Color:  in.Color,
		Active: true,
	}
	if err := tag.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.CreateTag(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (u *UseCase) ListTags(ctx context.Context) ([]crm.Tag, error) {
	return u.repo.ListTags(ctx)
}
