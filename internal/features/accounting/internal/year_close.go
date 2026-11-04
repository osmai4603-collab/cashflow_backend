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
	"github.com/shopspring/decimal"
)

type fiscalPeriodRecord struct {
	id        uuid.UUID
	number    int
	startDate time.Time
	endDate   time.Time
	state     accounting.PeriodState
}

func (r *PostgresRepository) CloseFiscalYear(
	ctx context.Context,
	fiscalYearID, journalID, retainedAccountID, actorID uuid.UUID,
) (*accounting.YearCloseResult, error) {
	if fiscalYearID == uuid.Nil || journalID == uuid.Nil || actorID == uuid.Nil {
		return nil, errors.New("fiscal year, closing journal, and authenticated actor are required")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin fiscal year close: %w", err)
	}
	defer tx.Rollback(ctx)

	var year accounting.FiscalYear
	if err := tx.QueryRow(ctx, `
		SELECT id, year, start_date, end_date, state
		FROM fiscal_years WHERE id = $1 FOR UPDATE`,
		fiscalYearID,
	).Scan(&year.ID, &year.Year, &year.StartDate, &year.EndDate, &year.State); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, accounting.ErrFiscalYearNotFound
		}
		return nil, fmt.Errorf("lock fiscal year: %w", err)
	}
	if year.State != "open" {
		return nil, accounting.ErrFiscalYearNotClosable
	}
	periods, err := fiscalYearPeriods(ctx, tx, fiscalYearID)
	if err != nil {
		return nil, err
	}
	if len(periods) != 12 {
		return nil, fmt.Errorf("%w: fiscal year must have exactly twelve periods", accounting.ErrFiscalYearNotClosable)
	}
	var closingPeriod *fiscalPeriodRecord
	for i := range periods {
		if periods[i].number != i+1 || periods[i].state != accounting.PeriodStateHardClosed {
			return nil, fmt.Errorf("%w: all twelve fiscal periods must be hard closed in sequence", accounting.ErrFiscalYearNotClosable)
		}
		if periods[i].number == 12 {
			closingPeriod = &periods[i]
		}
	}
	if closingPeriod == nil || !sameDate(closingPeriod.endDate, year.EndDate) {
		return nil, fmt.Errorf("%w: final period does not end on the fiscal year end date", accounting.ErrFiscalYearNotClosable)
	}
	var hasDraftMoves bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM account_moves
			WHERE date >= $1 AND date <= $2 AND state = 'draft'
		)`, year.StartDate, year.EndDate).Scan(&hasDraftMoves); err != nil {
		return nil, fmt.Errorf("check fiscal year draft moves: %w", err)
	}
	if hasDraftMoves {
		return nil, fmt.Errorf("%w: draft moves remain in the fiscal year", accounting.ErrFiscalYearNotClosable)
	}
	var nextYearID uuid.UUID
	var nextStart time.Time
	if err := tx.QueryRow(ctx, `
		SELECT id, start_date
		FROM fiscal_years
		WHERE year = $1 AND state = 'open'
		FOR UPDATE`, year.Year+1,
	).Scan(&nextYearID, &nextStart); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: the following fiscal year must be configured first", accounting.ErrFiscalYearNotClosable)
		}
		return nil, fmt.Errorf("lock successor fiscal year: %w", err)
	}
	if !nextStart.After(year.EndDate) || !sameDate(nextStart, year.EndDate.AddDate(0, 0, 1)) {
		return nil, fmt.Errorf("%w: successor fiscal year must begin the day after this year ends", accounting.ErrFiscalYearNotClosable)
	}
	var openingPeriodID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id FROM fiscal_periods
		WHERE fiscal_year_id = $1 AND period_number = 1
			AND start_date <= $2 AND end_date >= $2 AND state = 'open'
		FOR SHARE`, nextYearID, nextStart,
	).Scan(&openingPeriodID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: successor year opening period must be open", accounting.ErrFiscalYearNotClosable)
		}
		return nil, fmt.Errorf("lock successor opening period: %w", err)
	}

	if retainedAccountID == uuid.Nil {
		if err := tx.QueryRow(ctx, `
			SELECT retained_earnings_account_id
			FROM accounting_configurations
			ORDER BY updated_at DESC LIMIT 1`,
		).Scan(&retainedAccountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, accounting.ErrAccountingConfigurationMissing
			}
			return nil, fmt.Errorf("load retained earnings account: %w", err)
		}
	}
	if err := validateClosingSetup(ctx, tx, journalID, retainedAccountID); err != nil {
		return nil, err
	}
	var functionalCurrency string
	if err := tx.QueryRow(ctx, `
		SELECT functional_currency FROM accounting_configurations
		ORDER BY updated_at DESC LIMIT 1`,
	).Scan(&functionalCurrency); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, accounting.ErrAccountingConfigurationMissing
		}
		return nil, fmt.Errorf("load functional currency: %w", err)
	}

	result := &accounting.YearCloseResult{SuccessorYearID: nextYearID}
	closingLines, err := fiscalClosingLines(ctx, tx, year.EndDate, retainedAccountID, functionalCurrency)
	if err != nil {
		return nil, err
	}
	if len(closingLines) > 0 {
		moveID, err := insertSystemMove(ctx, tx, journalID, closingPeriod.id, year.EndDate,
			actorID, "Year-end profit and loss close", closingLines)
		if err != nil {
			return nil, err
		}
		result.ClosingMoveID = &moveID
	}
	openingLines, err := fiscalOpeningLines(ctx, tx, year.EndDate, functionalCurrency)
	if err != nil {
		return nil, err
	}
	if len(openingLines) > 0 {
		moveID, err := insertSystemMove(ctx, tx, journalID, openingPeriodID, nextStart,
			actorID, "Fiscal year opening balances", openingLines)
		if err != nil {
			return nil, err
		}
		result.OpeningMoveID = &moveID
	}
	payload, err := json.Marshal(map[string]any{
		"fiscal_year_id": fiscalYearID, "successor_year_id": nextYearID,
		"closing_move_id": result.ClosingMoveID, "opening_move_id": result.OpeningMoveID,
	})
	if err != nil {
		return nil, fmt.Errorf("encode fiscal close audit payload: %w", err)
	}
	if err := insertAuditLog(ctx, tx, &accounting.FinancialAuditLog{
		EntityType: "fiscal_year", EntityID: fiscalYearID, Action: "CLOSE",
		ActorID: actorID, PayloadAfter: payload, Timestamp: time.Now().UTC(),
	}); err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, `UPDATE fiscal_years SET state = 'closed' WHERE id = $1 AND state = 'open'`, fiscalYearID)
	if err != nil {
		return nil, fmt.Errorf("mark fiscal year closed: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return nil, accounting.ErrFiscalYearNotClosable
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit fiscal year close: %w", err)
	}
	return result, nil
}

