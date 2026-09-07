package analyticstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/analytic"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedAccountFilterFields = map[string]string{
	"plan_id":    "plan_id",
	"name":       "name",
	"code":       "code",
	"partner_id": "partner_id",
	"company_id": "company_id",
	"active":     "active",
}

var allowedLineFilterFields = map[string]string{
	"account_id":    "account_id",
	"date_from":     "date",
	"date_to":       "date",
	"partner_id":    "partner_id",
	"user_id":       "user_id",
	"company_id":    "company_id",
	"source":        "source",
	"move_line_id":  "move_line_id",
	"name":          "name",
	"category":      "category",
}

// PostgresRepo implements analytic.Repository against a PostgreSQL database.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Plans
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreatePlan(ctx context.Context, p *analytic.AnalyticPlan) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		var rootID int64
		if p.ParentID != nil && *p.ParentID > 0 {
			var parentName, parentPath string
			if err := tx.QueryRow(ctx,
				`SELECT name, COALESCE(parent_path, '') FROM account_analytic_plan WHERE id = $1`, *p.ParentID,
			).Scan(&parentName, &parentPath); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return platformerrors.NotFound(fmt.Sprintf("parent analytic plan with ID %d not found", *p.ParentID))
				}
				return platformerrors.Internal("failed to resolve parent analytic plan", err)
			}

			// parent_path = the parent's complete name (handles nested heritage).
			p.ParentPath = parentName
			if parentPath != "" && !strings.EqualFold(parentPath, "/") {
				p.ParentPath = parentPath + " / " + parentName
			}

			if err := tx.QueryRow(ctx,
				`SELECT COALESCE(root_id, id) FROM account_analytic_plan WHERE id = $1`, *p.ParentID,
			).Scan(&rootID); err != nil {
				return platformerrors.Internal("failed to resolve root analytic plan", err)
			}
		}

		p.Active = true
		p.CompleteName = p.BuildCompleteName()

		query := `
			INSERT INTO account_analytic_plan (
				name, description, parent_id, parent_path, root_id, sequence, color,
				default_applicability, active, created_at, updated_at, created_by, updated_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW(), $10, $11)
			RETURNING id, root_id, created_at, updated_at
		`
		err := tx.QueryRow(ctx, query,
			p.Name, p.Description, p.ParentID, p.ParentPath, rootID,
			p.Sequence, p.Color, string(p.DefaultApplicability), p.Active,
			p.Audit.CreatedBy, p.Audit.UpdatedBy,
		).Scan(&p.ID, &p.RootID, &p.Audit.CreatedAt, &p.Audit.UpdatedAt)
		if err != nil {
			return platformerrors.Internal("failed to create analytic plan", err)
		}

		// A root plan references itself as its own root.
		if p.ParentID == nil || *p.ParentID <= 0 {
			p.RootID = p.ID
			if _, err := tx.Exec(ctx,
				`UPDATE account_analytic_plan SET root_id = $1 WHERE id = $1`, p.ID); err != nil {
				return platformerrors.Internal("failed to finalize root plan", err)
			}
		}

		for i := range p.Applicabilities {
			rule := &p.Applicabilities[i]
			rule.PlanID = p.ID
			if err := insertApplicability(ctx, tx, rule); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertApplicability(ctx context.Context, tx pgx.Tx, a *analytic.AnalyticApplicability) error {
	query := `
		INSERT INTO account_analytic_applicability (
			analytic_plan_id, business_domain, applicability, company_id, sequence,
			created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6, $7)
		RETURNING id
	`
	err := tx.QueryRow(ctx, query,
		a.PlanID, string(a.BusinessDomain), string(a.Applicability), a.CompanyID, a.Sequence,
		nil, nil,
	).Scan(&a.ID)
	if err != nil {
		return platformerrors.Internal("failed to create applicability rule", err)
	}
	return nil
}

func (r *PostgresRepo) GetPlanByID(ctx context.Context, id int64) (*analytic.AnalyticPlan, error) {
	query := `
		SELECT id, name, COALESCE(description, ''), parent_id, COALESCE(parent_path, ''),
		       COALESCE(root_id, 0), COALESCE(sequence, 10), COALESCE(color, 0),
		       default_applicability, active, created_at, updated_at, created_by, updated_by
		FROM account_analytic_plan
		WHERE id = $1
	`
	p := &analytic.AnalyticPlan{}
	var parentID *int64
	var defaultApp string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &parentID, &p.ParentPath,
		&p.RootID, &p.Sequence, &p.Color, &defaultApp, &p.Active,
		&p.Audit.CreatedAt, &p.Audit.UpdatedAt, &p.Audit.CreatedBy, &p.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch analytic plan", err)
	}
	p.ParentID = parentID
	p.DefaultApplicability = analytic.Applicability(defaultApp)
	p.CompleteName = p.BuildCompleteName()

	applicabilities, err := r.GetApplicabilities(ctx, p.ID)
	if err == nil {
		p.Applicabilities = applicabilities
	}
	return p, nil
}

