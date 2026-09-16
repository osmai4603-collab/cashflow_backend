package crmstorage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedLeadFilterFields = map[string]string{
	"type":           "type",
	"stage_id":       "stage_id",
	"salesperson_id": "salesperson_id",
	"partner_id":     "partner_id",
	"priority":       "priority",
	"active":         "active",
	"name":           "name",
}

// PostgresRepo implements crm.Repository against a PostgreSQL database.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Leads & Opportunities
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateLead(ctx context.Context, lead *crm.Lead) error {
	lead.ComputeProratedRevenue()
	if lead.Type == "" {
		lead.Type = crm.LeadTypeLead
	}
	if lead.Priority == "" {
		lead.Priority = crm.PriorityNormal
	}

	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO crm_leads (
				name, type, partner_id, partner_name, contact_name, email_from, phone,
				stage_id, salesperson_id, expected_revenue, prorated_revenue, probability,
				source, priority, lost_reason_id, lost_feedback, date_deadline, date_closed,
				date_conversion, notes, company_id, active, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12,
				$13, $14, $15, $16, $17, $18,
				$19, $20, $21, $22, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`

		err := tx.QueryRow(ctx, query,
			lead.Name, string(lead.Type), lead.PartnerID, lead.PartnerName, lead.ContactName, lead.EmailFrom, lead.Phone,
			lead.StageID, lead.SalespersonID, lead.ExpectedRevenue, lead.ProratedRevenue, lead.Probability,
			lead.Source, string(lead.Priority), lead.LostReasonID, lead.LostFeedback, lead.DateDeadline, lead.DateClosed,
			lead.DateConversion, lead.Notes, lead.CompanyID, lead.Active,
		).Scan(&lead.ID, &lead.CreatedAt, &lead.UpdatedAt)

		if err != nil {
			return platformerrors.Internal("failed to create lead", err)
		}

		if len(lead.TagIDs) > 0 {
			for _, tagID := range lead.TagIDs {
				tagQ := `INSERT INTO crm_lead_tags (lead_id, tag_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING`
				if _, err := tx.Exec(ctx, tagQ, lead.ID, tagID); err != nil {
					return platformerrors.Internal("failed to associate tag with lead", err)
				}
			}
		}

		return nil
	})
}

func (r *PostgresRepo) GetLeadByID(ctx context.Context, id int64) (*crm.Lead, error) {
	query := `
		SELECT 
			id, name, type, partner_id,
			COALESCE(partner_name, '') AS partner_name,
			COALESCE(contact_name, '') AS contact_name,
			COALESCE(email_from, '') AS email_from,
			COALESCE(phone, '') AS phone,
			stage_id, salesperson_id, expected_revenue, prorated_revenue, probability,
			COALESCE(source, '') AS source, priority, lost_reason_id,
			COALESCE(lost_feedback, '') AS lost_feedback, date_deadline, date_closed,
			date_conversion, COALESCE(notes, '') AS notes, company_id, active, created_at, updated_at, created_by, updated_by
		FROM crm_leads
		WHERE id = $1
	`

	lead := &crm.Lead{}
	var leadType, priority string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&lead.ID, &lead.Name, &leadType, &lead.PartnerID, &lead.PartnerName, &lead.ContactName, &lead.EmailFrom, &lead.Phone,
		&lead.StageID, &lead.SalespersonID, &lead.ExpectedRevenue, &lead.ProratedRevenue, &lead.Probability,
		&lead.Source, &priority, &lead.LostReasonID, &lead.LostFeedback, &lead.DateDeadline, &lead.DateClosed,
		&lead.DateConversion, &lead.Notes, &lead.CompanyID, &lead.Active, &lead.CreatedAt, &lead.UpdatedAt,
		&lead.CreatedBy, &lead.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("lead/opportunity with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch lead", err)
	}

	lead.Type = crm.LeadType(leadType)
	lead.Priority = crm.Priority(priority)

	tags, err := r.GetTagsByLeadID(ctx, lead.ID)
	if err == nil {
		lead.Tags = tags
		for _, t := range tags {
			lead.TagIDs = append(lead.TagIDs, t.ID)
		}
	}

	return lead, nil
}

func (r *PostgresRepo) UpdateLead(ctx context.Context, lead *crm.Lead) error {
	lead.ComputeProratedRevenue()

	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE crm_leads SET
				name = $1, type = $2, partner_id = $3, partner_name = $4, contact_name = $5,
				email_from = $6, phone = $7, stage_id = $8, salesperson_id = $9, expected_revenue = $10,
				prorated_revenue = $11, probability = $12, source = $13, priority = $14, lost_reason_id = $15,
				lost_feedback = $16, date_deadline = $17, date_closed = $18, date_conversion = $19,
				notes = $20, company_id = $21, active = $22, updated_at = NOW()
			WHERE id = $23
			RETURNING updated_at
		`

		err := tx.QueryRow(ctx, query,
			lead.Name, string(lead.Type), lead.PartnerID, lead.PartnerName, lead.ContactName,
			lead.EmailFrom, lead.Phone, lead.StageID, lead.SalespersonID, lead.ExpectedRevenue,
			lead.ProratedRevenue, lead.Probability, lead.Source, string(lead.Priority), lead.LostReasonID,
			lead.LostFeedback, lead.DateDeadline, lead.DateClosed, lead.DateConversion,
			lead.Notes, lead.CompanyID, lead.Active, lead.ID,
		).Scan(&lead.UpdatedAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("lead/opportunity with ID %d not found", lead.ID))
			}
			return platformerrors.Internal("failed to update lead", err)
		}

		if lead.TagIDs != nil {
			delQ := `DELETE FROM crm_lead_tags WHERE lead_id = $1`
			if _, err := tx.Exec(ctx, delQ, lead.ID); err != nil {
				return platformerrors.Internal("failed to clear lead tags", err)
			}
			for _, tagID := range lead.TagIDs {
				tagQ := `INSERT INTO crm_lead_tags (lead_id, tag_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING`
				if _, err := tx.Exec(ctx, tagQ, lead.ID, tagID); err != nil {
					return platformerrors.Internal("failed to associate tag with lead", err)
				}
			}
		}

		return nil
	})
}

