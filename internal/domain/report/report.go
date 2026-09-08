package report

import (
	"time"
)

type Engine string

const (
	EngineDomain       Engine = "domain"
	EngineTaxTags      Engine = "tax_tags"
	EngineAggregation  Engine = "aggregation"
	EngineAccountCodes Engine = "account_codes"
	EngineExternal     Engine = "external"
	EngineCustom       Engine = "custom"
)

type DateScope string

const (
	DateScopeStrictRange             DateScope = "strict_range"
	DateScopeFromBeginning           DateScope = "from_beginning"
	DateScopeFromFiscalYear          DateScope = "from_fiscalyear"
	DateScopeToBeginningOfFiscalYear DateScope = "to_beginning_of_fiscalyear"
	DateScopeToBeginningOfPeriod     DateScope = "to_beginning_of_period"
	DateScopePreviousReturnPeriod    DateScope = "previous_return_period"
)

type FigureType string

const (
	FigureTypeMonetary   FigureType = "monetary"
	FigureTypePercentage FigureType = "percentage"
	FigureTypeInteger    FigureType = "integer"
	FigureTypeFloat      FigureType = "float"
	FigureTypeDate       FigureType = "date"
	FigureTypeDateTime   FigureType = "datetime"
	FigureTypeBoolean    FigureType = "boolean"
	FigureTypeString     FigureType = "string"
)

type Report struct {
	ID                    int64
	Name                  string
	Code                  string
	Sequence              int
	Active                bool
	ChartTemplate         string
	CountryID             *int64
	AvailabilityCondition string // country, coa, always
	LoadMoreLimit         int
	SearchBar             bool

	FilterDateRange        bool
	FilterShowDraft        bool
	FilterUnreconciled     bool
	FilterUnfoldAll        bool
	FilterPeriodComparison bool
	FilterGrowthComparison bool
	FilterJournals         bool
	FilterAnalytic         bool
	FilterPartner          bool

	Lines   []ReportLine
	Columns []ReportColumn

	CompanyID int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReportLine struct {
	ID             int64
	ReportID       int64
	Name           string
	Code           string // unique identifier
	ParentID       *int64
	Sequence       int
	HierarchyLevel int
	GroupBy        string
	Foldable       bool
	PrintOnNewPage bool
	ActionID       *int64
	HideIfZero     bool

	Expressions []ReportExpression
	Children    []ReportLine
}

type ReportExpression struct {
	ID           int64
	ReportLineID int64
	Label        string // balance, debit, credit, etc.
	Engine       Engine
	Formula      string
	Subformula   string
	DateScope    DateScope
	FigureType   FigureType
	GreenOnPos   bool
	BlankIfZero  bool
	Auditable    bool
}

type ReportColumn struct {
	ID              int64
	ReportID        int64
	Name            string
	ExpressionLabel string
	Sequence        int
	Sortable        bool
	FigureType      FigureType
	BlankIfZero     bool
}

// Result structure for rendering
type ReportResult struct {
	ReportID int64                  `json:"report_id"`
	Name     string                 `json:"name"`
	Options  ReportOptions          `json:"options"`
	Columns  []ReportColumnResult   `json:"columns"`
	Lines    []ReportLineResult     `json:"lines"`
	Totals   map[string]interface{} `json:"totals"`
}

type ReportColumnResult struct {
	Name            string     `json:"name"`
	ExpressionLabel string     `json:"expression_label"`
	FigureType      FigureType `json:"figure_type"`
}

type ReportLineResult struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Code        string                 `json:"code"`
	Level       int                    `json:"level"`
	ParentID    *int64                 `json:"parent_id"`
	Values      map[string]interface{} `json:"values"` // key is ExpressionLabel
	IsFolded    bool                   `json:"is_folded"`
	IsFoldable  bool                   `json:"is_foldable"`
	HasChildren bool                   `json:"has_children"`
}

type ReportOptions struct {
	DateFrom    *time.Time         `json:"date_from"`
	DateTo      *time.Time         `json:"date_to"`
	CompanyID   int64              `json:"company_id"`
	JournalIDs  []int64            `json:"journal_ids"`
	AnalyticIDs []int64            `json:"analytic_ids"`
	PartnerIDs  []int64            `json:"partner_ids"`
	Comparison  *ComparisonOptions `json:"comparison,omitempty"`
}

type ComparisonOptions struct {
	Filter string `json:"filter"` // previous_period, same_last_year
	Number int    `json:"number"` // number of periods
}