func (r *PostgresRepo) UpdatePlan(ctx context.Context, p *analytic.AnalyticPlan) error {
	query := `
		UPDATE account_analytic_plan SET
			name = $1, description = $2, parent_id = $3, parent_path = $4,
			root_id = $5, sequence = $6, color = $7, default_applicability = $8,
			active = $9, updated_at = NOW(), updated_by = $10
		WHERE id = $11
		RETURNING updated_at
	`
	p.CompleteName = p.BuildCompleteName()
	err := r.pool.QueryRow(ctx, query,
		p.Name, p.Description, p.ParentID, p.ParentPath,
		p.RootID, p.Sequence, p.Color, string(p.DefaultApplicability),
		p.Active, p.Audit.UpdatedBy, p.ID,
	).Scan(&p.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", p.ID))
		}
		return platformerrors.Internal("failed to update analytic plan", err)
	}
	return nil
}

func (r *PostgresRepo) DeletePlan(ctx context.Context, id int64) error {
	var accountCount int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM account_analytic_account WHERE plan_id = $1`, id).Scan(&accountCount); err != nil {
		return platformerrors.Internal("failed to check analytic plan accounts", err)
	}
	if accountCount > 0 {
		return platformerrors.Conflict("cannot delete analytic plan with associated accounts")
	}

	query := `DELETE FROM account_analytic_plan WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete analytic plan", err)
	}
	if cmd.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", id))
	}
	_, _ = r.pool.Exec(ctx, `DELETE FROM ir_config_parameters WHERE key = 'analytic.project_plan' AND value = $1`, fmt.Sprintf("%d", id))
	return nil
}