func (r *PostgresRepo) DeleteLead(ctx context.Context, id int64) error {
	query := `DELETE FROM crm_leads WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete lead", err)
	}
	if cmd.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("lead/opportunity with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListLeads(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[crm.Lead], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedLeadFilterFields, 1)
	if err != nil {
		return pagination.PageResult[crm.Lead]{}, platformerrors.BadRequest("invalid filter criteria", err)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM crm_leads %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[crm.Lead]{}, platformerrors.Internal("failed to count leads", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedLeadFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()
	args = append(args, limit, offset)

	dataQuery := fmt.Sprintf(`
		SELECT 
			id, name, type, partner_id,
			COALESCE(partner_name, '') AS partner_name,
			COALESCE(contact_name, '') AS contact_name,
			COALESCE(email_from, '') AS email_from,
			COALESCE(phone, '') AS phone,
			stage_id, salesperson_id, expected_revenue, prorated_revenue, probability,
			COALESCE(source, '') AS source, priority, lost_reason_id,
			COALESCE(lost_feedback, '') AS lost_feedback, date_deadline, date_closed,
			date_conversion, COALESCE(notes, '') AS notes, company_id, active, created_at, updated_at, created_by, updated_by
		FROM crm_leads
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[crm.Lead]{}, platformerrors.Internal("failed to list leads", err)
	}
	defer rows.Close()

	var leads []crm.Lead
	for rows.Next() {
		var l crm.Lead
		var leadType, priority string

		if err := rows.Scan(
			&l.ID, &l.Name, &leadType, &l.PartnerID, &l.PartnerName, &l.ContactName, &l.EmailFrom, &l.Phone,
			&l.StageID, &l.SalespersonID, &l.ExpectedRevenue, &l.ProratedRevenue, &l.Probability,
			&l.Source, &priority, &l.LostReasonID, &l.LostFeedback, &l.DateDeadline, &l.DateClosed,
			&l.DateConversion, &l.Notes, &l.CompanyID, &l.Active, &l.CreatedAt, &l.UpdatedAt,
			&l.CreatedBy, &l.UpdatedBy,
		); err != nil {
			return pagination.PageResult[crm.Lead]{}, platformerrors.Internal("failed to scan lead", err)
		}
		l.Type = crm.LeadType(leadType)
		l.Priority = crm.Priority(priority)
		leads = append(leads, l)
	}

	return pagination.NewPageResult(leads, totalItems, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Stages
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateStage(ctx context.Context, stage *crm.Stage) error {
	query := `
		INSERT INTO crm_stages (name, sequence, is_won, is_closed, fold, requirements, company_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		stage.Name, stage.Sequence, stage.IsWon, stage.IsClosed, stage.Fold, stage.Requirements, stage.CompanyID,
	).Scan(&stage.ID, &stage.CreatedAt, &stage.UpdatedAt)
}

func (r *PostgresRepo) GetStageByID(ctx context.Context, id int64) (*crm.Stage, error) {
	query := `
		SELECT id, name, sequence, is_won, is_closed, COALESCE(fold, false) AS fold,
		       COALESCE(requirements, '') AS requirements, company_id, active, created_at, updated_at, created_by, updated_by
		FROM crm_stages
		WHERE id = $1
	`
	s := &crm.Stage{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Sequence, &s.IsWon, &s.IsClosed, &s.Fold, &s.Requirements,
		&s.CompanyID, &s.Active, &s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stage with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch stage", err)
	}
	return s, nil
}

func (r *PostgresRepo) UpdateStage(ctx context.Context, stage *crm.Stage) error {
	query := `
		UPDATE crm_stages SET
			name = $1, sequence = $2, is_won = $3, is_closed = $4, fold = $5,
			requirements = $6, company_id = $7, active = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		stage.Name, stage.Sequence, stage.IsWon, stage.IsClosed, stage.Fold,
		stage.Requirements, stage.CompanyID, stage.Active, stage.ID,
	).Scan(&stage.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("stage with ID %d not found", stage.ID))
		}
		return platformerrors.Internal("failed to update stage", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteStage(ctx context.Context, id int64) error {
	var count int
	checkQ := `SELECT COUNT(*) FROM crm_leads WHERE stage_id = $1`
	if err := r.pool.QueryRow(ctx, checkQ, id).Scan(&count); err == nil && count > 0 {
		return platformerrors.Conflict("cannot delete stage with associated leads/opportunities")
	}

	query := `DELETE FROM crm_stages WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete stage", err)
	}
	if cmd.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("stage with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListStages(ctx context.Context) ([]crm.Stage, error) {
	query := `
		SELECT id, name, sequence, is_won, is_closed, COALESCE(fold, false) AS fold,
		       COALESCE(requirements, '') AS requirements, company_id, active, created_at, updated_at
		FROM crm_stages
		WHERE active = true
		ORDER BY sequence ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list stages", err)
	}
	defer rows.Close()

	var stages []crm.Stage
	for rows.Next() {
		var s crm.Stage
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Sequence, &s.IsWon, &s.IsClosed, &s.Fold,
			&s.Requirements, &s.CompanyID, &s.Active, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan stage", err)
		}
		stages = append(stages, s)
	}
	return stages, nil
}

