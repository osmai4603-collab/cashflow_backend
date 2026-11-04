package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/features/accounting"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type reconciliationLine struct {
	id               uuid.UUID
	accountID        uuid.UUID
	debit            decimal.Decimal
	credit           decimal.Decimal
	amountCurrency   decimal.Decimal
	reconciledAmount decimal.Decimal
	currencyID       string
	date             time.Time
	isReconcilable   bool
	isActive         bool
	state            accounting.MoveState
}

func (r *PostgresRepository) ReconcileMoveLines(
	ctx context.Context,
	date time.Time,
	actorID, accountID uuid.UUID,
	currencyID string,
	allocations []accounting.ReconciliationAllocation,
) (uuid.UUID, error) {
	if actorID == uuid.Nil || len(allocations) < 2 {
		return uuid.Nil, errors.New("actor and at least two reconciliation allocations are required")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin reconciliation: %w", err)
	}
	defer tx.Rollback(ctx)

	lines := make([]reconciliationLine, len(allocations))
	seen := make(map[uuid.UUID]struct{}, len(allocations))
	var debitForeign, creditForeign, debitFunctional, creditFunctional decimal.Decimal
	for i, allocation := range allocations {
		if allocation.MoveLineID == uuid.Nil || !allocation.Amount.IsPositive() {
			return uuid.Nil, errors.New("reconciliation line ids and positive allocations are required")
		}
		if _, exists := seen[allocation.MoveLineID]; exists {
			return uuid.Nil, errors.New("a move line cannot be allocated more than once")
		}
		seen[allocation.MoveLineID] = struct{}{}
		line, err := lockReconciliationLine(ctx, tx, allocation.MoveLineID)
		if err != nil {
			return uuid.Nil, err
		}
		if line.state != accounting.MoveStatePosted || !line.isActive || !line.isReconcilable {
			return uuid.Nil, accounting.ErrAccountNotReconcilable
		}
		if date.Before(line.date) {
			return uuid.Nil, errors.New("reconciliation date cannot precede a move line date")
		}
		if accountID != uuid.Nil && accountID != line.accountID {
			return uuid.Nil, errors.New("all reconciled lines must belong to the requested account")
		}
		if currencyID != "" && currencyID != line.currencyID {
			return uuid.Nil, errors.New("all reconciled lines must use the requested currency")
		}
		if i > 0 && (lines[0].accountID != line.accountID || lines[0].currencyID != line.currencyID) {
			return uuid.Nil, errors.New("all reconciled lines must belong to one account and currency")
		}
		capacity := line.amountCurrency.Abs()
		if capacity.IsZero() {
			capacity = line.debit.Sub(line.credit).Abs()
		}
		residual := capacity.Sub(line.reconciledAmount)
		if allocation.Amount.GreaterThan(residual) {
			return uuid.Nil, fmt.Errorf("allocation %s exceeds residual %s for move line %s", allocation.Amount, residual, allocation.MoveLineID)
		}
		functionalCapacity := line.debit.Sub(line.credit).Abs()
		functionalAllocation := functionalCapacity
		if !allocation.Amount.Equal(capacity) {
			functionalAllocation = functionalCapacity.Mul(allocation.Amount).Div(capacity).Round(4)
		}
		if line.debit.GreaterThan(line.credit) {
			debitForeign = debitForeign.Add(allocation.Amount)
			debitFunctional = debitFunctional.Add(functionalAllocation)
		} else {
			creditForeign = creditForeign.Add(allocation.Amount)
			creditFunctional = creditFunctional.Add(functionalAllocation)
		}
		lines[i] = line
	}
	if !debitForeign.Equal(creditForeign) {
		return uuid.Nil, fmt.Errorf("allocated debit amount %s does not equal allocated credit amount %s", debitForeign, creditForeign)
	}
	if debitForeign.IsZero() {
		return uuid.Nil, errors.New("reconciliation allocation must not be zero")
	}
	reconciliationID := uuid.New()
	currencyID = lines[0].currencyID
	const insertReconciliation = `
		INSERT INTO account_reconciliations (id, date, account_id, currency_id, created_by)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := tx.Exec(ctx, insertReconciliation, reconciliationID, date, lines[0].accountID, currencyID, actorID); err != nil {
		return uuid.Nil, fmt.Errorf("insert reconciliation: %w", err)
	}
	for i, allocation := range allocations {
		const insertItem = `
			INSERT INTO account_reconciliation_items (reconciliation_id, move_line_id, allocated_amount, currency_id)
			VALUES ($1, $2, $3, $4)`
		if _, err := tx.Exec(ctx, insertItem, reconciliationID, allocation.MoveLineID, allocation.Amount, currencyID); err != nil {
			return uuid.Nil, fmt.Errorf("insert reconciliation allocation %d: %w", i+1, err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE account_move_lines
			SET reconciled_amount = reconciled_amount + $2
			WHERE id = $1`, allocation.MoveLineID, allocation.Amount); err != nil {
			return uuid.Nil, fmt.Errorf("update reconciled amount for line %s: %w", allocation.MoveLineID, err)
		}
	}

	var realizedMoveID *uuid.UUID
	realized := debitFunctional.Sub(creditFunctional)
	if !realized.IsZero() {
		configuration, err := getAccountingConfiguration(ctx, tx)
		if err != nil {
			return uuid.Nil, err
		}
		moveID, err := postRealizedFXMove(ctx, tx, lines[0].accountID, realized, date, actorID, reconciliationID, configuration)
		if err != nil {
			return uuid.Nil, err
		}
		realizedMoveID = &moveID
		if _, err := tx.Exec(ctx, `
			UPDATE account_reconciliations SET realized_fx_move_id = $2 WHERE id = $1`,
			reconciliationID, moveID,
		); err != nil {
			return uuid.Nil, fmt.Errorf("link realized FX move: %w", err)
		}
	}
	payload, err := json.Marshal(map[string]any{
		"id": reconciliationID, "account_id": lines[0].accountID,
		"currency_id": currencyID, "realized_fx_move_id": realizedMoveID,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode reconciliation audit payload: %w", err)
	}
	if err := insertAuditLog(ctx, tx, &accounting.FinancialAuditLog{
		EntityType: "account_reconciliation", EntityID: reconciliationID,
		Action: "RECONCILE", ActorID: actorID, PayloadAfter: payload, Timestamp: time.Now().UTC(),
	}); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit reconciliation: %w", err)
	}
	return reconciliationID, nil
}