func (r *PostgresRepo) ListPlans(ctx context.Context, includeInactive bool) ([]analytic.AnalyticPlan, error) {
	query := `
		SELECT id, name, COALESCE(description, ''), parent_id, COALESCE(parent_path, ''),
		       COALESCE(root_id, 0), COALESCE(sequence, 10), COALESCE(color, 0),
		       default_applicability, active, created_at, updated_at
		FROM account_analytic_plan
	`
	if !includeInactive {
		query += ` WHERE active = true`
	}
	query += ` ORDER BY sequence ASC, id ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list analytic plans", err)
	}
	defer rows.Close()

	var plans []analytic.AnalyticPlan
	for rows.Next() {
		var p analytic.AnalyticPlan
		var parentID *int64
		var defaultApp string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &parentID, &p.ParentPath,
			&p.RootID, &p.Sequence, &p.Color, &defaultApp, &p.Active,
			&p.Audit.CreatedAt, &p.Audit.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan analytic plan", err)
		}
		p.ParentID = parentID
		p.DefaultApplicability = analytic.Applicability(defaultApp)
		p.CompleteName = p.BuildCompleteName()
		plans = append(plans, p)
	}
	return plans, nil
}

func (r *PostgresRepo) GetChildrenPlans(ctx context.Context, parentID int64) ([]analytic.AnalyticPlan, error) {
	query := `
		SELECT id, name, COALESCE(description, ''), parent_id, COALESCE(parent_path, ''),
		       COALESCE(root_id, 0), COALESCE(sequence, 10), COALESCE(color, 0),
		       default_applicability, active, created_at, updated_at
		FROM account_analytic_plan
		WHERE parent_id = $1
		ORDER BY sequence ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, parentID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list child analytic plans", err)
	}
	defer rows.Close()

	var plans []analytic.AnalyticPlan
	for rows.Next() {
		var p analytic.AnalyticPlan
		var parent *int64
		var defaultApp string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &parent, &p.ParentPath,
			&p.RootID, &p.Sequence, &p.Color, &defaultApp, &p.Active,
			&p.Audit.CreatedAt, &p.Audit.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan child analytic plan", err)
		}
		p.ParentID = parent
		p.DefaultApplicability = analytic.Applicability(defaultApp)
		p.CompleteName = p.BuildCompleteName()
		plans = append(plans, p)
	}
	return plans, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Applicabilities (G2)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) SetApplicability(ctx context.Context, a *analytic.AnalyticApplicability) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM account_analytic_plan WHERE id = $1)`, a.PlanID).Scan(&exists); err != nil {
			return platformerrors.Internal("failed to check analytic plan", err)
		}
		if !exists {
			return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", a.PlanID))
		}

		query := `
			UPDATE account_analytic_applicability SET
				applicability = $3, sequence = $4, updated_at = NOW()
			WHERE analytic_plan_id = $1 AND business_domain = $2
			  AND (company_id IS NOT DISTINCT FROM $5)
			RETURNING id
		`
		err := tx.QueryRow(ctx, query,
			a.PlanID, string(a.BusinessDomain), string(a.Applicability), a.Sequence, a.CompanyID,
		).Scan(&a.ID)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.Internal("failed to update applicability rule", err)
		}
		return insertApplicability(ctx, tx, a)
	})
}

func (r *PostgresRepo) GetApplicabilities(ctx context.Context, planID int64) ([]analytic.AnalyticApplicability, error) {
	query := `
		SELECT id, analytic_plan_id, business_domain, applicability, company_id, sequence
		FROM account_analytic_applicability
		WHERE analytic_plan_id = $1
		ORDER BY sequence ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, planID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list applicability rules", err)
	}
	defer rows.Close()

	var res []analytic.AnalyticApplicability
	for rows.Next() {
		var a analytic.AnalyticApplicability
		var domain, app string
		if err := rows.Scan(&a.ID, &a.PlanID, &domain, &app, &a.CompanyID, &a.Sequence); err != nil {
			return nil, platformerrors.Internal("failed to scan applicability rule", err)
		}
		a.BusinessDomain = analytic.BusinessDomain(domain)
		a.Applicability = analytic.Applicability(app)
		res = append(res, a)
	}
	return res, nil
}

