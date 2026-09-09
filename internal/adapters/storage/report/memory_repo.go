package reportstorage

import (
	"context"
	"sort"
	"strings"
	"sync"

	"cashflow_backend/internal/domain/report"
)

// MemoryRepo provides a thread-safe, in-memory implementation of report.Repository
// and report.DataRepository for development and tests.
type MemoryRepo struct {
	mu      sync.RWMutex
	reports map[int64]*report.Report
	nextID  int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		reports: make(map[int64]*report.Report),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// report.Repository
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) GetReport(ctx context.Context, id int64) (*report.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rep, exists := r.reports[id]
	if !exists {
		return nil, nil
	}
	clone := *rep
	return &clone, nil
}

func (r *MemoryRepo) GetReportByCode(ctx context.Context, code string) (*report.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rep := range r.reports {
		if rep.Code == code {
			clone := *rep
			return &clone, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) ListReports(ctx context.Context, companyID int64) ([]*report.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*report.Report
	for _, rep := range r.reports {
		if rep.CompanyID != companyID {
			continue
		}
		clone := *rep
		result = append(result, &clone)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) SaveReport(ctx context.Context, rep *report.Report) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rep.ID == 0 {
		r.nextID++
		rep.ID = r.nextID
	}
	clone := *rep
	r.reports[rep.ID] = &clone
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// report.DataRepository
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) GetBalancesByAccountPrefix(ctx context.Context, prefixes []string, options report.ReportOptions) (map[string]float64, error) {
	result := make(map[string]float64, len(prefixes))
	for _, prefix := range prefixes {
		result[strings.TrimSpace(prefix)] = 0
	}
	return result, nil
}

func (r *MemoryRepo) GetAnalyticBalances(ctx context.Context, analyticIDs []int64, options report.ReportOptions) (map[int64]float64, error) {
	return make(map[int64]float64), nil
}