func lockReconciliationLine(ctx context.Context, tx pgx.Tx, lineID uuid.UUID) (reconciliationLine, error) {
	var line reconciliationLine
	const query = `
		SELECT l.id, l.account_id, l.debit, l.credit,
			CASE WHEN l.amount_currency IS NULL OR l.amount_currency = 0
				THEN l.debit - l.credit ELSE l.amount_currency END,
			l.reconciled_amount,
			l.currency_id, l.date, a.is_reconcilable, a.is_active, m.state
		FROM account_move_lines l
		JOIN accounts a ON a.id = l.account_id
		JOIN account_moves m ON m.id = l.move_id
		WHERE l.id = $1
		FOR UPDATE OF l`
	if err := tx.QueryRow(ctx, query, lineID).Scan(
		&line.id, &line.accountID, &line.debit, &line.credit, &line.amountCurrency,
		&line.reconciledAmount, &line.currencyID, &line.date, &line.isReconcilable,
		&line.isActive, &line.state,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reconciliationLine{}, fmt.Errorf("move line %s not found", lineID)
		}
		return reconciliationLine{}, fmt.Errorf("lock move line %s for reconciliation: %w", lineID, err)
	}
	return line, nil
}

func getAccountingConfiguration(ctx context.Context, tx pgx.Tx) (*accounting.AccountingConfiguration, error) {
	var configuration accounting.AccountingConfiguration
	err := tx.QueryRow(ctx, `
		SELECT retained_earnings_account_id, fx_gain_account_id, fx_loss_account_id,
			fx_reconciliation_journal_id, functional_currency
		FROM accounting_configurations
		ORDER BY updated_at DESC
		LIMIT 1`).Scan(
		&configuration.RetainedEarningsAccountID, &configuration.FXGainAccountID,
		&configuration.FXLossAccountID, &configuration.FXReconciliationJournalID,
		&configuration.FunctionalCurrency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrAccountingConfigurationMissing
	}
	if err != nil {
		return nil, fmt.Errorf("load accounting configuration: %w", err)
	}
	return &configuration, nil
}

