package delivery

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/delivery"
	platformerrors "cashflow_backend/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateCarrier(ctx context.Context, c *delivery.DeliveryCarrier) error {
	query := `
		INSERT INTO delivery_carrier (
			name, active, sequence, delivery_type, integration_level, invoice_policy,
			product_id, fixed_price, margin, fixed_margin, free_over, amount,
			max_weight, max_volume, company_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		c.Name, c.Active, c.Sequence, c.DeliveryType, c.IntegrationLevel, c.InvoicePolicy,
		c.ProductID, c.FixedPrice, c.Margin, c.FixedMargin, c.FreeOver, c.Amount,
		c.MaxWeight, c.MaxVolume, c.CompanyID,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return err
	}

	return r.updateRelations(ctx, c)
}

func (r *PostgresRepository) GetCarrierByID(ctx context.Context, id int64) (*delivery.DeliveryCarrier, error) {
	c := &delivery.DeliveryCarrier{}
	query := `
		SELECT
			id, name, active, sequence, delivery_type, integration_level, invoice_policy,
			product_id, fixed_price, margin, fixed_margin, free_over, amount,
			max_weight, max_volume, company_id, created_at, updated_at
		FROM delivery_carrier WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Active, &c.Sequence, &c.DeliveryType, &c.IntegrationLevel, &c.InvoicePolicy,
		&c.ProductID, &c.FixedPrice, &c.Margin, &c.FixedMargin, &c.FreeOver, &c.Amount,
		&c.MaxWeight, &c.MaxVolume, &c.CompanyID, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platformerrors.NotFound(fmt.Sprintf("delivery carrier %d not found", id))
		}
		return nil, err
	}

	// Load Rules
	rulesQuery := `SELECT id, sequence, variable, operator, max_value, list_base_price, list_price, variable_factor
	               FROM delivery_price_rule WHERE carrier_id = $1 ORDER BY sequence, id`
	rows, err := r.pool.Query(ctx, rulesQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		rule := delivery.DeliveryPriceRule{CarrierID: id}
		if err := rows.Scan(&rule.ID, &rule.Sequence, &rule.Variable, &rule.Operator, &rule.MaxValue, &rule.ListBasePrice, &rule.ListPrice, &rule.VariableFactor); err != nil {
			return nil, err
		}
		c.PriceRules = append(c.PriceRules, rule)
	}

	// Load Relations
	r.pool.QueryRow(ctx, "SELECT array_agg(country_id) FROM delivery_carrier_country_rel WHERE carrier_id = $1", id).Scan(&c.CountryIDs)
	r.pool.QueryRow(ctx, "SELECT array_agg(state_id) FROM delivery_carrier_state_rel WHERE carrier_id = $1", id).Scan(&c.StateIDs)
	r.pool.QueryRow(ctx, "SELECT array_agg(zip_prefix_id) FROM delivery_carrier_zip_prefix_rel WHERE carrier_id = $1", id).Scan(&c.ZipPrefixIDs)

	if c.CountryIDs == nil { c.CountryIDs = []int64{} }
	if c.StateIDs == nil { c.StateIDs = []int64{} }
	if c.ZipPrefixIDs == nil { c.ZipPrefixIDs = []int64{} }

	return c, nil
}

func (r *PostgresRepository) ListCarriers(ctx context.Context, filters map[string]interface{}) ([]delivery.DeliveryCarrier, error) {
	query := `SELECT id, name, active, delivery_type FROM delivery_carrier WHERE active = true ORDER BY sequence, id`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var carriers []delivery.DeliveryCarrier
	for rows.Next() {
		var c delivery.DeliveryCarrier
		if err := rows.Scan(&c.ID, &c.Name, &c.Active, &c.DeliveryType); err != nil {
			return nil, err
		}
		carriers = append(carriers, c)
	}
	return carriers, nil
}

