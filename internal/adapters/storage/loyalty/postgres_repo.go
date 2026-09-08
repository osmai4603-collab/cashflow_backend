package loyaltystorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/platform/audit"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedProgramFilterFields = map[string]string{
	"id":           "id",
	"name":         "name",
	"active":       "active",
	"program_type": "program_type",
	"applies_on":   "applies_on",
	"trigger":      "trigger",
	"company_id":   "company_id",
}

var allowedCardFilterFields = map[string]string{
	"id":         "id",
	"program_id": "program_id",
	"code":       "code",
	"partner_id": "partner_id",
	"active":     "active",
	"order_id":   "order_id",
}

// PostgresRepo implements loyalty.Repository using PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Programs (with nested rules, rewards, mails and pricelists)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateProgram(ctx context.Context, p *loyalty.LoyaltyProgram) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO loyalty_programs (
				name, active, sequence, company_id, currency, program_type,
				date_from, date_to, limit_usage, max_usage, applies_on, trigger,
				portal_visible, portal_point_name, is_nominative, is_payment_program, sale_ok,
				created_at, updated_at, created_by, updated_by
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
				$13, $14, $15, $16, $17, NOW(), NOW(), $18, $18
			) RETURNING id, created_at, updated_at
		`
		userID := audit.UserIDFromContext(ctx)
		err := tx.QueryRow(ctx, query,
			p.Name, p.Active, p.Sequence, p.CompanyID, p.Currency, string(p.ProgramType),
			p.DateFrom, p.DateTo, p.LimitUsage, p.MaxUsage, string(p.AppliesOn), string(p.Trigger),
			p.PortalVisible, p.PortalPointName, p.IsNominative, p.IsPaymentProgram, p.SaleOK,
			userID,
		).Scan(&p.ID, &p.Audit.CreatedAt, &p.Audit.UpdatedAt)
		if err != nil {
			return platformerrors.Internal("failed to create loyalty program", err)
		}
		p.Audit.CreatedBy = userID
		p.Audit.UpdatedBy = userID

		if err := r.replaceProgramChildren(ctx, tx, p.ID, p); err != nil {
			return err
		}
		return nil
	})
}

func (r *PostgresRepo) UpdateProgram(ctx context.Context, p *loyalty.LoyaltyProgram) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE loyalty_programs SET
				name = $1, active = $2, sequence = $3, currency = $4, program_type = $5,
				date_from = $6, date_to = $7, limit_usage = $8, max_usage = $9,
				applies_on = $10, trigger = $11, portal_visible = $12,
				portal_point_name = $13, is_nominative = $14, is_payment_program = $15,
				sale_ok = $16, updated_at = NOW(), updated_by = $17
			WHERE id = $18
		`
		userID := audit.UserIDFromContext(ctx)
		tag, err := tx.Exec(ctx, query,
			p.Name, p.Active, p.Sequence, p.Currency, string(p.ProgramType),
			p.DateFrom, p.DateTo, p.LimitUsage, p.MaxUsage,
			string(p.AppliesOn), string(p.Trigger), p.PortalVisible,
			p.PortalPointName, p.IsNominative, p.IsPaymentProgram,
			p.SaleOK, userID, p.ID,
		)
		if err != nil {
			return platformerrors.Internal("failed to update loyalty program", err)
		}
		if tag.RowsAffected() == 0 {
			return platformerrors.NotFound(fmt.Sprintf("loyalty program with id %d not found", p.ID))
		}
		p.Audit.UpdatedBy = userID

		if err := r.replaceProgramChildren(ctx, tx, p.ID, p); err != nil {
			return err
		}
		return nil
	})
}