func postRealizedFXMove(
	ctx context.Context,
	tx pgx.Tx,
	reconAccountID uuid.UUID,
	debitMinusCredit decimal.Decimal,
	date time.Time,
	actorID, reconciliationID uuid.UUID,
	configuration *accounting.AccountingConfiguration,
) (uuid.UUID, error) {
	period, err := periodInTransaction(ctx, tx, date)
	if err != nil {
		return uuid.Nil, err
	}
	if err := lockOpenPeriod(ctx, tx, period, date); err != nil {
		return uuid.Nil, err
	}
	var journalActive bool
	if err := tx.QueryRow(ctx, `SELECT is_active FROM journals WHERE id = $1`, configuration.FXReconciliationJournalID).Scan(&journalActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, accounting.ErrJournalNotFound
		}
		return uuid.Nil, fmt.Errorf("check FX reconciliation journal: %w", err)
	}
	if !journalActive {
		return uuid.Nil, errors.New("FX reconciliation journal must be active")
	}
	var gainType, lossType accounting.AccountType
	if err := tx.QueryRow(ctx, `SELECT account_type FROM accounts WHERE id = $1 AND is_active`, configuration.FXGainAccountID).Scan(&gainType); err != nil {
		return uuid.Nil, fmt.Errorf("load active FX gain account: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT account_type FROM accounts WHERE id = $1 AND is_active`, configuration.FXLossAccountID).Scan(&lossType); err != nil {
		return uuid.Nil, fmt.Errorf("load active FX loss account: %w", err)
	}
	if gainType != accounting.AccountTypeIncome || lossType != accounting.AccountTypeExpense {
		return uuid.Nil, errors.New("configured FX gain account must be income and FX loss account must be expense")
	}
	amount := debitMinusCredit.Abs().Round(4)
	if amount.IsZero() {
		return uuid.Nil, errors.New("realized FX adjustment rounds to zero")
	}
	sequence, err := nextSequence(ctx, tx, configuration.FXReconciliationJournalID)
	if err != nil {
		return uuid.Nil, err
	}
	moveID := uuid.New()
	now := time.Now().UTC()
	lines := []accounting.AccountMoveLine{
		{ID: uuid.New(), MoveID: moveID, AccountID: reconAccountID, CurrencyID: configuration.FunctionalCurrency, Date: date, CreatedAt: now},
		{ID: uuid.New(), MoveID: moveID, CurrencyID: configuration.FunctionalCurrency, Date: date, CreatedAt: now},
	}
	if debitMinusCredit.IsPositive() {
		lines[0].Credit = amount
		lines[1].AccountID, lines[1].Debit = configuration.FXLossAccountID, amount
	} else {
		lines[0].Debit = amount
		lines[1].AccountID, lines[1].Credit = configuration.FXGainAccountID, amount
	}
	previousHash, err := previousPostedMoveHash(ctx, tx, configuration.FXReconciliationJournalID)
	if err != nil {
		return uuid.Nil, err
	}
	name := sequence
	hash := hashMove(moveID, name, date, configuration.FXReconciliationJournalID, lines, previousHash)
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_moves (
			id, name, journal_id, period_id, date, state, ref, hash_signature, posted_at,
			posted_by, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'posted', $6, $7, $8, $9, $9, $8, $8)`,
		moveID, name, configuration.FXReconciliationJournalID, period, date,
		"Realized FX reconciliation "+reconciliationID.String(), hash, now, actorID,
	); err != nil {
		return uuid.Nil, fmt.Errorf("insert realized FX move: %w", err)
	}
	if err := insertMoveLines(ctx, tx, moveID, date, lines); err != nil {
		return uuid.Nil, err
	}
	payload, err := json.Marshal(map[string]any{"move_id": moveID, "reconciliation_id": reconciliationID})
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode realized FX audit payload: %w", err)
	}
	if err := insertAuditLog(ctx, tx, &accounting.FinancialAuditLog{
		EntityType: "account_move", EntityID: moveID, Action: "POST",
		ActorID: actorID, PayloadAfter: payload, Timestamp: now,
	}); err != nil {
		return uuid.Nil, err
	}
	return moveID, nil
}

func nextSequence(ctx context.Context, tx pgx.Tx, journalID uuid.UUID) (string, error) {
	var prefix string
	var next int64
	if err := tx.QueryRow(ctx, `
		SELECT sequence_prefix, next_number FROM journals WHERE id = $1 FOR UPDATE`,
		journalID,
	).Scan(&prefix, &next); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", accounting.ErrJournalNotFound
		}
		return "", fmt.Errorf("lock journal sequence: %w", err)
	}
	if next < 1 {
		return "", fmt.Errorf("journal %s has invalid next sequence number %d", journalID, next)
	}
	if _, err := tx.Exec(ctx, `UPDATE journals SET next_number = next_number + 1 WHERE id = $1`, journalID); err != nil {
		return "", fmt.Errorf("increment journal sequence: %w", err)
	}
	return fmt.Sprintf("%s%05d", prefix, next), nil
}

func periodInTransaction(ctx context.Context, tx pgx.Tx, date time.Time) (uuid.UUID, error) {
	var periodID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id FROM fiscal_periods
		WHERE start_date <= $1 AND end_date >= $1
		ORDER BY start_date DESC LIMIT 1`, date,
	).Scan(&periodID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, accounting.ErrPeriodNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find fiscal period for FX move: %w", err)
	}
	return periodID, nil
}