func (r *PostgresRepo) GetWonStage(ctx context.Context) (*crm.Stage, error) {
	query := `
		SELECT id, name, sequence, is_won, is_closed, COALESCE(fold, false) AS fold,
		       COALESCE(requirements, '') AS requirements, company_id, active, created_at, updated_at
		FROM crm_stages
		WHERE active = true AND is_won = true
		ORDER BY sequence ASC LIMIT 1
	`
	s := &crm.Stage{}
	err := r.pool.QueryRow(ctx, query).Scan(
		&s.ID, &s.Name, &s.Sequence, &s.IsWon, &s.IsClosed, &s.Fold,
		&s.Requirements, &s.CompanyID, &s.Active, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("won stage not configured")
		}
		return nil, platformerrors.Internal("failed to fetch won stage", err)
	}
	return s, nil
}

func (r *PostgresRepo) GetInitialStage(ctx context.Context) (*crm.Stage, error) {
	query := `
		SELECT id, name, sequence, is_won, is_closed, COALESCE(fold, false) AS fold,
		       COALESCE(requirements, '') AS requirements, company_id, active, created_at, updated_at
		FROM crm_stages
		WHERE active = true AND is_won = false AND is_closed = false
		ORDER BY sequence ASC LIMIT 1
	`
	s := &crm.Stage{}
	err := r.pool.QueryRow(ctx, query).Scan(
		&s.ID, &s.Name, &s.Sequence, &s.IsWon, &s.IsClosed, &s.Fold,
		&s.Requirements, &s.CompanyID, &s.Active, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("initial stage not configured")
		}
		return nil, platformerrors.Internal("failed to fetch initial stage", err)
	}
	return s, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Lost Reasons
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateLostReason(ctx context.Context, reason *crm.LostReason) error {
	query := `
		INSERT INTO crm_lost_reasons (name, active, created_at, updated_at)
		VALUES ($1, true, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, reason.Name).Scan(&reason.ID, &reason.CreatedAt, &reason.UpdatedAt)
}

func (r *PostgresRepo) GetLostReasonByID(ctx context.Context, id int64) (*crm.LostReason, error) {
	query := `SELECT id, name, active, created_at, updated_at FROM crm_lost_reasons WHERE id = $1`
	reason := &crm.LostReason{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&reason.ID, &reason.Name, &reason.Active, &reason.CreatedAt, &reason.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("lost reason with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch lost reason", err)
	}
	return reason, nil
}

func (r *PostgresRepo) UpdateLostReason(ctx context.Context, reason *crm.LostReason) error {
	query := `UPDATE crm_lost_reasons SET name = $1, active = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`
	err := r.pool.QueryRow(ctx, query, reason.Name, reason.Active, reason.ID).Scan(&reason.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("lost reason with ID %d not found", reason.ID))
		}
		return platformerrors.Internal("failed to update lost reason", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteLostReason(ctx context.Context, id int64) error {
	query := `DELETE FROM crm_lost_reasons WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete lost reason", err)
	}
	if cmd.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("lost reason with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListLostReasons(ctx context.Context) ([]crm.LostReason, error) {
	query := `SELECT id, name, active, created_at, updated_at FROM crm_lost_reasons WHERE active = true ORDER BY id ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list lost reasons", err)
	}
	defer rows.Close()

	var reasons []crm.LostReason
	for rows.Next() {
		var lr crm.LostReason
		if err := rows.Scan(&lr.ID, &lr.Name, &lr.Active, &lr.CreatedAt, &lr.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan lost reason", err)
		}
		reasons = append(reasons, lr)
	}
	return reasons, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tags
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateTag(ctx context.Context, tag *crm.Tag) error {
	query := `
		INSERT INTO crm_tags (name, color, active, created_at, updated_at)
		VALUES ($1, $2, true, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query, tag.Name, tag.Color).Scan(&tag.ID, &tag.CreatedAt, &tag.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "crm_tags_name_key") {
			return platformerrors.Conflict(fmt.Sprintf("tag '%s' already exists", tag.Name), err)
		}
		return platformerrors.Internal("failed to create tag", err)
	}
	return nil
}

