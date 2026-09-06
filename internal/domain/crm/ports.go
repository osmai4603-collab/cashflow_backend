package crm

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// PipelineStageData encapsulates an aggregation bucket for a pipeline stage in Kanban view.
type PipelineStageData struct {
	Stage                Stage   `json:"stage"`
	TotalOpportunities   int     `json:"total_opportunities"`
	TotalExpectedRevenue float64 `json:"total_expected_revenue"`
	TotalProratedRevenue float64 `json:"total_prorated_revenue"`
	Opportunities        []Lead  `json:"opportunities"`
}

// LostReasonStat captures analytics regarding opportunities lost to a specific cause.
type LostReasonStat struct {
	ReasonID    int64   `json:"reason_id"`
	ReasonName  string  `json:"reason_name"`
	Count       int     `json:"count"`
	LostRevenue float64 `json:"lost_revenue"`
}

// StageStat represents stage distribution metrics.
type StageStat struct {
	StageID         int64   `json:"stage_id"`
	StageName       string  `json:"stage_name"`
	Count           int     `json:"count"`
	ExpectedRevenue float64 `json:"expected_revenue"`
	ProratedRevenue float64 `json:"prorated_revenue"`
}

// CRMStats provides high-level business metrics across the entire sales funnel.
type CRMStats struct {
	TotalLeads           int              `json:"total_leads"`
	TotalOpportunities   int              `json:"total_opportunities"`
	WonCount             int              `json:"won_count"`
	LostCount            int              `json:"lost_count"`
	WinRate              float64          `json:"win_rate"` // won / (won + lost) * 100
	ConversionRate       float64          `json:"conversion_rate"`
	TotalExpectedRevenue float64          `json:"total_expected_revenue"`
	TotalProratedRevenue float64          `json:"total_prorated_revenue"`
	TotalWonRevenue      float64          `json:"total_won_revenue"`
	AvgDealSize          float64          `json:"avg_deal_size"`
	Stages               []StageStat      `json:"stages"`
	TopLostReasons       []LostReasonStat `json:"top_lost_reasons"`
}

// Repository defines the persistent storage operations required by the CRM domain.
type Repository interface {
	// Leads & Opportunities
	CreateLead(ctx context.Context, lead *Lead) error
	GetLeadByID(ctx context.Context, id int64) (*Lead, error)
	UpdateLead(ctx context.Context, lead *Lead) error
	DeleteLead(ctx context.Context, id int64) error
	ListLeads(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Lead], error)

	// Stages
	CreateStage(ctx context.Context, stage *Stage) error
	GetStageByID(ctx context.Context, id int64) (*Stage, error)
	UpdateStage(ctx context.Context, stage *Stage) error
	DeleteStage(ctx context.Context, id int64) error
	ListStages(ctx context.Context) ([]Stage, error)
	GetWonStage(ctx context.Context) (*Stage, error)
	GetInitialStage(ctx context.Context) (*Stage, error)

	// Lost Reasons
	CreateLostReason(ctx context.Context, reason *LostReason) error
	GetLostReasonByID(ctx context.Context, id int64) (*LostReason, error)
	UpdateLostReason(ctx context.Context, reason *LostReason) error
	DeleteLostReason(ctx context.Context, id int64) error
	ListLostReasons(ctx context.Context) ([]LostReason, error)

	// Tags
	CreateTag(ctx context.Context, tag *Tag) error
	GetTagByID(ctx context.Context, id int64) (*Tag, error)
	ListTags(ctx context.Context) ([]Tag, error)
	AssignTags(ctx context.Context, leadID int64, tagIDs []int64) error
	GetTagsByLeadID(ctx context.Context, leadID int64) ([]Tag, error)

	// Pipeline & Analytics
	GetPipeline(ctx context.Context, salespersonID *int64) ([]PipelineStageData, error)
	GetStats(ctx context.Context) (*CRMStats, error)
}