// replaceProgramChildren re-syncs pricelists, rules, rewards and mails for a program.
func (r *PostgresRepo) replaceProgramChildren(ctx context.Context, tx pgx.Tx, programID int64, p *loyalty.LoyaltyProgram) error {
	if _, err := tx.Exec(ctx, `DELETE FROM loyalty_program_pricelists WHERE program_id = $1`, programID); err != nil {
		return platformerrors.Internal("failed to reset program pricelists", err)
	}
	if p.PricelistIDs != nil {
		for _, plID := range p.PricelistIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO loyalty_program_pricelists (program_id, pricelist_id) VALUES ($1, $2)`, programID, plID); err != nil {
				return platformerrors.Internal("failed to link program pricelist", err)
			}
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM loyalty_rules WHERE program_id = $1`, programID); err != nil {
		return platformerrors.Internal("failed to reset program rules", err)
	}
	for i := range p.Rules {
		p.Rules[i].ProgramID = programID
		if err := r.insertRuleTx(ctx, tx, &p.Rules[i]); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM loyalty_rewards WHERE program_id = $1`, programID); err != nil {
		return platformerrors.Internal("failed to reset program rewards", err)
	}
	for i := range p.Rewards {
		p.Rewards[i].ProgramID = programID
		if err := r.insertRewardTx(ctx, tx, &p.Rewards[i]); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM loyalty_mails WHERE program_id = $1`, programID); err != nil {
		return platformerrors.Internal("failed to reset program mails", err)
	}
	for i := range p.Mails {
		p.Mails[i].ProgramID = programID
		if err := r.insertMailTx(ctx, tx, &p.Mails[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepo) GetProgramByID(ctx context.Context, id int64) (*loyalty.LoyaltyProgram, error) {
	query := `
		SELECT
			id, name, active, sequence, company_id, currency, program_type,
			date_from, date_to, limit_usage, max_usage, applies_on, trigger,
			portal_visible, portal_point_name, is_nominative, is_payment_program, sale_ok,
			created_at, updated_at, created_by, updated_by
		FROM loyalty_programs WHERE id = $1
	`
	p := &loyalty.LoyaltyProgram{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Active, &p.Sequence, &p.CompanyID, &p.Currency, (*string)(&p.ProgramType),
		&p.DateFrom, &p.DateTo, &p.LimitUsage, &p.MaxUsage, (*string)(&p.AppliesOn), (*string)(&p.Trigger),
		&p.PortalVisible, &p.PortalPointName, &p.IsNominative, &p.IsPaymentProgram, &p.SaleOK,
		&p.Audit.CreatedAt, &p.Audit.UpdatedAt, &p.Audit.CreatedBy, &p.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("loyalty program with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get loyalty program", err)
	}

	if p.PricelistIDs, err = r.listPricelistIDs(ctx, id); err != nil {
		return nil, err
	}
	if p.Rules, err = r.listRulesByProgram(ctx, id); err != nil {
		return nil, err
	}
	if p.Rewards, err = r.listRewardsByProgram(ctx, id); err != nil {
		return nil, err
	}
	p.Mails, err = r.listMailsByProgram(ctx, id)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PostgresRepo) ListPrograms(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[loyalty.LoyaltyProgram], error) {
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		if f == nil {
			f = filter.NewFilter()
		}
		f.Add("company_id", filter.OpEqual, *companyID)
	}
	if f == nil {
		f = filter.NewFilter()
	}

	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedProgramFilterFields, 1)
	if err != nil {
		return pagination.PageResult[loyalty.LoyaltyProgram]{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM loyalty_programs %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[loyalty.LoyaltyProgram]{}, platformerrors.Internal("failed to count loyalty programs", err)
	}
	if totalItems == 0 {
		return pagination.NewPageResult([]loyalty.LoyaltyProgram{}, 0, page), nil
	}

	dataQuery := fmt.Sprintf(`
		SELECT
			id, name, active, sequence, company_id, currency, program_type,
			date_from, date_to, limit_usage, max_usage, applies_on, trigger,
			portal_visible, portal_point_name, is_nominative, is_payment_program, sale_ok,
			created_at, updated_at, created_by, updated_by
		FROM loyalty_programs
		%s
		ORDER BY sequence, id
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[loyalty.LoyaltyProgram]{}, platformerrors.Internal("failed to list loyalty programs", err)
	}
	defer rows.Close()

	var programs []loyalty.LoyaltyProgram
	for rows.Next() {
		var p loyalty.LoyaltyProgram
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Active, &p.Sequence, &p.CompanyID, &p.Currency, (*string)(&p.ProgramType),
			&p.DateFrom, &p.DateTo, &p.LimitUsage, &p.MaxUsage, (*string)(&p.AppliesOn), (*string)(&p.Trigger),
			&p.PortalVisible, &p.PortalPointName, &p.IsNominative, &p.IsPaymentProgram, &p.SaleOK,
			&p.Audit.CreatedAt, &p.Audit.UpdatedAt, &p.Audit.CreatedBy, &p.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[loyalty.LoyaltyProgram]{}, platformerrors.Internal("failed to scan loyalty program", err)
		}
		programs = append(programs, p)
	}

	if err := r.decorateProgramCounts(ctx, programs); err != nil {
		return pagination.PageResult[loyalty.LoyaltyProgram]{}, err
	}
	return pagination.NewPageResult(programs, totalItems, page), nil
}

func (r *PostgresRepo) DeleteProgram(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM loyalty_programs WHERE id = $1`, id)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			return platformerrors.Conflict("loyalty program is referenced by existing records and cannot be deleted", err)
		}
		return platformerrors.Internal("failed to delete loyalty program", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty program with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) decorateProgramCounts(ctx context.Context, programs []loyalty.LoyaltyProgram) error {
	if len(programs) == 0 {
		return nil
	}
	for i := range programs {
		if err := r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM loyalty_cards WHERE program_id = $1`, programs[i].ID).Scan(&programs[i].CouponCount); err != nil {
			return platformerrors.Internal("failed to count program coupons", err)
		}
		if err := r.pool.QueryRow(ctx,
			`SELECT COALESCE(COUNT(*), 0) FROM sale_orders WHERE $1 = ANY(applied_coupon_ids)`, programs[i].ID).Scan(&programs[i].TotalOrderCount); err != nil {
			return platformerrors.Internal("failed to count program orders", err)
		}
	}
	return nil
}

func (r *PostgresRepo) listPricelistIDs(ctx context.Context, programID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT pricelist_id FROM loyalty_program_pricelists WHERE program_id = $1 ORDER BY pricelist_id`, programID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list program pricelists", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, platformerrors.Internal("failed to scan program pricelist", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Rules
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateRule(ctx context.Context, rule *loyalty.LoyaltyRule) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		return r.insertRuleTx(ctx, tx, rule)
	})
}

func (r *PostgresRepo) insertRuleTx(ctx context.Context, getter interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, rule *loyalty.LoyaltyRule) error {
	productIDs := rule.ProductIDs
	if productIDs == nil {
		productIDs = []int64{}
	}
	query := `
		INSERT INTO loyalty_rules (
			active, program_id, company_id, product_ids, product_category_id,
			product_tag_id, product_domain, reward_point_amount, reward_point_split,
			reward_point_mode, minimum_qty, minimum_amount, minimum_amount_tax_mode,
			mode, code, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			NOW(), NOW(), $16, $16
		) RETURNING id
	`
	userID := audit.UserIDFromContext(ctx)
	if err := getter.QueryRow(ctx, query,
		rule.Active, rule.ProgramID, rule.CompanyID, productIDs, rule.ProductCategoryID,
		rule.ProductTagID, rule.ProductDomain, rule.RewardPointAmount, rule.RewardPointSplit,
		string(rule.RewardPointMode), rule.MinimumQty, rule.MinimumAmount, string(rule.MinimumAmountTaxMode),
		string(rule.Mode), rule.Code, userID,
	).Scan(&rule.ID); err != nil {
		return platformerrors.Internal("failed to create loyalty rule", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateRule(ctx context.Context, rule *loyalty.LoyaltyRule) error {
	query := `
		UPDATE loyalty_rules SET
			active = $1, company_id = $2, product_ids = $3,
			product_category_id = $4, product_tag_id = $5, product_domain = $6,
			reward_point_amount = $7, reward_point_split = $8, reward_point_mode = $9,
			minimum_qty = $10, minimum_amount = $11, minimum_amount_tax_mode = $12,
			mode = $13, code = $14, updated_at = NOW(), updated_by = $15
		WHERE id = $16
	`
	productIDs := rule.ProductIDs
	if productIDs == nil {
		productIDs = []int64{}
	}
	tag, err := r.pool.Exec(ctx, query,
		rule.Active, rule.CompanyID, productIDs,
		rule.ProductCategoryID, rule.ProductTagID, rule.ProductDomain,
		rule.RewardPointAmount, rule.RewardPointSplit, string(rule.RewardPointMode),
		rule.MinimumQty, rule.MinimumAmount, string(rule.MinimumAmountTaxMode),
		string(rule.Mode), rule.Code, audit.UserIDFromContext(ctx), rule.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update loyalty rule", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty rule with id %d not found", rule.ID))
	}
	return nil
}

func (r *PostgresRepo) DeleteRule(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM loyalty_rules WHERE id = $1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete loyalty rule", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty rule with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) listRulesByProgram(ctx context.Context, programID int64) ([]loyalty.LoyaltyRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id, active, program_id, company_id, product_ids, product_category_id,
			product_tag_id, product_domain, reward_point_amount, reward_point_split,
			reward_point_mode, minimum_qty, minimum_amount, minimum_amount_tax_mode,
			mode, code, created_at, updated_at, created_by, updated_by
		FROM loyalty_rules WHERE program_id = $1 ORDER BY id
	`, programID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list loyalty rules", err)
	}
	defer rows.Close()

	var rules []loyalty.LoyaltyRule
	for rows.Next() {
		var rule loyalty.LoyaltyRule
		if err := rows.Scan(
			&rule.ID, &rule.Active, &rule.ProgramID, &rule.CompanyID, &rule.ProductIDs, &rule.ProductCategoryID,
			&rule.ProductTagID, &rule.ProductDomain, &rule.RewardPointAmount, &rule.RewardPointSplit,
			(*string)(&rule.RewardPointMode), &rule.MinimumQty, &rule.MinimumAmount, (*string)(&rule.MinimumAmountTaxMode),
			(*string)(&rule.Mode), &rule.Code, &rule.Audit.CreatedAt, &rule.Audit.UpdatedAt, &rule.Audit.CreatedBy, &rule.Audit.UpdatedBy,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan loyalty rule", err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Rewards
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateReward(ctx context.Context, reward *loyalty.LoyaltyReward) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		return r.insertRewardTx(ctx, tx, reward)
	})
}

func (r *PostgresRepo) insertRewardTx(ctx context.Context, getter interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, reward *loyalty.LoyaltyReward) error {
	productIDs := reward.DiscountProductIDs
	if productIDs == nil {
		productIDs = []int64{}
	}
	query := `
		INSERT INTO loyalty_rewards (
			active, program_id, description, reward_type, discount, discount_mode,
			discount_applicability, discount_product_ids, discount_product_category_id,
			discount_product_tag_id, discount_max_amount, discount_line_product_id,
			reward_product_id, reward_product_qty, reward_product_uom_id,
			required_points, clear_wallet, product_domain,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, NOW(), NOW(), $19, $19
		) RETURNING id
	`
	userID := audit.UserIDFromContext(ctx)
	if err := getter.QueryRow(ctx, query,
		reward.Active, reward.ProgramID, reward.Description, string(reward.RewardType), reward.Discount, string(reward.DiscountMode),
		string(reward.DiscountApplicability), productIDs, reward.DiscountProductCategoryID,
		reward.DiscountProductTagID, reward.DiscountMaxAmount, reward.DiscountLineProductID,
		reward.RewardProductID, reward.RewardProductQty, reward.RewardProductUomID,
		reward.RequiredPoints, reward.ClearWallet, reward.ProductDomain, userID,
	).Scan(&reward.ID); err != nil {
		return platformerrors.Internal("failed to create loyalty reward", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateReward(ctx context.Context, reward *loyalty.LoyaltyReward) error {
	productIDs := reward.DiscountProductIDs
	if productIDs == nil {
		productIDs = []int64{}
	}
	query := `
		UPDATE loyalty_rewards SET
			active = $1, description = $2, reward_type = $3, discount = $4, discount_mode = $5,
			discount_applicability = $6, discount_product_ids = $7, discount_product_category_id = $8,
			discount_product_tag_id = $9, discount_max_amount = $10, discount_line_product_id = $11,
			reward_product_id = $12, reward_product_qty = $13, reward_product_uom_id = $14,
			required_points = $15, clear_wallet = $16, product_domain = $17,
			updated_at = NOW(), updated_by = $18
		WHERE id = $19
	`
	tag, err := r.pool.Exec(ctx, query,
		reward.Active, reward.Description, string(reward.RewardType), reward.Discount, string(reward.DiscountMode),
		string(reward.DiscountApplicability), productIDs, reward.DiscountProductCategoryID,
		reward.DiscountProductTagID, reward.DiscountMaxAmount, reward.DiscountLineProductID,
		reward.RewardProductID, reward.RewardProductQty, reward.RewardProductUomID,
		reward.RequiredPoints, reward.ClearWallet, reward.ProductDomain,
		audit.UserIDFromContext(ctx), reward.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update loyalty reward", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty reward with id %d not found", reward.ID))
	}
	return nil
}

func (r *PostgresRepo) DeleteReward(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM loyalty_rewards WHERE id = $1`, id)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			return platformerrors.Conflict("loyalty reward is referenced by existing order lines and cannot be deleted", err)
		}
		return platformerrors.Internal("failed to delete loyalty reward", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty reward with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) GetRewardByID(ctx context.Context, id int64) (*loyalty.LoyaltyReward, error) {
	reward := &loyalty.LoyaltyReward{}
	err := r.pool.QueryRow(ctx, `
		SELECT
			id, active, program_id, description, reward_type, discount, discount_mode,
			discount_applicability, discount_product_ids, discount_product_category_id,
			discount_product_tag_id, discount_max_amount, discount_line_product_id,
			reward_product_id, reward_product_qty, reward_product_uom_id,
			required_points, clear_wallet, product_domain,
			created_at, updated_at, created_by, updated_by
		FROM loyalty_rewards WHERE id = $1
	`, id).Scan(
		&reward.ID, &reward.Active, &reward.ProgramID, &reward.Description, (*string)(&reward.RewardType), &reward.Discount, (*string)(&reward.DiscountMode),
		(*string)(&reward.DiscountApplicability), &reward.DiscountProductIDs, &reward.DiscountProductCategoryID,
		&reward.DiscountProductTagID, &reward.DiscountMaxAmount, &reward.DiscountLineProductID,
		&reward.RewardProductID, &reward.RewardProductQty, &reward.RewardProductUomID,
		&reward.RequiredPoints, &reward.ClearWallet, &reward.ProductDomain,
		&reward.Audit.CreatedAt, &reward.Audit.UpdatedAt, &reward.Audit.CreatedBy, &reward.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("loyalty reward with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get loyalty reward", err)
	}
	return reward, nil
}

func (r *PostgresRepo) listRewardsByProgram(ctx context.Context, programID int64) ([]loyalty.LoyaltyReward, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id, active, program_id, description, reward_type, discount, discount_mode,
			discount_applicability, discount_product_ids, discount_product_category_id,
			discount_product_tag_id, discount_max_amount, discount_line_product_id,
			reward_product_id, reward_product_qty, reward_product_uom_id,
			required_points, clear_wallet, product_domain,
			created_at, updated_at, created_by, updated_by
		FROM loyalty_rewards WHERE program_id = $1 ORDER BY id
	`, programID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list loyalty rewards", err)
	}
	defer rows.Close()

	var rewards []loyalty.LoyaltyReward
	for rows.Next() {
		var reward loyalty.LoyaltyReward
		if err := rows.Scan(
			&reward.ID, &reward.Active, &reward.ProgramID, &reward.Description, (*string)(&reward.RewardType), &reward.Discount, (*string)(&reward.DiscountMode),
			(*string)(&reward.DiscountApplicability), &reward.DiscountProductIDs, &reward.DiscountProductCategoryID,
			&reward.DiscountProductTagID, &reward.DiscountMaxAmount, &reward.DiscountLineProductID,
			&reward.RewardProductID, &reward.RewardProductQty, &reward.RewardProductUomID,
			&reward.RequiredPoints, &reward.ClearWallet, &reward.ProductDomain,
			&reward.Audit.CreatedAt, &reward.Audit.UpdatedAt, &reward.Audit.CreatedBy, &reward.Audit.UpdatedBy,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan loyalty reward", err)
		}
		rewards = append(rewards, reward)
	}
	return rewards, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Mails (config only)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateMail(ctx context.Context, mail *loyalty.LoyaltyMail) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		return r.insertMailTx(ctx, tx, mail)
	})
}

func (r *PostgresRepo) insertMailTx(ctx context.Context, getter interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, mail *loyalty.LoyaltyMail) error {
	query := `
		INSERT INTO loyalty_mails (active, program_id, trigger, points, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id, created_at, updated_at
	`
	if err := getter.QueryRow(ctx, query, mail.Active, mail.ProgramID, string(mail.Trigger), mail.Points).
		Scan(&mail.ID, &mail.Audit.CreatedAt, &mail.Audit.UpdatedAt); err != nil {
		return platformerrors.Internal("failed to create loyalty mail", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateMail(ctx context.Context, mail *loyalty.LoyaltyMail) error {
	query := `
		UPDATE loyalty_mails SET active = $1, trigger = $2, points = $3, updated_at = NOW()
		WHERE id = $4
	`
	tag, err := r.pool.Exec(ctx, query, mail.Active, string(mail.Trigger), mail.Points, mail.ID)
	if err != nil {
		return platformerrors.Internal("failed to update loyalty mail", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty mail with id %d not found", mail.ID))
	}
	return nil
}

func (r *PostgresRepo) DeleteMail(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM loyalty_mails WHERE id = $1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete loyalty mail", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty mail with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) listMailsByProgram(ctx context.Context, programID int64) ([]loyalty.LoyaltyMail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, active, program_id, trigger, points, created_at, updated_at
		FROM loyalty_mails WHERE program_id = $1 ORDER BY id
	`, programID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list loyalty mails", err)
	}
	defer rows.Close()

	var mails []loyalty.LoyaltyMail
	for rows.Next() {
		var mail loyalty.LoyaltyMail
		if err := rows.Scan(&mail.ID, &mail.Active, &mail.ProgramID, (*string)(&mail.Trigger), &mail.Points,
			&mail.Audit.CreatedAt, &mail.Audit.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan loyalty mail", err)
		}
		mails = append(mails, mail)
	}
	return mails, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Cards
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateCard(ctx context.Context, card *loyalty.LoyaltyCard) error {
	query := `
		INSERT INTO loyalty_cards (
			program_id, company_id, partner_id, points, code, expiration_date,
			use_count, order_id, active, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW(), $10, $10
		) RETURNING id, created_at, updated_at
	`
	userID := audit.UserIDFromContext(ctx)
	err := r.pool.QueryRow(ctx, query,
		card.ProgramID, card.CompanyID, card.PartnerID, card.Points, card.Code, card.ExpirationDate,
		card.UseCount, card.OrderID, card.Active, userID,
	).Scan(&card.ID, &card.Audit.CreatedAt, &card.Audit.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "loyalty_cards_code_key") {
			return platformerrors.Conflict(fmt.Sprintf("loyalty card code '%s' already exists", card.Code), err)
		}
		return platformerrors.Internal("failed to create loyalty card", err)
	}
	card.Audit.CreatedBy = userID
	card.Audit.UpdatedBy = userID
	return nil
}

func (r *PostgresRepo) UpdateCard(ctx context.Context, card *loyalty.LoyaltyCard) error {
	query := `
		UPDATE loyalty_cards SET
			program_id = $1, company_id = $2, partner_id = $3, points = $4,
			expiration_date = $5, use_count = $6, order_id = $7, active = $8,
			updated_at = NOW(), updated_by = $9
		WHERE id = $10
	`
	tag, err := r.pool.Exec(ctx, query,
		card.ProgramID, card.CompanyID, card.PartnerID, card.Points,
		card.ExpirationDate, card.UseCount, card.OrderID, card.Active,
		audit.UserIDFromContext(ctx), card.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update loyalty card", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("loyalty card with id %d not found", card.ID))
	}
	return nil
}

func (r *PostgresRepo) GetCardByID(ctx context.Context, id int64) (*loyalty.LoyaltyCard, error) {
	return r.scanCard(ctx, `SELECT
			id, program_id, company_id, partner_id, points, code, expiration_date,
			use_count, order_id, active, created_at, updated_at, created_by, updated_by
		FROM loyalty_cards WHERE id = $1`, id)
}

func (r *PostgresRepo) GetCardByCode(ctx context.Context, code string) (*loyalty.LoyaltyCard, error) {
	return r.scanCard(ctx, `SELECT
			id, program_id, company_id, partner_id, points, code, expiration_date,
			use_count, order_id, active, created_at, updated_at, created_by, updated_by
		FROM loyalty_cards WHERE code = $1`, code)
}

func (r *PostgresRepo) scanCard(ctx context.Context, query string, arg any) (*loyalty.LoyaltyCard, error) {
	card := &loyalty.LoyaltyCard{}
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&card.ID, &card.ProgramID, &card.CompanyID, &card.PartnerID, &card.Points, &card.Code, &card.ExpirationDate,
		&card.UseCount, &card.OrderID, &card.Active, &card.Audit.CreatedAt, &card.Audit.UpdatedAt, &card.Audit.CreatedBy, &card.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("loyalty card not found")
		}
		return nil, platformerrors.Internal("failed to get loyalty card", err)
	}
	return card, nil
}