func (r *PostgresRepo) GetRelevantPlans(ctx context.Context, companyID int64, businessDomain analytic.BusinessDomain) ([]analytic.RelevantPlan, error) {
	// Root plans with at least one active account.
	planQuery := `
		SELECT p.id, p.name, p.default_applicability
		FROM account_analytic_plan p
		WHERE p.parent_id IS NULL AND p.active = true
		  AND EXISTS (SELECT 1 FROM account_analytic_account a WHERE a.plan_id = p.id AND a.active = true)
		ORDER BY p.id ASC
	`
	type planRow struct {
		ID          int64
		Name        string
		DefaultApp  analytic.Applicability
		rules       []analytic.AnalyticApplicability
		bestScore   float64
		bestRule    *analytic.AnalyticApplicability
	}
	var planRows []planRow
	rows, err := r.pool.Query(ctx, planQuery)
	if err != nil {
		return nil, platformerrors.Internal("failed to query relevant analytic plans", err)
	}
	for rows.Next() {
		var pr planRow
		var defaultApp string
		if err := rows.Scan(&pr.ID, &pr.Name, &defaultApp); err != nil {
			rows.Close()
			return nil, platformerrors.Internal("failed to scan analytic plan", err)
		}
		pr.DefaultApp = analytic.Applicability(defaultApp)
		planRows = append(planRows, pr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate analytic plans", err)
	}

	// Applicability rules for the requested business domain.
	rulesByPlan := make(map[int64][]analytic.AnalyticApplicability)
	ruleQuery := `
		SELECT analytic_plan_id, business_domain, applicability, company_id, sequence
		FROM account_analytic_applicability
		WHERE business_domain = $1
		ORDER BY sequence ASC, id ASC
	`
	ruleRows, err := r.pool.Query(ctx, ruleQuery, string(businessDomain))
	if err != nil {
		return nil, platformerrors.Internal("failed to query applicability rules", err)
	}
	defer ruleRows.Close()
	for ruleRows.Next() {
		var a analytic.AnalyticApplicability
		var domain, app string
		if err := ruleRows.Scan(&a.PlanID, &domain, &app, &a.CompanyID, &a.Sequence); err != nil {
			return nil, platformerrors.Internal("failed to scan applicability rule", err)
		}
		a.BusinessDomain = analytic.BusinessDomain(domain)
		a.Applicability = analytic.Applicability(app)
		rulesByPlan[a.PlanID] = append(rulesByPlan[a.PlanID], a)
	}

	projectPlanID, _ := r.GetProjectPlanID(ctx)

	relevant := make([]analytic.RelevantPlan, 0, len(planRows))
	for _, pr := range planRows {
		applicability := pr.DefaultApp
		bestScore := 0.0
		var bestRule *analytic.AnalyticApplicability
		for i := range rulesByPlan[pr.ID] {
			rule := &rulesByPlan[pr.ID][i]
			// Company-scoped rules only compete within their own company; other
			// contexts fall back to generic rules (doc G2 scoring).
			if rule.CompanyID != nil && *rule.CompanyID != companyID {
				continue
			}
			score := 1.0 // matches business domain
			if rule.CompanyID != nil {
				score += 0.5 // company match weighs 0.5 (G2)
			}
			if bestRule == nil || score > bestScore {
				bestRule = rule
				bestScore = score
			}
		}
		if bestRule != nil {
			applicability = bestRule.Applicability
		}
		if applicability == analytic.AppUnavailable {
			continue
		}

		columnName := "x_plan" + fmt.Sprintf("%d", pr.ID) + "_id"
		if pr.ID == projectPlanID {
			columnName = "account_id"
		}
		relevant = append(relevant, analytic.RelevantPlan{
			ID:            pr.ID,
			Name:          pr.Name,
			Applicability: applicability,
			ColumnName:    columnName,
		})
	}
	return relevant, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Accounts
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateAccount(ctx context.Context, a *analytic.AnalyticAccount) error {
	query := `
		INSERT INTO account_analytic_account (
			name, code, plan_id, root_plan_id, partner_id, color, company_id, active,
			created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), $9, $10)
		RETURNING id, created_at, updated_at
	`
	a.Active = true
	if a.RootPlanID == 0 {
		if err := r.pool.QueryRow(ctx,
			`SELECT COALESCE(root_id, id) FROM account_analytic_plan WHERE id = $1`, a.PlanID,
		).Scan(&a.RootPlanID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", a.PlanID))
			}
			return platformerrors.Internal("failed to resolve analytic plan root", err)
		}
	}
	if a.Currency == "" {
		a.Currency = "USD"
	}

	err := r.pool.QueryRow(ctx, query,
		a.Name, a.Code, a.PlanID, a.RootPlanID, a.PartnerID, a.Color, a.CompanyID, a.Active,
		a.Audit.CreatedBy, a.Audit.UpdatedBy,
	).Scan(&a.ID, &a.Audit.CreatedAt, &a.Audit.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "idx_analytic_account_code") {
			return platformerrors.Conflict(fmt.Sprintf("analytic account with code '%s' already exists", a.Code), err)
		}
		return platformerrors.Internal("failed to create analytic account", err)
	}
	return nil
}

func (r *PostgresRepo) GetAccountByID(ctx context.Context, id int64) (*analytic.AnalyticAccount, error) {
	query := `
		SELECT id, name, COALESCE(code, ''), plan_id, COALESCE(root_plan_id, 0),
		       partner_id, COALESCE(color, 0), company_id, active, created_at, updated_at,
		       created_by, updated_by
		FROM account_analytic_account
		WHERE id = $1
	`
	a := &analytic.AnalyticAccount{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.Name, &a.Code, &a.PlanID, &a.RootPlanID,
		&a.PartnerID, &a.Color, &a.CompanyID, &a.Active,
		&a.Audit.CreatedAt, &a.Audit.UpdatedAt, &a.Audit.CreatedBy, &a.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch analytic account", err)
	}

	totals, err := r.GetAccountTotals(ctx, id, nil, nil)
	if err == nil {
		a.Debit = totals.Debit
		a.Credit = totals.Credit
		a.Balance = totals.Balance
		a.Currency = totals.Currency
	}
	if a.Currency == "" {
		a.Currency = "USD"
	}
	return a, nil
}

