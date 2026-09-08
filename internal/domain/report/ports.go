package report

import "context"

type Repository interface {
	GetReport(ctx context.Context, id int64) (*Report, error)
	GetReportByCode(ctx context.Context, code string) (*Report, error)
	ListReports(ctx context.Context, companyID int64) ([]*Report, error)
	SaveReport(ctx context.Context, report *Report) error
}

type DataRepository interface {
	GetBalancesByAccountPrefix(ctx context.Context, prefixes []string, options ReportOptions) (map[string]float64, error)
	GetAnalyticBalances(ctx context.Context, analyticIDs []int64, options ReportOptions) (map[int64]float64, error)
}

type ReportGenerator interface {
	Generate(ctx context.Context, report *Report, options ReportOptions) (*ReportResult, error)
}
