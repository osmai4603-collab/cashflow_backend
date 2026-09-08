package project

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// Project is an operational project linked optionally to an analytic account.
type Project struct {
	ID                 int64      `json:"id"`
	Name               i18n.TranslationString     `json:"name"`
	Description        string     `json:"description,omitempty"`
	PartnerID          *int64     `json:"partner_id,omitempty"`
	ManagerID          *int64     `json:"manager_id,omitempty"`
	StageID            *int64     `json:"stage_id,omitempty"`
	DateStart          *time.Time `json:"date_start,omitempty"`
	DateEnd            *time.Time `json:"date_end,omitempty"`
	Active             bool       `json:"active"`
	AllowMilestones    bool       `json:"allow_milestones"`
	AllowSubtasks      bool       `json:"allow_subtasks"`
	AllowDependencies  bool       `json:"allow_dependencies"`
	AnalyticAccountID  *int64     `json:"analytic_account_id,omitempty"`
	CompanyID          int64      `json:"company_id"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (p *Project) Validate() error {
	if len(p.Name) == 0 {
		return platformerrors.Validation("project name is required", nil)
	}
	if p.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	if p.DateStart != nil && p.DateEnd != nil && p.DateEnd.Before(*p.DateStart) {
		return platformerrors.Validation("date_end cannot be before date_start", nil)
	}
	return nil
}