func (r *PostgresRepository) UpdateCarrier(ctx context.Context, c *delivery.DeliveryCarrier) error {
	query := `
		UPDATE delivery_carrier SET
			name=$1, active=$2, sequence=$3, delivery_type=$4, integration_level=$5, invoice_policy=$6,
			product_id=$7, fixed_price=$8, margin=$9, fixed_margin=$10, free_over=$11, amount=$12,
			max_weight=$13, max_volume=$14, company_id=$15, updated_at=CURRENT_TIMESTAMP
		WHERE id=$16`

	_, err := r.pool.Exec(ctx, query,
		c.Name, c.Active, c.Sequence, c.DeliveryType, c.IntegrationLevel, c.InvoicePolicy,
		c.ProductID, c.FixedPrice, c.Margin, c.FixedMargin, c.FreeOver, c.Amount,
		c.MaxWeight, c.MaxVolume, c.CompanyID, c.ID,
	)
	if err != nil {
		return err
	}

	return r.updateRelations(ctx, c)
}

func (r *PostgresRepository) DeleteCarrier(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM delivery_carrier WHERE id = $1", id)
	return err
}

func (r *PostgresRepository) CreatePriceRule(ctx context.Context, rule *delivery.DeliveryPriceRule) error {
	query := `INSERT INTO delivery_price_rule (carrier_id, sequence, variable, operator, max_value, list_base_price, list_price, variable_factor)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	return r.pool.QueryRow(ctx, query, rule.CarrierID, rule.Sequence, rule.Variable, rule.Operator, rule.MaxValue, rule.ListBasePrice, rule.ListPrice, rule.VariableFactor).Scan(&rule.ID)
}

func (r *PostgresRepository) DeletePriceRule(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM delivery_price_rule WHERE id = $1", id)
	return err
}

func (r *PostgresRepository) GetZipPrefixes(ctx context.Context, ids []int64) ([]delivery.DeliveryZipPrefix, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, name FROM delivery_zip_prefix WHERE id = ANY($1)", ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prefixes []delivery.DeliveryZipPrefix
	for rows.Next() {
		var p delivery.DeliveryZipPrefix
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		prefixes = append(prefixes, p)
	}
	return prefixes, nil
}

func (r *PostgresRepository) CreateZipPrefix(ctx context.Context, name string) (*delivery.DeliveryZipPrefix, error) {
	p := &delivery.DeliveryZipPrefix{Name: name}
	err := r.pool.QueryRow(ctx, "INSERT INTO delivery_zip_prefix (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id", name).Scan(&p.ID)
	return p, err
}

func (r *PostgresRepository) updateRelations(ctx context.Context, c *delivery.DeliveryCarrier) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Countries
		if _, err := tx.Exec(ctx, "DELETE FROM delivery_carrier_country_rel WHERE carrier_id = $1", c.ID); err != nil {
			return err
		}
		for _, cid := range c.CountryIDs {
			if _, err := tx.Exec(ctx, "INSERT INTO delivery_carrier_country_rel (carrier_id, country_id) VALUES ($1, $2)", c.ID, cid); err != nil {
				return err
			}
		}

		// States
		if _, err := tx.Exec(ctx, "DELETE FROM delivery_carrier_state_rel WHERE carrier_id = $1", c.ID); err != nil {
			return err
		}
		for _, sid := range c.StateIDs {
			if _, err := tx.Exec(ctx, "INSERT INTO delivery_carrier_state_rel (carrier_id, state_id) VALUES ($1, $2)", c.ID, sid); err != nil {
				return err
			}
		}

		// Zip Prefixes
		if _, err := tx.Exec(ctx, "DELETE FROM delivery_carrier_zip_prefix_rel WHERE carrier_id = $1", c.ID); err != nil {
			return err
		}
		for _, zid := range c.ZipPrefixIDs {
			if _, err := tx.Exec(ctx, "INSERT INTO delivery_carrier_zip_prefix_rel (carrier_id, zip_prefix_id) VALUES ($1, $2)", c.ID, zid); err != nil {
				return err
			}
		}
		return nil
	})
}