func (r *PostgresRepo) ListCards(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[loyalty.LoyaltyCard], error) {
	if f == nil {
		f = filter.NewFilter()
	}
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedCardFilterFields, 1)
	if err != nil {
		return pagination.PageResult[loyalty.LoyaltyCard]{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM loyalty_cards %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[loyalty.LoyaltyCard]{}, platformerrors.Internal("failed to count loyalty cards", err)
	}
	if totalItems == 0 {
		return pagination.NewPageResult([]loyalty.LoyaltyCard{}, 0, page), nil
	}

	dataQuery := fmt.Sprintf(`
		SELECT
			id, program_id, company_id, partner_id, points, code, expiration_date,
			use_count, order_id, active, created_at, updated_at, created_by, updated_by
		FROM loyalty_cards
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[loyalty.LoyaltyCard]{}, platformerrors.Internal("failed to list loyalty cards", err)
	}
	defer rows.Close()

	var cards []loyalty.LoyaltyCard
	for rows.Next() {
		var card loyalty.LoyaltyCard
		if err := rows.Scan(
			&card.ID, &card.ProgramID, &card.CompanyID, &card.PartnerID, &card.Points, &card.Code, &card.ExpirationDate,
			&card.UseCount, &card.OrderID, &card.Active, &card.Audit.CreatedAt, &card.Audit.UpdatedAt, &card.Audit.CreatedBy, &card.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[loyalty.LoyaltyCard]{}, platformerrors.Internal("failed to scan loyalty card", err)
		}
		cards = append(cards, card)
	}
	return pagination.NewPageResult(cards, totalItems, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// History
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) AddHistory(ctx context.Context, h *loyalty.LoyaltyHistory) error {
	query := `
		INSERT INTO loyalty_card_history (
			card_id, company_id, description, issued, used, order_model, order_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW()) RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		h.CardID, h.CompanyID, h.Description, h.Issued, h.Used, h.OrderModel, h.OrderID,
	).Scan(&h.ID, &h.CreatedAt)
	if err != nil {
		return platformerrors.Internal("failed to add loyalty history entry", err)
	}
	return nil
}

