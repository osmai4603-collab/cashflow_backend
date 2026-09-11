package crm

import (
	"math"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// LeadType distinguishes between an inbound raw inquiry and a qualified sales deal.
type LeadType string

const (
	LeadTypeLead        LeadType = "lead"
	LeadTypeOpportunity LeadType = "opportunity"
)

// Priority defines the urgency or importance rating of a deal (0 to 3 stars in Odoo).
type Priority string

const (
	PriorityLow      Priority = "0"
	PriorityNormal   Priority = "1"
	PriorityHigh     Priority = "2"
	PriorityVeryHigh Priority = "3"
)

// Lead represents a potential customer interaction or qualified sales pipeline deal (crm.lead in Odoo).
type Lead struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"` // Subject/Title
	Type            LeadType   `json:"type"` // lead or opportunity
	PartnerID       *int64     `json:"partner_id,omitempty"`
	PartnerName     string     `json:"partner_name,omitempty"`
	ContactName     string     `json:"contact_name,omitempty"`
	EmailFrom       string     `json:"email_from,omitempty"`
	Phone           string     `json:"phone,omitempty"`
	StageID         int64      `json:"stage_id"`
	SalespersonID   *int64     `json:"salesperson_id,omitempty"`
	ExpectedRevenue float64    `json:"expected_revenue"`
	ProratedRevenue float64    `json:"prorated_revenue"`
	Probability     float64    `json:"probability"` // 0.00 to 100.00
	Source          string     `json:"source,omitempty"`
	Priority        Priority   `json:"priority"`
	LostReasonID    *int64     `json:"lost_reason_id,omitempty"`
	LostFeedback    string     `json:"lost_feedback,omitempty"`
	DateDeadline    *time.Time `json:"date_deadline,omitempty"`
	DateClosed      *time.Time `json:"date_closed,omitempty"`
	DateConversion  *time.Time `json:"date_conversion,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	CompanyID       *int64     `json:"company_id,omitempty"`
	Active          bool       `json:"active"`
	Tags            []Tag      `json:"tags,omitempty"`
	TagIDs          []int64    `json:"tag_ids,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *int64     `json:"created_by,omitempty"`
	UpdatedBy       *int64     `json:"updated_by,omitempty"`
}

// ComputeProratedRevenue calculates expected revenue weighted by close probability:
// ProratedRevenue = ExpectedRevenue * (Probability / 100.0)
func (l *Lead) ComputeProratedRevenue() {
	if l.ExpectedRevenue < 0 {
		l.ExpectedRevenue = 0
	}
	if l.Probability < 0 {
		l.Probability = 0
	} else if l.Probability > 100 {
		l.Probability = 100
	}
	l.ProratedRevenue = roundTo4(l.ExpectedRevenue * (l.Probability / 100.0))
}

// ActionConvert transforms an unverified Lead into a qualified sales Opportunity.
func (l *Lead) ActionConvert(newStageID int64, partnerID *int64) error {
	if l.Type == LeadTypeOpportunity {
		return platformerrors.Conflict("record is already an opportunity")
	}
	if newStageID <= 0 {
		return platformerrors.Validation("target stage is required for conversion", nil)
	}

	now := time.Now().UTC()
	l.Type = LeadTypeOpportunity
	l.StageID = newStageID
	if partnerID != nil && *partnerID > 0 {
		l.PartnerID = partnerID
	}
	l.DateConversion = &now
	l.ComputeProratedRevenue()
	return nil
}

// ActionMarkWon marks the opportunity as successfully closed/won.
func (l *Lead) ActionMarkWon(wonStageID int64) error {
	if wonStageID <= 0 {
		return platformerrors.Validation("won stage id is required", nil)
	}
	now := time.Now().UTC()
	l.StageID = wonStageID
	l.Probability = 100.0
	l.DateClosed = &now
	l.LostReasonID = nil
	l.LostFeedback = ""
	l.Active = true
	l.ComputeProratedRevenue()
	return nil
}

// ActionMarkLost closes the opportunity as lost and records the reason.
func (l *Lead) ActionMarkLost(reasonID int64, feedback string) error {
	if reasonID <= 0 {
		return platformerrors.Validation("lost reason id is required", nil)
	}
	now := time.Now().UTC()
	l.LostReasonID = &reasonID
	l.LostFeedback = strings.TrimSpace(feedback)
	l.Probability = 0.0
	l.Active = false
	l.DateClosed = &now
	l.ComputeProratedRevenue()
	return nil
}

// Validate verifies integrity and business invariants of the Lead/Opportunity record.
func (l *Lead) Validate() error {
	l.Name = strings.TrimSpace(l.Name)
	if l.Name == "" {
		return platformerrors.Validation("lead/opportunity name is required", nil)
	}
	if len(l.Name) > 255 {
		return platformerrors.Validation("name cannot exceed 255 characters", nil)
	}

	if l.Type != LeadTypeLead && l.Type != LeadTypeOpportunity {
		return platformerrors.Validation("type must be 'lead' or 'opportunity'", nil)
	}

	if l.StageID <= 0 {
		return platformerrors.Validation("stage_id is required", nil)
	}

	if l.ExpectedRevenue < 0 {
		return platformerrors.Validation("expected_revenue cannot be negative", nil)
	}

	if l.Probability < 0 || l.Probability > 100 {
		return platformerrors.Validation("probability must be between 0.0 and 100.0", nil)
	}

	if l.Priority == "" {
		l.Priority = PriorityNormal
	} else if l.Priority != PriorityLow && l.Priority != PriorityNormal && l.Priority != PriorityHigh && l.Priority != PriorityVeryHigh {
		return platformerrors.Validation("priority must be '0', '1', '2', or '3'", nil)
	}

	l.ComputeProratedRevenue()
	return nil
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}

// ThreadModel satisfies activity.Threadable.
func (l *Lead) ThreadModel() string { return "crm.lead" }

// ThreadID satisfies activity.Threadable.
func (l *Lead) ThreadID() int64 { return l.ID }

// ThreadCompanyID satisfies activity.Threadable.
func (l *Lead) ThreadCompanyID() int64 {
	if l.CompanyID != nil && *l.CompanyID > 0 {
		return *l.CompanyID
	}
	return 1
}
