package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/features/accounting"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *PostgresRepository) GetTrialBalanceData(ctx context.Context, fromDate, toDate time.Time) ([]*accounting.TrialBalanceRow, error) {
	if fromDate.After(toDate) {
		return nil, errors.New("from date must not be after to date")
	}
	const query = `
		WITH initial_balances AS (
			SELECT l.account_id, SUM(l.debit) AS initial_debit, SUM(l.credit) AS initial_credit
			FROM account_move_lines l
			JOIN account_moves m ON m.id = l.move_id
			WHERE m.state = 'posted' AND l.date < $1
			GROUP BY l.account_id
		),
		period_balances AS (
			SELECT l.account_id, SUM(l.debit) AS period_debit, SUM(l.credit) AS period_credit
			FROM account_move_lines l
			JOIN account_moves m ON m.id = l.move_id
			WHERE m.state = 'posted' AND l.date >= $1 AND l.date <= $2
			GROUP BY l.account_id
		)
		SELECT
			a.id, a.code, a.name,
			COALESCE(i.initial_debit, 0), COALESCE(i.initial_credit, 0),
			COALESCE(p.period_debit, 0), COALESCE(p.period_credit, 0),
			COALESCE(i.initial_debit, 0) + COALESCE(p.period_debit, 0),
			COALESCE(i.initial_credit, 0) + COALESCE(p.period_credit, 0)
		FROM accounts a
		LEFT JOIN initial_balances i ON i.account_id = a.id
		LEFT JOIN period_balances p ON p.account_id = a.id
		ORDER BY a.code ASC`

	rows, err := r.pool.Query(ctx, query, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("query trial balance: %w", err)
	}
	defer rows.Close()

	results := make([]*accounting.TrialBalanceRow, 0)
	for rows.Next() {
		var row accounting.TrialBalanceRow
		if err := rows.Scan(
			&row.AccountID, &row.AccountCode, &row.AccountName,
			&row.InitialDebit, &row.InitialCredit, &row.PeriodDebit, &row.PeriodCredit,
			&row.EndingDebit, &row.EndingCredit,
		); err != nil {
			return nil, fmt.Errorf("scan trial balance row: %w", err)
		}
		results = append(results, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trial balance rows: %w", err)
	}
	return results, nil
}

func (r *PostgresRepository) GetGeneralLedgerLines(ctx context.Context, accountID uuid.UUID, fromDate, toDate time.Time) ([]*accounting.GeneralLedgerLineRow, error) {
	if fromDate.After(toDate) {
		return nil, errors.New("from date must not be after to date")
	}
	const query = `
		WITH ledger AS (
			SELECT
				m.id AS move_id,
				m.name AS move_name,
				m.date,
				COALESCE(m.ref, '') AS ref,
				l.id AS line_id,
				l.debit,
				l.credit,
				SUM(l.debit - l.credit) OVER (
					ORDER BY m.date, m.name, l.id
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) AS running_balance
			FROM account_move_lines l
			JOIN account_moves m ON m.id = l.move_id
			WHERE l.account_id = $1 AND m.state = 'posted' AND m.date <= $3
		)
		SELECT move_id, move_name, date, ref, debit, credit, running_balance
		FROM ledger
		WHERE date >= $2
		ORDER BY date, move_name, line_id`

	rows, err := r.pool.Query(ctx, query, accountID, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("query general ledger: %w", err)
	}
	defer rows.Close()

	results := make([]*accounting.GeneralLedgerLineRow, 0)
	for rows.Next() {
		var row accounting.GeneralLedgerLineRow
		if err := rows.Scan(
			&row.MoveID, &row.MoveName, &row.Date, &row.Ref,
			&row.Debit, &row.Credit, &row.RunningBalance,
		); err != nil {
			return nil, fmt.Errorf("scan general ledger row: %w", err)
		}
		results = append(results, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate general ledger rows: %w", err)
	}
	return results, nil
}

func (r *PostgresRepository) GetFXMonetaryBalances(ctx context.Context, asOf time.Time) ([]*accounting.FXMonetaryBalance, error) {
	const query = `
		SELECT
			l.account_id, a.account_type, l.partner_id, l.cost_center_id, l.currency_id,
			SUM(COALESCE(l.amount_currency, 0)),
			SUM(l.debit - l.credit)
		FROM account_move_lines l
		JOIN account_moves m ON m.id = l.move_id
		JOIN accounts a ON a.id = l.account_id
		WHERE m.state = 'posted' AND l.date <= $1
		GROUP BY l.account_id, a.account_type, l.partner_id, l.cost_center_id, l.currency_id
		HAVING SUM(COALESCE(l.amount_currency, 0)) <> 0
		ORDER BY l.account_id, l.currency_id, l.partner_id, l.cost_center_id`

	rows, err := r.pool.Query(ctx, query, asOf)
	if err != nil {
		return nil, fmt.Errorf("query FX monetary balances: %w", err)
	}
	defer rows.Close()

	balances := make([]*accounting.FXMonetaryBalance, 0)
	for rows.Next() {
		var balance accounting.FXMonetaryBalance
		var partnerID, costCenterID pgtype.UUID
		if err := rows.Scan(
			&balance.AccountID, &balance.AccountType, &partnerID, &costCenterID, &balance.CurrencyID,
			&balance.ForeignAmount, &balance.FunctionalCarryingAmount,
		); err != nil {
			return nil, fmt.Errorf("scan FX monetary balance: %w", err)
		}
		balance.PartnerID = nullableUUID(partnerID)
		balance.CostCenterID = nullableUUID(costCenterID)
		balances = append(balances, &balance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate FX monetary balances: %w", err)
	}
	return balances, nil
}