func (r *PostgresRepository) ListUnreconciledLines(
	ctx context.Context,
	filter accounting.UnreconciledLineFilter,
) ([]*accounting.UnreconciledLine, int, error) {
	const baseWhere = `
		FROM account_move_lines l
		JOIN account_moves m ON m.id = l.move_id
		JOIN accounts a ON a.id = l.account_id
		WHERE m.state = 'posted' AND a.is_reconcilable = TRUE
			AND ABS(CASE WHEN l.amount_currency IS NULL OR l.amount_currency = 0
				THEN l.debit - l.credit ELSE l.amount_currency END) > l.reconciled_amount
			AND ($1::uuid IS NULL OR a.id = $1)
			AND ($2::text = '' OR l.currency_id = $2)`
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) `+baseWhere, filter.AccountID, filter.CurrencyID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count unreconciled lines: %w", err)
	}
	query := `
		SELECT l.id, m.id, m.name, l.date, a.id, a.code, a.name,
			l.debit, l.credit,
			CASE WHEN l.amount_currency IS NULL OR l.amount_currency = 0
				THEN l.debit - l.credit ELSE l.amount_currency END,
			l.currency_id,
			ABS(CASE WHEN l.amount_currency IS NULL OR l.amount_currency = 0
				THEN l.debit - l.credit ELSE l.amount_currency END) - l.reconciled_amount,
			l.partner_id` + baseWhere + `
		ORDER BY l.date, m.name, l.id
		LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, query, filter.AccountID, filter.CurrencyID, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query unreconciled lines: %w", err)
	}
	defer rows.Close()
	lines := make([]*accounting.UnreconciledLine, 0)
	for rows.Next() {
		line := &accounting.UnreconciledLine{}
		var partnerID pgtype.UUID
		if err := rows.Scan(
			&line.MoveLineID, &line.MoveID, &line.MoveName, &line.Date,
			&line.AccountID, &line.AccountCode, &line.AccountName,
			&line.Debit, &line.Credit, &line.AmountCurrency, &line.CurrencyID,
			&line.ResidualAmount, &partnerID,
		); err != nil {
			return nil, 0, fmt.Errorf("scan unreconciled line: %w", err)
		}
		line.PartnerID = nullableUUID(partnerID)
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate unreconciled lines: %w", err)
	}
	return lines, total, nil
}

func (r *PostgresRepository) UnreconcileMoveLines(
	ctx context.Context,
	reconciliationID uuid.UUID,
	date time.Time,
	actorID uuid.UUID,
) error {
	if reconciliationID == uuid.Nil || actorID == uuid.Nil {
		return errors.New("reconciliation id and actor are required")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin unreconciliation: %w", err)
	}
	defer tx.Rollback(ctx)

	var realizedMove pgtype.UUID
	var accountID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT account_id, realized_fx_move_id
		FROM account_reconciliations WHERE id = $1 FOR UPDATE`,
		reconciliationID,
	).Scan(&accountID, &realizedMove); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrReconciliationNotFound
		}
		return fmt.Errorf("lock reconciliation: %w", err)
	}
	const itemQuery = `
		SELECT move_line_id, allocated_amount
		FROM account_reconciliation_items
		WHERE reconciliation_id = $1
		ORDER BY move_line_id`
	rows, err := tx.Query(ctx, itemQuery, reconciliationID)
	if err != nil {
		return fmt.Errorf("query reconciliation allocations: %w", err)
	}
	type item struct {
		lineID uuid.UUID
		amount decimal.Decimal
	}
	items := make([]item, 0)
	for rows.Next() {
		var current item
		if err := rows.Scan(&current.lineID, &current.amount); err != nil {
			rows.Close()
			return fmt.Errorf("scan reconciliation allocation: %w", err)
		}
		items = append(items, current)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate reconciliation allocations: %w", err)
	}
	rows.Close()
	for _, current := range items {
		tag, err := tx.Exec(ctx, `
			UPDATE account_move_lines
			SET reconciled_amount = reconciled_amount - $2
			WHERE id = $1 AND reconciled_amount >= $2`,
			current.lineID, current.amount,
		)
		if err != nil {
			return fmt.Errorf("restore residual for line %s: %w", current.lineID, err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("reconciled amount for line %s is inconsistent", current.lineID)
		}
	}
	if realizedMove.Valid {
		if _, err := reversePostedMoveInTransaction(ctx, tx, uuid.UUID(realizedMove.Bytes), date, actorID); err != nil {
			return fmt.Errorf("reverse realized FX move: %w", err)
		}
	}
	payload, err := json.Marshal(map[string]any{
		"id": reconciliationID, "account_id": accountID, "action": "UNRECONCILE",
	})
	if err != nil {
		return fmt.Errorf("encode unreconciliation audit payload: %w", err)
	}
	if err := insertAuditLog(ctx, tx, &accounting.FinancialAuditLog{
		EntityType: "account_reconciliation", EntityID: reconciliationID,
		Action: "UNRECONCILE", ActorID: actorID, PayloadAfter: payload, Timestamp: time.Now().UTC(),
	}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM account_reconciliations WHERE id = $1`, reconciliationID); err != nil {
		return fmt.Errorf("delete reconciliation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit unreconciliation: %w", err)
	}
	return nil
}

func reversePostedMoveInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	originalMoveID uuid.UUID,
	date time.Time,
	actorID uuid.UUID,
) (uuid.UUID, error) {
	var state accounting.MoveState
	var journalID, periodID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT state, journal_id, period_id
		FROM account_moves WHERE id = $1 FOR UPDATE`,
		originalMoveID,
	).Scan(&state, &journalID, &periodID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, accounting.ErrMoveNotFound
		}
		return uuid.Nil, fmt.Errorf("lock realized FX move: %w", err)
	}
	if state != accounting.MoveStatePosted {
		return uuid.Nil, accounting.ErrCannotReverseUnpostedMove
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM account_moves WHERE reversed_move_id = $1)`, originalMoveID).Scan(&exists); err != nil {
		return uuid.Nil, fmt.Errorf("check FX move reversal: %w", err)
	}
	if exists {
		return uuid.Nil, accounting.ErrMoveAlreadyReversed
	}
	periodID, err := periodInTransaction(ctx, tx, date)
	if err != nil {
		return uuid.Nil, err
	}
	if err := lockOpenPeriod(ctx, tx, periodID, date); err != nil {
		return uuid.Nil, err
	}
	sequence, err := nextSequence(ctx, tx, journalID)
	if err != nil {
		return uuid.Nil, err
	}
	originalLines, err := loadMoveLines(ctx, tx, originalMoveID)
	if err != nil {
		return uuid.Nil, err
	}
	if len(originalLines) < 2 {
		return uuid.Nil, errors.New("realized FX move has fewer than two lines")
	}
	now, reversalID := time.Now().UTC(), uuid.New()
	reversalLines := make([]accounting.AccountMoveLine, len(originalLines))
	for i, line := range originalLines {
		reversalLines[i] = accounting.AccountMoveLine{
			ID: uuid.New(), MoveID: reversalID, AccountID: line.AccountID,
			PartnerID: line.PartnerID, Debit: line.Credit, Credit: line.Debit,
			AmountCurrency: line.AmountCurrency.Neg(), CurrencyID: line.CurrencyID,
			CostCenterID: line.CostCenterID, Date: date, CreatedAt: now,
		}
	}
	previousHash, err := previousPostedMoveHash(ctx, tx, journalID)
	if err != nil {
		return uuid.Nil, err
	}
	hash := hashMove(reversalID, sequence, date, journalID, reversalLines, previousHash)
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_moves (
			id, name, journal_id, period_id, date, state, ref, reversed_move_id,
			hash_signature, posted_at, posted_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'posted', $6, $7, $8, $9, $10, $9, $9)`,
		reversalID, sequence, journalID, periodID, date, "Reversal of "+originalMoveID.String(),
		originalMoveID, hash, now, actorID,
	); err != nil {
		return uuid.Nil, fmt.Errorf("insert realized FX reversal: %w", err)
	}
	if err := insertMoveLines(ctx, tx, reversalID, date, reversalLines); err != nil {
		return uuid.Nil, err
	}
	payload, err := json.Marshal(map[string]any{"move_id": reversalID, "reversed_move_id": originalMoveID})
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode FX reversal audit payload: %w", err)
	}
	if err := insertAuditLog(ctx, tx, &accounting.FinancialAuditLog{
		EntityType: "account_move", EntityID: reversalID, Action: "REVERSE",
		ActorID: actorID, PayloadAfter: payload, Timestamp: now,
	}); err != nil {
		return uuid.Nil, err
	}
	return reversalID, nil
}