func (r *PostgresRepo) ListHistoryByCard(ctx context.Context, cardID int64) ([]loyalty.LoyaltyHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, card_id, company_id, description, issued, used, order_model, order_id, created_at
		FROM loyalty_card_history WHERE card_id = $1 ORDER BY created_at DESC, id DESC
	`, cardID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list loyalty history", err)
	}
	defer rows.Close()

	var history []loyalty.LoyaltyHistory
	for rows.Next() {
		var h loyalty.LoyaltyHistory
		if err := rows.Scan(&h.ID, &h.CardID, &h.CompanyID, &h.Description, &h.Issued, &h.Used, &h.OrderModel, &h.OrderID, &h.CreatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan loyalty history", err)
		}
		history = append(history, h)
	}
	return history, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Sale Order Coupon Points
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) UpsertCouponPoints(ctx context.Context, orderID, couponID int64, points float64) error {
	query := `
		INSERT INTO sale_order_coupon_points (order_id, coupon_id, points, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (order_id, coupon_id) DO UPDATE SET points = EXCLUDED.points
	`
	if _, err := r.pool.Exec(ctx, query, orderID, couponID, points); err != nil {
		return platformerrors.Internal("failed to upsert coupon points", err)
	}
	return nil
}

func (r *PostgresRepo) RemoveCouponPointsByOrder(ctx context.Context, orderID int64) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM sale_order_coupon_points WHERE order_id = $1`, orderID); err != nil {
		return platformerrors.Internal("failed to remove coupon points", err)
	}
	return nil
}

func (r *PostgresRepo) ListCouponPointsByOrder(ctx context.Context, orderID int64) ([]loyalty.OrderCouponPoints, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, coupon_id, points, created_at
		FROM sale_order_coupon_points WHERE order_id = $1 ORDER BY id
	`, orderID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list coupon points", err)
	}
	defer rows.Close()

	var pts []loyalty.OrderCouponPoints
	for rows.Next() {
		var p loyalty.OrderCouponPoints
		if err := rows.Scan(&p.ID, &p.OrderID, &p.CouponID, &p.Points, &p.CreatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan coupon points", err)
		}
		pts = append(pts, p)
	}
	return pts, nil
}