func (r *PostgresRepo) GetTagByID(ctx context.Context, id int64) (*crm.Tag, error) {
	query := `SELECT id, name, color, active, created_at, updated_at FROM crm_tags WHERE id = $1`
	tag := &crm.Tag{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Active, &tag.CreatedAt, &tag.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("tag with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch tag", err)
	}
	return tag, nil
}

func (r *PostgresRepo) ListTags(ctx context.Context) ([]crm.Tag, error) {
	query := `SELECT id, name, color, active, created_at, updated_at FROM crm_tags WHERE active = true ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list tags", err)
	}
	defer rows.Close()

	var tags []crm.Tag
	for rows.Next() {
		var t crm.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Active, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan tag", err)
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (r *PostgresRepo) AssignTags(ctx context.Context, leadID int64, tagIDs []int64) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		delQ := `DELETE FROM crm_lead_tags WHERE lead_id = $1`
		if _, err := tx.Exec(ctx, delQ, leadID); err != nil {
			return platformerrors.Internal("failed to clear tags", err)
		}
		for _, tid := range tagIDs {
			insQ := `INSERT INTO crm_lead_tags (lead_id, tag_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING`
			if _, err := tx.Exec(ctx, insQ, leadID, tid); err != nil {
				return platformerrors.Internal("failed to insert lead tag", err)
			}
		}
		return nil
	})
}

func (r *PostgresRepo) GetTagsByLeadID(ctx context.Context, leadID int64) ([]crm.Tag, error) {
	query := `
		SELECT t.id, t.name, t.color, t.active, t.created_at, t.updated_at
		FROM crm_tags t
		INNER JOIN crm_lead_tags lt ON t.id = lt.tag_id
		WHERE lt.lead_id = $1 AND t.active = true
		ORDER BY t.name ASC
	`
	rows, err := r.pool.Query(ctx, query, leadID)
	if err != nil {
		return nil, platformerrors.Internal("failed to get lead tags", err)
	}
	defer rows.Close()

	var tags []crm.Tag
	for rows.Next() {
		var t crm.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Active, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan tag", err)
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pipeline & Analytics
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) GetPipeline(ctx context.Context, salespersonID *int64) ([]crm.PipelineStageData, error) {
	stages, err := r.ListStages(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]crm.PipelineStageData, len(stages))
	for i, stage := range stages {
		baseQ := `
			SELECT 
				id, name, type, partner_id,
				COALESCE(partner_name, '') AS partner_name,
				COALESCE(contact_name, '') AS contact_name,
				COALESCE(email_from, '') AS email_from,
				COALESCE(phone, '') AS phone,
				stage_id, salesperson_id, expected_revenue, prorated_revenue, probability,
				COALESCE(source, '') AS source, priority, lost_reason_id,
				COALESCE(lost_feedback, '') AS lost_feedback, date_deadline, date_closed,
				date_conversion, COALESCE(notes, '') AS notes, company_id, active, created_at, updated_at
			FROM crm_leads
			WHERE stage_id = $1 AND type = 'opportunity' AND active = true
		`
		args := []any{stage.ID}
		if salespersonID != nil {
			baseQ += ` AND salesperson_id = $2`
			args = append(args, *salespersonID)
		}
		baseQ += ` ORDER BY id DESC`

		rows, err := r.pool.Query(ctx, baseQ, args...)
		if err != nil {
			return nil, platformerrors.Internal("failed to query pipeline stage leads", err)
		}

		var opps []crm.Lead
		var totalExpected, totalProrated float64
		for rows.Next() {
			var l crm.Lead
			var leadType, priority string
			if err := rows.Scan(
				&l.ID, &l.Name, &leadType, &l.PartnerID, &l.PartnerName, &l.ContactName, &l.EmailFrom, &l.Phone,
				&l.StageID, &l.SalespersonID, &l.ExpectedRevenue, &l.ProratedRevenue, &l.Probability,
				&l.Source, &priority, &l.LostReasonID, &l.LostFeedback, &l.DateDeadline, &l.DateClosed,
				&l.DateConversion, &l.Notes, &l.CompanyID, &l.Active, &l.CreatedAt, &l.UpdatedAt,
			); err != nil {
				rows.Close()
				return nil, platformerrors.Internal("failed to scan pipeline lead", err)
			}
			l.Type = crm.LeadType(leadType)
			l.Priority = crm.Priority(priority)
			totalExpected += l.ExpectedRevenue
			totalProrated += l.ProratedRevenue
			opps = append(opps, l)
		}
		rows.Close()

		result[i] = crm.PipelineStageData{
			Stage:                stage,
			TotalOpportunities:   len(opps),
			TotalExpectedRevenue: math.Round(totalExpected*10000) / 10000,
			TotalProratedRevenue: math.Round(totalProrated*10000) / 10000,
			Opportunities:        opps,
		}
	}

	return result, nil
}

func (r *PostgresRepo) GetStats(ctx context.Context) (*crm.CRMStats, error) {
	stats := &crm.CRMStats{
		Stages:         make([]crm.StageStat, 0),
		TopLostReasons: make([]crm.LostReasonStat, 0),
	}

	summaryQ := `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE type = 'lead'), 0) AS total_leads,
			COALESCE(COUNT(*) FILTER (WHERE type = 'opportunity'), 0) AS total_opps,
			COALESCE(COUNT(*) FILTER (WHERE probability >= 100.0), 0) AS won_count,
			COALESCE(COUNT(*) FILTER (WHERE lost_reason_id IS NOT NULL OR (active = false AND probability < 100.0)), 0) AS lost_count,
			COALESCE(SUM(expected_revenue) FILTER (WHERE type = 'opportunity' AND active = true), 0) AS total_expected,
			COALESCE(SUM(prorated_revenue) FILTER (WHERE type = 'opportunity' AND active = true), 0) AS total_prorated,
			COALESCE(SUM(expected_revenue) FILTER (WHERE probability >= 100.0), 0) AS total_won
		FROM crm_leads
	`
	if err := r.pool.QueryRow(ctx, summaryQ).Scan(
		&stats.TotalLeads,
		&stats.TotalOpportunities,
		&stats.WonCount,
		&stats.LostCount,
		&stats.TotalExpectedRevenue,
		&stats.TotalProratedRevenue,
		&stats.TotalWonRevenue,
	); err != nil {
		return nil, platformerrors.Internal("failed to fetch CRM summary stats", err)
	}

	totalDecided := stats.WonCount + stats.LostCount
	if totalDecided > 0 {
		stats.WinRate = math.Round((float64(stats.WonCount)/float64(totalDecided)*100.0)*100) / 100
	}

	totalEntities := stats.TotalLeads + stats.TotalOpportunities
	if totalEntities > 0 {
		stats.ConversionRate = math.Round((float64(stats.TotalOpportunities)/float64(totalEntities)*100.0)*100) / 100
	}

	if stats.TotalOpportunities > 0 {
		stats.AvgDealSize = math.Round((stats.TotalExpectedRevenue/float64(stats.TotalOpportunities))*10000) / 10000
	}

	// Stage breakdown
	stageQ := `
		SELECT 
			s.id, s.name,
			COALESCE(COUNT(l.id), 0) AS opp_count,
			COALESCE(SUM(l.expected_revenue), 0) AS expected_rev,
			COALESCE(SUM(l.prorated_revenue), 0) AS prorated_rev
		FROM crm_stages s
		LEFT JOIN crm_leads l ON s.id = l.stage_id AND l.active = true AND l.type = 'opportunity'
		WHERE s.active = true
		GROUP BY s.id, s.name, s.sequence
		ORDER BY s.sequence ASC
	`
	rows, err := r.pool.Query(ctx, stageQ)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch stage stats", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ss crm.StageStat
		if err := rows.Scan(&ss.StageID, &ss.StageName, &ss.Count, &ss.ExpectedRevenue, &ss.ProratedRevenue); err != nil {
			return nil, platformerrors.Internal("failed to scan stage stat", err)
		}
		stats.Stages = append(stats.Stages, ss)
	}

	// Lost reasons breakdown
	lostQ := `
		SELECT 
			lr.id, lr.name,
			COUNT(l.id) AS lost_count,
			COALESCE(SUM(l.expected_revenue), 0) AS lost_rev
		FROM crm_lost_reasons lr
		INNER JOIN crm_leads l ON lr.id = l.lost_reason_id
		GROUP BY lr.id, lr.name
		ORDER BY lost_count DESC
	`
	lrows, err := r.pool.Query(ctx, lostQ)
	if err == nil {
		defer lrows.Close()
		for lrows.Next() {
			var lrs crm.LostReasonStat
			if err := lrows.Scan(&lrs.ReasonID, &lrs.ReasonName, &lrs.Count, &lrs.LostRevenue); err == nil {
				stats.TopLostReasons = append(stats.TopLostReasons, lrs)
			}
		}
	}

	return stats, nil
}
