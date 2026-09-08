package crmhttp

import (
	"time"

	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/sale"
	crmusecase "cashflow_backend/internal/usecase/crm"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateLeadRequest struct {
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

func (r CreateLeadRequest) ToInput() crmusecase.CreateLeadInput {
	return crmusecase.CreateLeadInput{
		Name:            r.Name,
		Type:            r.Type,
		PartnerID:       r.PartnerID,
		PartnerName:     r.PartnerName,
		ContactName:     r.ContactName,
		EmailFrom:       r.EmailFrom,
		Phone:           r.Phone,
		StageID:         r.StageID,
		SalespersonID:   r.SalespersonID,
		ExpectedRevenue: r.ExpectedRevenue,
		Probability:     r.Probability,
		Source:          r.Source,
		Priority:        r.Priority,
		DateDeadline:    r.DateDeadline,
		Notes:           r.Notes,
		CompanyID:       r.CompanyID,
		TagIDs:          r.TagIDs,
	}
}

type UpdateLeadRequest struct {
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

func (r UpdateLeadRequest) ToInput() crmusecase.UpdateLeadInput {
	return crmusecase.UpdateLeadInput{
		Name:            r.Name,
		Type:            r.Type,
		PartnerID:       r.PartnerID,
		PartnerName:     r.PartnerName,
		ContactName:     r.ContactName,
		EmailFrom:       r.EmailFrom,
		Phone:           r.Phone,
		StageID:         r.StageID,
		SalespersonID:   r.SalespersonID,
		ExpectedRevenue: r.ExpectedRevenue,
		Probability:     r.Probability,
		Source:          r.Source,
		Priority:        r.Priority,
		DateDeadline:    r.DateDeadline,
		Notes:           r.Notes,
		CompanyID:       r.CompanyID,
		Active:          r.Active,
		TagIDs:          r.TagIDs,
	}
}

type ConvertLeadRequest struct {
	StageID       *int64 `json:"stage_id"`
	PartnerID     *int64 `json:"partner_id"`
	CreatePartner *bool  `json:"create_partner"`
}

func (r ConvertLeadRequest) ToInput() crmusecase.ConvertLeadInput {
	createPartner := true
	if r.CreatePartner != nil {
		createPartner = *r.CreatePartner
	}
	return crmusecase.ConvertLeadInput{
		StageID:       r.StageID,
		PartnerID:     r.PartnerID,
		CreatePartner: createPartner,
	}
}

type MarkWonRequest struct {
	StageID         *int64 `json:"stage_id"`
	CreateSaleOrder bool   `json:"create_sale_order"`
}

func (r MarkWonRequest) ToInput() crmusecase.MarkWonInput {
	return crmusecase.MarkWonInput{
		StageID:         r.StageID,
		CreateSaleOrder: r.CreateSaleOrder,
	}
}

type MarkLostRequest struct {
	LostReasonID int64  `json:"lost_reason_id"`
	LostFeedback string `json:"lost_feedback"`
}

func (r MarkLostRequest) ToInput() crmusecase.MarkLostInput {
	return crmusecase.MarkLostInput{
		LostReasonID: r.LostReasonID,
		LostFeedback: r.LostFeedback,
	}
}

type CreateStageRequest struct {
	Name         string `json:"name"`
	Sequence     int    `json:"sequence"`
	IsWon        bool   `json:"is_won"`
	IsClosed     bool   `json:"is_closed"`
	Fold         bool   `json:"fold"`
	Requirements string `json:"requirements"`
	CompanyID    *int64 `json:"company_id"`
}

type UpdateStageRequest struct {
	Name         *string `json:"name"`
	Sequence     *int    `json:"sequence"`
	IsWon        *bool   `json:"is_won"`
	IsClosed     *bool   `json:"is_closed"`
	Fold         *bool   `json:"fold"`
	Requirements *string `json:"requirements"`
	CompanyID    *int64  `json:"company_id"`
	Active       *bool   `json:"active"`
}

type CreateLostReasonRequest struct {
	Name string `json:"name"`
}

type UpdateLostReasonRequest struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

type CreateTagRequest struct {
	Name  string `json:"name"`
	Color int    `json:"color"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Response DTOs
// ─────────────────────────────────────────────────────────────────────────────

type LeadResponse struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	Type            string         `json:"type"`
	PartnerID       *int64         `json:"partner_id,omitempty"`
	PartnerName     string         `json:"partner_name,omitempty"`
	ContactName     string         `json:"contact_name,omitempty"`
	EmailFrom       string         `json:"email_from,omitempty"`
	Phone           string         `json:"phone,omitempty"`
	StageID         int64          `json:"stage_id"`
	SalespersonID   *int64         `json:"salesperson_id,omitempty"`
	ExpectedRevenue float64        `json:"expected_revenue"`
	ProratedRevenue float64        `json:"prorated_revenue"`
	Probability     float64        `json:"probability"`
	Source          string         `json:"source,omitempty"`
	Priority        string         `json:"priority"`
	LostReasonID    *int64         `json:"lost_reason_id,omitempty"`
	LostFeedback    string         `json:"lost_feedback,omitempty"`
	DateDeadline    *time.Time     `json:"date_deadline,omitempty"`
	DateClosed      *time.Time     `json:"date_closed,omitempty"`
	DateConversion  *time.Time     `json:"date_conversion,omitempty"`
	Notes           string         `json:"notes,omitempty"`
	CompanyID       *int64         `json:"company_id,omitempty"`
	Active          bool           `json:"active"`
	Tags            []TagResponse  `json:"tags,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func ToLeadResponse(l *crm.Lead) LeadResponse {
	var tags []TagResponse
	for _, t := range l.Tags {
		tags = append(tags, ToTagResponse(&t))
	}
	return LeadResponse{
		ID:              l.ID,
		Name:            l.Name,
		Type:            string(l.Type),
		PartnerID:       l.PartnerID,
		PartnerName:     l.PartnerName,
		ContactName:     l.ContactName,
		EmailFrom:       l.EmailFrom,
		Phone:           l.Phone,
		StageID:         l.StageID,
		SalespersonID:   l.SalespersonID,
		ExpectedRevenue: l.ExpectedRevenue,
		ProratedRevenue: l.ProratedRevenue,
		Probability:     l.Probability,
		Source:          l.Source,
		Priority:        string(l.Priority),
		LostReasonID:    l.LostReasonID,
		LostFeedback:    l.LostFeedback,
		DateDeadline:    l.DateDeadline,
		DateClosed:      l.DateClosed,
		DateConversion:  l.DateConversion,
		Notes:           l.Notes,
		CompanyID:       l.CompanyID,
		Active:          l.Active,
		Tags:            tags,
		CreatedAt:       l.CreatedAt,
		UpdatedAt:       l.UpdatedAt,
	}
}

type StageResponse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Sequence     int       `json:"sequence"`
	IsWon        bool      `json:"is_won"`
	IsClosed     bool      `json:"is_closed"`
	Fold         bool      `json:"fold"`
	Requirements string    `json:"requirements,omitempty"`
	CompanyID    *int64    `json:"company_id,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func ToStageResponse(s *crm.Stage) StageResponse {
	return StageResponse{
		ID:           s.ID,
		Name:         string(s.Name),
		Sequence:     s.Sequence,
		IsWon:        s.IsWon,
		IsClosed:     s.IsClosed,
		Fold:         s.Fold,
		Requirements: s.Requirements,
		CompanyID:    s.CompanyID,
		Active:       s.Active,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

type LostReasonResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToLostReasonResponse(r *crm.LostReason) LostReasonResponse {
	return LostReasonResponse{
		ID:        r.ID,
		Name:      string(r.Name),
		Active:    r.Active,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

type TagResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color int    `json:"color"`
}

func ToTagResponse(t *crm.Tag) TagResponse {
	return TagResponse{
		ID:    t.ID,
		Name:  string(t.Name),
		Color: t.Color,
	}
}

type MarkWonResponse struct {
	Lead      LeadResponse    `json:"lead"`
	SaleOrder *sale.SaleOrder `json:"sale_order,omitempty"`
}

type PipelineStageResponse struct {
	Stage                StageResponse  `json:"stage"`
	TotalOpportunities   int            `json:"total_opportunities"`
	TotalExpectedRevenue float64        `json:"total_expected_revenue"`
	TotalProratedRevenue float64        `json:"total_prorated_revenue"`
	Opportunities        []LeadResponse `json:"opportunities"`
}

func ToPipelineResponse(data []crm.PipelineStageData) []PipelineStageResponse {
	res := make([]PipelineStageResponse, len(data))
	for i, d := range data {
		opps := make([]LeadResponse, len(d.Opportunities))
		for j, opp := range d.Opportunities {
			opps[j] = ToLeadResponse(&opp)
		}
		res[i] = PipelineStageResponse{
			Stage:                ToStageResponse(&d.Stage),
			TotalOpportunities:   d.TotalOpportunities,
			TotalExpectedRevenue: d.TotalExpectedRevenue,
			TotalProratedRevenue: d.TotalProratedRevenue,
			Opportunities:        opps,
		}
	}
	return res
}