func (r *PostgresRepo) UpdateAccount(ctx context.Context, a *analytic.AnalyticAccount) error {
	query := `
		UPDATE account_analytic_account SET
			name = $1, code = $2, plan_id = $3, root_plan_id = $4, partner_id = $5,
			color = $6, company_id = $7, active = $8, updated_at = NOW(), updated_by = $9
		WHERE id = $10
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		a.Name, a.Code, a.PlanID, a.RootPlanID, a.PartnerID,
		a.Color, a.CompanyID, a.Active, a.Audit.UpdatedBy, a.ID,
	).Scan(&a.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", a.ID))
		}
		if strings.Contains(err.Error(), "idx_analytic_account_code") {
			return platformerrors.Conflict(fmt.Sprintf("analytic account with code '%s' already exists", a.Code), err)
		}
		return platformerrors.Internal("failed to update analytic account", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteAccount(ctx context.Context, id int64) error {
	query := `DELETE FROM account_analytic_account WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Conflict("cannot delete analytic account with associated lines", err)
	}
	if cmd.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[analytic.AnalyticAccount], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedAccountFilterFields, 1)
	if err != nil {
		return pagination.PageResult[analytic.AnalyticAccount]{}, platformerrors.BadRequest("invalid filter criteria", err)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM account_analytic_account %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[analytic.AnalyticAccount]{}, platformerrors.Internal("failed to count analytic accounts", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedAccountFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()
	args = append(args, limit, offset)

	dataQuery := fmt.Sprintf(`
		SELECT id, name, COALESCE(code, ''), plan_id, COALESCE(root_plan_id, 0),
		       partner_id, COALESCE(color, 0), company_id, active, created_at, updated_at
		FROM account_analytic_account
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[analytic.AnalyticAccount]{}, platformerrors.Internal("failed to list analytic accounts", err)
	}
	defer rows.Close()

	var accounts []analytic.AnalyticAccount
	for rows.Next() {
		var a analytic.AnalyticAccount
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Code, &a.PlanID, &a.RootPlanID,
			&a.PartnerID, &a.Color, &a.CompanyID, &a.Active,
			&a.Audit.CreatedAt, &a.Audit.UpdatedAt,
		); err != nil {
			return pagination.PageResult[analytic.AnalyticAccount]{}, platformerrors.Internal("failed to scan analytic account", err)
		}
		accounts = append(accounts, a)
	}
	return pagination.NewPageResult(accounts, totalItems, page), nil
}

func (r *PostgresRepo) GetAccountTotals(ctx context.Context, accountID int64, fromDate, toDate *time.Time) (analytic.DebitCreditBalance, error) {
	var b analytic.DebitCreditBalance
	b.Currency = "USD" // company currency; default until multi-currency support (C2)
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) AS credit,
			COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0) AS debit,
			COALESCE(SUM(amount), 0) AS balance
		FROM account_analytic_line
		WHERE account_id = $1
		  AND ($2::date IS NULL OR date >= $2)
		  AND ($3::date IS NULL OR date <= $3)
	`
	err := r.pool.QueryRow(ctx, query, accountID, nullableDate(fromDate), nullableDate(toDate)).
		Scan(&b.Credit, &b.Debit, &b.Balance)
	if err != nil {
		return analytic.DebitCreditBalance{}, platformerrors.Internal("failed to compute analytic account totals", err)
	}
	return b, nil
}

func nullableDate(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

// ─────────────────────────────────────────────────────────────────────────────
// Lines
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateLine(ctx context.Context, l *analytic.AnalyticLine) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		return insertLine(ctx, tx, l)
	})
}