func fiscalYearPeriods(ctx context.Context, tx pgx.Tx, yearID uuid.UUID) ([]fiscalPeriodRecord, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, period_number, start_date, end_date, state
		FROM fiscal_periods WHERE fiscal_year_id = $1
		ORDER BY period_number FOR UPDATE`, yearID)
	if err != nil {
		return nil, fmt.Errorf("lock fiscal periods: %w", err)
	}
	defer rows.Close()
	periods := make([]fiscalPeriodRecord, 0, 12)
	for rows.Next() {
		var period fiscalPeriodRecord
		if err := rows.Scan(&period.id, &period.number, &period.startDate, &period.endDate, &period.state); err != nil {
			return nil, fmt.Errorf("scan fiscal period: %w", err)
		}
		periods = append(periods, period)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fiscal periods: %w", err)
	}
	return periods, nil
}

func validateClosingSetup(ctx context.Context, tx pgx.Tx, journalID, retainedAccountID uuid.UUID) error {
	var journalActive bool
	if err := tx.QueryRow(ctx, `SELECT is_active FROM journals WHERE id = $1`, journalID).Scan(&journalActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrJournalNotFound
		}
		return fmt.Errorf("load year-end journal: %w", err)
	}
	if !journalActive {
		return errors.New("year-end closing journal must be active")
	}
	var accountType accounting.AccountType
	var active bool
	if err := tx.QueryRow(ctx, `SELECT account_type, is_active FROM accounts WHERE id = $1`, retainedAccountID).Scan(&accountType, &active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrAccountNotFound
		}
		return fmt.Errorf("load retained earnings account: %w", err)
	}
	if !active || accountType != accounting.AccountTypeEquity {
		return errors.New("retained earnings account must be active and have equity type")
	}
	return nil
}

func fiscalClosingLines(
	ctx context.Context,
	tx pgx.Tx,
	asOf time.Time,
	retainedAccountID uuid.UUID,
	functionalCurrency string,
) ([]accounting.AccountMoveLine, error) {
	rows, err := tx.Query(ctx, `
		SELECT l.account_id, SUM(l.debit - l.credit)
		FROM account_move_lines l
		JOIN account_moves m ON m.id = l.move_id AND m.state = 'posted'
		JOIN accounts a ON a.id = l.account_id
		WHERE m.date <= $1 AND a.account_type IN ('income', 'expense')
		GROUP BY l.account_id, a.account_type
		ORDER BY l.account_id`, asOf)
	if err != nil {
		return nil, fmt.Errorf("query income and expense balances for closing: %w", err)
	}
	defer rows.Close()
	now := time.Now().UTC()
	lines := make([]accounting.AccountMoveLine, 0)
	var retainedDebitMinusCredit decimal.Decimal
	for rows.Next() {
		var accountID uuid.UUID
		var balance decimal.Decimal
		if err := rows.Scan(&accountID, &balance); err != nil {
			return nil, fmt.Errorf("scan income and expense balance: %w", err)
		}
		if balance.IsZero() {
			continue
		}
		line := accounting.AccountMoveLine{
			ID: uuid.New(), AccountID: accountID, CurrencyID: functionalCurrency,
			Date: asOf, CreatedAt: now,
		}
		if balance.IsPositive() {
			line.Credit = balance
		} else {
			line.Debit = balance.Abs()
		}
		retainedDebitMinusCredit = retainedDebitMinusCredit.Add(balance)
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate income and expense balances: %w", err)
	}
	if !retainedDebitMinusCredit.IsZero() {
		line := accounting.AccountMoveLine{
			ID: uuid.New(), AccountID: retainedAccountID, CurrencyID: functionalCurrency,
			Date: asOf, CreatedAt: now,
		}
		if retainedDebitMinusCredit.IsPositive() {
			line.Debit = retainedDebitMinusCredit
		} else {
			line.Credit = retainedDebitMinusCredit.Abs()
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func fiscalOpeningLines(
	ctx context.Context,
	tx pgx.Tx,
	asOf time.Time,
	functionalCurrency string,
) ([]accounting.AccountMoveLine, error) {
	rows, err := tx.Query(ctx, `
		SELECT l.account_id, l.currency_id,
			SUM(l.debit - l.credit), SUM(COALESCE(l.amount_currency, 0))
		FROM account_move_lines l
		JOIN account_moves m ON m.id = l.move_id AND m.state = 'posted'
		JOIN accounts a ON a.id = l.account_id
		WHERE m.date <= $1 AND a.account_type IN ('asset', 'liability', 'equity')
		GROUP BY l.account_id, a.account_type, l.currency_id
		HAVING SUM(l.debit - l.credit) <> 0 OR SUM(COALESCE(l.amount_currency, 0)) <> 0
		ORDER BY l.account_id, l.currency_id`, asOf)
	if err != nil {
		return nil, fmt.Errorf("query balance sheet balances for opening: %w", err)
	}
	defer rows.Close()
	now := time.Now().UTC()
	lines := make([]accounting.AccountMoveLine, 0)
	var totalDebit, totalCredit decimal.Decimal
	for rows.Next() {
		var accountID uuid.UUID
		var currency string
		var net, amountCurrency decimal.Decimal
		if err := rows.Scan(&accountID, &currency, &net, &amountCurrency); err != nil {
			return nil, fmt.Errorf("scan balance sheet opening balance: %w", err)
		}
		if currency != functionalCurrency && amountCurrency.IsZero() && !net.IsZero() {
			return nil, fmt.Errorf("account %s has a non-functional balance without a foreign amount", accountID)
		}
		line := accounting.AccountMoveLine{
			ID: uuid.New(), AccountID: accountID, AmountCurrency: amountCurrency,
			CurrencyID: currency, Date: asOf.AddDate(0, 0, 1), CreatedAt: now,
		}
		if net.IsPositive() {
			line.Debit = net
			totalDebit = totalDebit.Add(net)
		} else {
			line.Credit = net.Abs()
			totalCredit = totalCredit.Add(net.Abs())
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balance sheet opening balances: %w", err)
	}
	if !totalDebit.Equal(totalCredit) {
		return nil, fmt.Errorf("opening balance lines are not balanced: debit %s != credit %s", totalDebit, totalCredit)
	}
	return lines, nil
}

func insertSystemMove(
	ctx context.Context,
	tx pgx.Tx,
	journalID, periodID uuid.UUID,
	date time.Time,
	actorID uuid.UUID,
	ref string,
	lines []accounting.AccountMoveLine,
) (uuid.UUID, error) {
	moveID := uuid.New()
	now := time.Now().UTC()
	for i := range lines {
		lines[i].MoveID = moveID
		if lines[i].Date.IsZero() {
			lines[i].Date = date
		}
		if lines[i].CreatedAt.IsZero() {
			lines[i].CreatedAt = now
		}
	}
	move := &accounting.AccountMove{ID: moveID, Name: "", JournalID: journalID, PeriodID: periodID, Date: date, Lines: lines}
	if err := move.ValidateBalance(); err != nil {
		return uuid.Nil, fmt.Errorf("validate system move: %w", err)
	}
	sequence, err := nextSequence(ctx, tx, journalID)
	if err != nil {
		return uuid.Nil, err
	}
	previousHash, err := previousPostedMoveHash(ctx, tx, journalID)
	if err != nil {
		return uuid.Nil, err
	}
	hash := hashMove(moveID, sequence, date, journalID, lines, previousHash)
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_moves (
			id, name, journal_id, period_id, date, state, ref, hash_signature, posted_at,
			posted_by, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'posted', $6, $7, $8, $9, $9, $8, $8)`,
		moveID, sequence, journalID, periodID, date, ref, hash, now, actorID,
	); err != nil {
		return uuid.Nil, fmt.Errorf("insert system move: %w", err)
	}
	if err := insertMoveLines(ctx, tx, moveID, date, lines); err != nil {
		return uuid.Nil, err
	}
	payload, err := json.Marshal(map[string]any{"move_id": moveID, "name": sequence, "ref": ref})
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode system move audit payload: %w", err)
	}
	if err := insertAuditLog(ctx, tx, &accounting.FinancialAuditLog{
		EntityType: "account_move", EntityID: moveID, Action: "POST",
		ActorID: actorID, PayloadAfter: payload, Timestamp: now,
	}); err != nil {
		return uuid.Nil, err
	}
	return moveID, nil
}
