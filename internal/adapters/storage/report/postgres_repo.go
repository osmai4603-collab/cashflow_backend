package reportstorage

import (
	"context"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/report"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// DataRepository implementation

func (r *PostgresRepo) GetBalancesByAccountPrefix(ctx context.Context, prefixes []string, options report.ReportOptions) (map[string]float64, error) {
	if len(prefixes) == 0 {
		return make(map[string]float64), nil
	}

	results := make(map[string]float64)

	// Fetch all account balances once to avoid multiple subqueries
	query := `
		SELECT a.code, SUM(l.balance) as balance
		FROM account_move_lines l
		JOIN account_accounts a ON l.account_id = a.id
		JOIN account_moves m ON l.move_id = m.id
		WHERE a.company_id = $1
	`
	args := []interface{}{options.CompanyID}

	if options.DateFrom != nil {
		args = append(args, *options.DateFrom)
		query += fmt.Sprintf(" AND m.date >= $%d", len(args))
	}
	if options.DateTo != nil {
		args = append(args, *options.DateTo)
		query += fmt.Sprintf(" AND m.date <= $%d", len(args))
	}

	// Filter by journals if provided
	if len(options.JournalIDs) > 0 {
		query += " AND m.journal_id = ANY($"+fmt.Sprint(len(args)+1)+")"
		args = append(args, options.JournalIDs)
	}

	query += " GROUP BY a.code"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accountBalances := make(map[string]float64)
	for rows.Next() {
		var code string
		var bal float64
		if err := rows.Scan(&code, &bal); err != nil {
			return nil, err
		}
		accountBalances[code] = bal
	}

	for _, prefix := range prefixes {
		var sum float64
		prefix = strings.TrimSpace(prefix)
		for code, bal := range accountBalances {
			if strings.HasPrefix(code, prefix) {
				sum += bal
			}
		}
		results[prefix] = sum
	}

	return results, nil
}

func (r *PostgresRepo) GetAnalyticBalances(ctx context.Context, analyticIDs []int64, options report.ReportOptions) (map[int64]float64, error) {
	// Implementation for analytic accounting integration
	// This would query account_analytic_line
	results := make(map[int64]float64)
	if len(analyticIDs) == 0 {
		return results, nil
	}

	query := `
		SELECT account_id, SUM(amount)
		FROM account_analytic_line
		WHERE company_id = $1 AND account_id = ANY($2)
	`
	args := []interface{}{options.CompanyID, analyticIDs}

	if options.DateFrom != nil {
		args = append(args, *options.DateFrom)
		query += fmt.Sprintf(" AND date >= $%d", len(args))
	}
	if options.DateTo != nil {
		args = append(args, *options.DateTo)
		query += fmt.Sprintf(" AND date <= $%d", len(args))
	}

	query += " GROUP BY account_id"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var amt float64
		if err := rows.Scan(&id, &amt); err != nil {
			return nil, err
		}
		results[id] = amt
	}

	return results, nil
}

// Repository implementation (Metadata)
// Note: This requires tables for reports, lines, etc.
// For now, these are placeholder implementations or can be used to store/load report configs.

func (r *PostgresRepo) GetReport(ctx context.Context, id int64) (*report.Report, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *PostgresRepo) GetReportByCode(ctx context.Context, code string) (*report.Report, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *PostgresRepo) ListReports(ctx context.Context, companyID int64) ([]*report.Report, error) {
	return nil, nil
}

func (r *PostgresRepo) SaveReport(ctx context.Context, report *report.Report) error {
	return nil
}