func insertLine(ctx context.Context, tx pgx.Tx, l *analytic.AnalyticLine) error {
	if l.Source == "" {
		l.Source = analytic.SourceManual
	}
	if l.Category == "" {
		l.Category = "other"
	}
	if l.Currency == "" {
		l.Currency = "USD"
	}
	if l.Date.IsZero() {
		l.Date = time.Now().UTC()
	}

	var distJSON []byte
	if len(l.Distribution) > 0 {
		b, err := json.Marshal(l.Distribution)
		if err != nil {
			return platformerrors.Internal("failed to encode analytic distribution", err)
		}
		distJSON = b
	}

	query := `
		INSERT INTO account_analytic_line (
			name, date, amount, unit_amount, product_uom_id, partner_id, user_id,
			company_id, currency_code, category, account_id, analytic_distribution,
			move_line_id, general_account_id, source,
			created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb,
		          $13, $14, $15, NOW(), NOW(), $16, $17)
		RETURNING id, created_at, updated_at
	`
	err := tx.QueryRow(ctx, query,
		l.Name, l.Date.Format("2006-01-02"), l.Amount, l.UnitAmount, l.ProductUoMID, l.PartnerID, l.UserID,
		l.CompanyID, l.Currency, l.Category, l.AccountID, distJSON,
		l.MoveLineID, l.GeneralAccountID, string(l.Source),
		l.Audit.CreatedBy, l.Audit.UpdatedBy,
	).Scan(&l.ID, &l.Audit.CreatedAt, &l.Audit.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "account_analytic_account") {
			return platformerrors.BadRequest("invalid analytic account reference", err)
		}
		return platformerrors.Internal("failed to create analytic line", err)
	}
	return nil
}

func (r *PostgresRepo) CreateLines(ctx context.Context, lines []analytic.AnalyticLine) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		for i := range lines {
			if err := insertLine(ctx, tx, &lines[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostgresRepo) GetLineByID(ctx context.Context, id int64) (*analytic.AnalyticLine, error) {
	query := `
		SELECT id, name, date, amount, unit_amount, product_uom_id, partner_id, user_id,
		       company_id, currency_code, category, account_id, analytic_distribution,
		       move_line_id, general_account_id, source, created_at, updated_at, created_by, updated_by
		FROM account_analytic_line
		WHERE id = $1
	`
	l, err := scanLine(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("analytic line with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch analytic line", err)
	}
	return l, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLine(row rowScanner) (*analytic.AnalyticLine, error) {
	l := &analytic.AnalyticLine{}
	var distJSON []byte
	var source string
	err := row.Scan(
		&l.ID, &l.Name, &l.Date, &l.Amount, &l.UnitAmount, &l.ProductUoMID, &l.PartnerID, &l.UserID,
		&l.CompanyID, &l.Currency, &l.Category, &l.AccountID, &distJSON,
		&l.MoveLineID, &l.GeneralAccountID, &source, &l.Audit.CreatedAt, &l.Audit.UpdatedAt,
		&l.Audit.CreatedBy, &l.Audit.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}
	if len(distJSON) > 0 {
		var dist analytic.AnalyticDistribution
		if json.Unmarshal(distJSON, &dist) == nil {
			l.Distribution = dist
		}
	}
	l.Source = analytic.LineSource(source)
	return l, nil
}

func (r *PostgresRepo) UpdateLine(ctx context.Context, l *analytic.AnalyticLine) error {
	if l.Source == "" {
		l.Source = analytic.SourceManual
	}
	if l.Category == "" {
		l.Category = "other"
	}
	if l.Date.IsZero() {
		l.Date = time.Now().UTC()
	}

	var distJSON []byte
	if len(l.Distribution) > 0 {
		b, err := json.Marshal(l.Distribution)
		if err != nil {
			return platformerrors.Internal("failed to encode analytic distribution", err)
		}
		distJSON = b
	}

	query := `
		UPDATE account_analytic_line SET
			name = $1, date = $2, amount = $3, unit_amount = $4, product_uom_id = $5,
			partner_id = $6, user_id = $7, company_id = $8, currency_code = $9,
			category = $10, account_id = $11, analytic_distribution = $12::jsonb,
			move_line_id = $13, general_account_id = $14, source = $15,
			updated_at = NOW(), updated_by = $16
		WHERE id = $17
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		l.Name, l.Date.Format("2006-01-02"), l.Amount, l.UnitAmount, l.ProductUoMID,
		l.PartnerID, l.UserID, l.CompanyID, l.Currency,
		l.Category, l.AccountID, distJSON,
		l.MoveLineID, l.GeneralAccountID, string(l.Source),
		l.Audit.UpdatedBy, l.ID,
	).Scan(&l.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("analytic line with ID %d not found", l.ID))
		}
		return platformerrors.Internal("failed to update analytic line", err)
	}
	return nil
}

func (r *PostgresRepo) ListLines(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[analytic.AnalyticLine], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedLineFilterFields, 1)
	if err != nil {
		return pagination.PageResult[analytic.AnalyticLine]{}, platformerrors.BadRequest("invalid filter criteria", err)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM account_analytic_line %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[analytic.AnalyticLine]{}, platformerrors.Internal("failed to count analytic lines", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedLineFilterFields[page.SortBy]; ok {
			if col == "date" {
				col = "date"
			}
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()
	args = append(args, limit, offset)

	dataQuery := fmt.Sprintf(`
		SELECT id, name, date, amount, unit_amount, product_uom_id, partner_id, user_id,
		       company_id, currency_code, category, account_id, analytic_distribution,
		       move_line_id, general_account_id, source, created_at, updated_at
		FROM account_analytic_line
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[analytic.AnalyticLine]{}, platformerrors.Internal("failed to list analytic lines", err)
	}
	defer rows.Close()

	var lines []analytic.AnalyticLine
	for rows.Next() {
		l, err := scanLine(rows)
		if err != nil {
			return pagination.PageResult[analytic.AnalyticLine]{}, platformerrors.Internal("failed to scan analytic line", err)
		}
		lines = append(lines, *l)
	}
	return pagination.NewPageResult(lines, totalItems, page), nil
}

func (r *PostgresRepo) ListLinesByMoveLine(ctx context.Context, moveLineID int64) ([]analytic.AnalyticLine, error) {
	query := `
		SELECT id, name, date, amount, unit_amount, product_uom_id, partner_id, user_id,
		       company_id, currency_code, category, account_id, analytic_distribution,
		       move_line_id, general_account_id, source, created_at, updated_at
		FROM account_analytic_line
		WHERE move_line_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query, moveLineID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list analytic lines by move line", err)
	}
	defer rows.Close()

	var lines []analytic.AnalyticLine
	for rows.Next() {
		l, err := scanLine(rows)
		if err != nil {
			return nil, platformerrors.Internal("failed to scan analytic line", err)
		}
		lines = append(lines, *l)
	}
	return lines, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Distribution Models (G7)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateDistributionModel(ctx context.Context, m *analytic.DistributionModel) error {
	distJSON, err := json.Marshal(m.Distribution)
	if err != nil {
		return platformerrors.Internal("failed to encode analytic distribution", err)
	}

	query := `
		INSERT INTO account_analytic_distribution_model (
			sequence, partner_id, partner_category_id, company_id, analytic_distribution,
			active, created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6, NOW(), NOW(), $7, $8)
		RETURNING id, created_at, updated_at
	`
	m.Active = true
	err = r.pool.QueryRow(ctx, query,
		m.Sequence, m.PartnerID, m.PartnerCategoryID, m.CompanyID, distJSON, m.Active,
		m.Audit.CreatedBy, m.Audit.UpdatedBy,
	).Scan(&m.ID, &m.Audit.CreatedAt, &m.Audit.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create distribution model", err)
	}
	return nil
}

func (r *PostgresRepo) GetDistributionModelByID(ctx context.Context, id int64) (*analytic.DistributionModel, error) {
	query := `
		SELECT id, sequence, partner_id, partner_category_id, company_id, analytic_distribution,
		       active, created_at, updated_at
		FROM account_analytic_distribution_model
		WHERE id = $1
	`
	m := &analytic.DistributionModel{}
	var distJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Sequence, &m.PartnerID, &m.PartnerCategoryID, &m.CompanyID, &distJSON,
		&m.Active, &m.Audit.CreatedAt, &m.Audit.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("analytic distribution model with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch distribution model", err)
	}
	if len(distJSON) > 0 {
		var dist analytic.AnalyticDistribution
		if json.Unmarshal(distJSON, &dist) == nil {
			m.Distribution = dist
		}
	}
	return m, nil
}

func (r *PostgresRepo) UpdateDistributionModel(ctx context.Context, m *analytic.DistributionModel) error {
	distJSON, err := json.Marshal(m.Distribution)
	if err != nil {
		return platformerrors.Internal("failed to encode analytic distribution", err)
	}

	query := `
		UPDATE account_analytic_distribution_model SET
			sequence = $1, partner_id = $2, partner_category_id = $3, company_id = $4,
			analytic_distribution = $5::jsonb, active = $6, updated_at = NOW(), updated_by = $7
		WHERE id = $8
		RETURNING updated_at
	`
	err = r.pool.QueryRow(ctx, query,
		m.Sequence, m.PartnerID, m.PartnerCategoryID, m.CompanyID, distJSON, m.Active,
		m.Audit.UpdatedBy, m.ID,
	).Scan(&m.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("analytic distribution model with ID %d not found", m.ID))
		}
		return platformerrors.Internal("failed to update distribution model", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteDistributionModel(ctx context.Context, id int64) error {
	query := `DELETE FROM account_analytic_distribution_model WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete distribution model", err)
	}
	if cmd.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("analytic distribution model with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListDistributionModels(ctx context.Context) ([]analytic.DistributionModel, error) {
	query := `
		SELECT id, sequence, partner_id, partner_category_id, company_id, analytic_distribution,
		       active, created_at, updated_at
		FROM account_analytic_distribution_model
		WHERE active = true
		ORDER BY sequence ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list distribution models", err)
	}
	defer rows.Close()

	var models []analytic.DistributionModel
	for rows.Next() {
		var m analytic.DistributionModel
		var distJSON []byte
		if err := rows.Scan(
			&m.ID, &m.Sequence, &m.PartnerID, &m.PartnerCategoryID, &m.CompanyID, &distJSON,
			&m.Active, &m.Audit.CreatedAt, &m.Audit.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan distribution model", err)
		}
		if len(distJSON) > 0 {
			var dist analytic.AnalyticDistribution
			if json.Unmarshal(distJSON, &dist) == nil {
				m.Distribution = dist
			}
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *PostgresRepo) MatchDistribution(ctx context.Context, partnerID, partnerCategoryID *int64, companyID int64) (analytic.AnalyticDistribution, error) {
	query := `
		SELECT analytic_distribution
		FROM account_analytic_distribution_model
		WHERE active = true
		  AND (company_id IS NULL OR company_id = $1)
		  AND (partner_id IS NULL OR partner_id = $2)
		  AND (partner_category_id IS NULL OR partner_category_id = $3)
		ORDER BY (partner_id IS NOT NULL) DESC, (partner_category_id IS NOT NULL) DESC, sequence ASC
		LIMIT 1
	`
	var distJSON []byte
	err := r.pool.QueryRow(ctx, query, companyID, partnerID, partnerCategoryID).Scan(&distJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, platformerrors.Internal("failed to match distribution model", err)
	}
	if len(distJSON) == 0 {
		return nil, nil
	}
	var dist analytic.AnalyticDistribution
	if err := json.Unmarshal(distJSON, &dist); err != nil {
		return nil, platformerrors.Internal("failed to decode matched distribution", err)
	}
	return dist, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Project Plan (G8)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) GetProjectPlanID(ctx context.Context) (int64, error) {
	var planID int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(NULLIF(value, '')::bigint, 0) FROM ir_config_parameters WHERE key = 'analytic.project_plan'`,
	).Scan(&planID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, platformerrors.Internal("failed to read analytic.project_plan config", err)
	}
	return planID, nil
}

func (r *PostgresRepo) SetProjectPlanID(ctx context.Context, planID int64) error {
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM account_analytic_plan WHERE id = $1)`, planID).Scan(&exists); err != nil {
		return platformerrors.Internal("failed to check analytic plan", err)
	}
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", planID))
	}

	query := `
		INSERT INTO ir_config_parameters (key, value, company_id)
		VALUES ('analytic.project_plan', $1, NULL)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, fmt.Sprintf("%d", planID))
	if err != nil {
		return platformerrors.Internal("failed to set analytic.project_plan config", err)
	}
	return nil
}