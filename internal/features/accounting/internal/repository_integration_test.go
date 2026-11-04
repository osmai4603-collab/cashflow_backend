package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cashflow_backend/internal/features/accounting"
	"cashflow_backend/migrations"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

func TestPostgresRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("ACCOUNTING_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set ACCOUNTING_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := newIntegrationPool(ctx, t, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	repository := NewPostgresRepository(pool)
	rates := NewPostgresFXRateProvider(pool)
	rate, err := rates.GetClosingRate(ctx, "USD", "SAR", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetClosingRate(USD/SAR): %v", err)
	}
	if !rate.Equal(decimal.RequireFromString("3.750000")) {
		t.Fatalf("USD/SAR rate = %s, want 3.750000", rate)
	}
	if _, err := rates.GetClosingRate(ctx, "EUR", "SAR", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, accounting.ErrExchangeRateMissing) {
		t.Fatalf("GetClosingRate(EUR/SAR) error = %v, want ErrExchangeRateMissing", err)
	}

	year := &accounting.FiscalYear{
		Year: 2026, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		State:   "open", CreatedAt: time.Now().UTC(),
	}
	period := &accounting.FiscalPeriod{
		PeriodNumber: 1, Name: "FY2026", StartDate: year.StartDate,
		EndDate: year.EndDate, State: accounting.PeriodStateOpen, CreatedAt: time.Now().UTC(),
	}
	if err := repository.CreateFiscalYear(ctx, year, []*accounting.FiscalPeriod{period}); err != nil {
		t.Fatalf("CreateFiscalYear(): %v", err)
	}

	debitAccount := integrationAccount("1000", "Cash")
	creditAccount := integrationAccount("2000", "Equity")
	if err := repository.CreateAccount(ctx, debitAccount); err != nil {
		t.Fatalf("CreateAccount(debit): %v", err)
	}
	if err := repository.CreateAccount(ctx, creditAccount); err != nil {
		t.Fatalf("CreateAccount(credit): %v", err)
	}

	journalID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO journals (id, code, name, type, sequence_prefix)
		VALUES ($1, 'GEN', 'General', 'general', 'GEN/')`, journalID); err != nil {
		t.Fatalf("insert test journal: %v", err)
	}
	subledgerJournalID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if _, err := pool.Exec(ctx, `
		INSERT INTO journals (id, code, name, type, sequence_prefix)
		VALUES ($1, 'SUB', 'Subledger', 'general', 'SUB/')`, subledgerJournalID); err != nil {
		t.Fatalf("insert subledger journal: %v", err)
	}
	costAccount := integrationAccount("501000", "Cost of goods sold")
	costAccount.Type = accounting.AccountTypeExpense
	inventoryAccount := integrationAccount("140000", "Inventory")
	if err := repository.CreateAccount(ctx, costAccount); err != nil {
		t.Fatalf("CreateAccount(cost): %v", err)
	}
	if err := repository.CreateAccount(ctx, inventoryAccount); err != nil {
		t.Fatalf("CreateAccount(inventory): %v", err)
	}
	parentID := debitAccount.ID
	creditAccount.ParentID = &parentID
	if err := repository.UpdateAccount(ctx, creditAccount); err != nil {
		t.Fatalf("UpdateAccount(set parent): %v", err)
	}
	parentID = creditAccount.ID
	debitAccount.ParentID = &parentID
	if err := repository.UpdateAccount(ctx, debitAccount); err == nil {
		t.Fatal("UpdateAccount() created a cycle in the account hierarchy")
	}
	debitAccount.ParentID = nil

	if _, err := pool.Exec(ctx, `
		INSERT INTO subledger_event_mappings (event_type, journal_id, debit_account_id, credit_account_id)
		VALUES ('goods_delivered', $1, $2, $3)`,
		subledgerJournalID, costAccount.ID, inventoryAccount.ID,
	); err != nil {
		t.Fatalf("insert subledger mapping: %v", err)
	}
	subledger := accounting.NewIngestSubledgerEventUseCase(repository, nil)
	eventRequest := accounting.IngestSubledgerEventRequest{
		IdempotencyKey: "integration-event-1", EventType: "goods_delivered",
		Date: "2026-11-03", Amount: decimal.NewFromInt(15), Currency: "SAR",
	}
	firstEventMove, err := subledger.Execute(ctx, eventRequest)
	if err != nil {
		t.Fatalf("first subledger event: %v", err)
	}
	secondEventMove, err := subledger.Execute(ctx, eventRequest)
	if err != nil {
		t.Fatalf("duplicate subledger event: %v", err)
	}
	if firstEventMove.ID != secondEventMove.ID {
		t.Fatalf("duplicate event returned move %s, want cached move %s", secondEventMove.ID, firstEventMove.ID)
	}
	var subledgerNextNumber int64
	if err := pool.QueryRow(ctx, `SELECT next_number FROM journals WHERE id = $1`, subledgerJournalID).Scan(&subledgerNextNumber); err != nil {
		t.Fatalf("get subledger sequence: %v", err)
	}
	if subledgerNextNumber != 2 {
		t.Fatalf("duplicate event advanced subledger sequence to %d, want 2", subledgerNextNumber)
	}

	priorDate := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	reportDate := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	priorMove := integrationMove(journalID, period.ID, priorDate, debitAccount.ID, creditAccount.ID, 100)
	postIntegrationMove(t, ctx, repository, priorMove)
	periodMove := integrationMove(journalID, period.ID, reportDate, debitAccount.ID, creditAccount.ID, 20)
	postIntegrationMove(t, ctx, repository, periodMove)

	staleDraft := integrationMove(journalID, period.ID, reportDate, debitAccount.ID, creditAccount.ID, 3)
	if err := repository.SaveDraftMove(ctx, staleDraft); err != nil {
		t.Fatalf("SaveDraftMove(stale case): %v", err)
	}
	staleSnapshot, err := repository.GetMoveByID(ctx, staleDraft.ID)
	if err != nil {
		t.Fatalf("GetMoveByID(stale case): %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE account_moves SET updated_at = updated_at + INTERVAL '1 second' WHERE id = $1`,
		staleDraft.ID,
	); err != nil {
		t.Fatalf("change draft timestamp: %v", err)
	}
	if err := repository.PostMoveAtomic(ctx, staleSnapshot, integrationAudit(staleDraft.ID)); !errors.Is(err, accounting.ErrMoveChanged) {
		t.Fatalf("PostMoveAtomic(stale draft) error = %v, want ErrMoveChanged", err)
	}

	const concurrentMoves = 50
	moves := make([]*accounting.AccountMove, concurrentMoves)
	for i := range moves {
		moves[i] = integrationMove(journalID, period.ID, reportDate, debitAccount.ID, creditAccount.ID, 1)
		if err := repository.SaveDraftMove(ctx, moves[i]); err != nil {
			t.Fatalf("SaveDraftMove(%d): %v", i, err)
		}
	}

	errs := make(chan error, concurrentMoves)
	var workers sync.WaitGroup
	for _, move := range moves {
		workers.Add(1)
		go func(move *accounting.AccountMove) {
			defer workers.Done()
			audit := integrationAudit(move.ID)
			errs <- repository.PostMoveAtomic(ctx, move, audit)
		}(move)
	}
	workers.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent PostMoveAtomic(): %v", err)
		}
	}

	reversal := &accounting.AccountMove{
		ID: uuid.New(), JournalID: journalID, PeriodID: period.ID,
		Date: reportDate, State: accounting.MoveStateDraft,
		Ref: "integration reversal",
	}
	if err := repository.ReverseMoveAtomic(ctx, moves[0].ID, reversal, integrationAudit(reversal.ID)); err != nil {
		t.Fatalf("ReverseMoveAtomic(): %v", err)
	}
	if err := repository.ReverseMoveAtomic(ctx, moves[0].ID, &accounting.AccountMove{
		JournalID: journalID, PeriodID: period.ID, Date: reportDate, State: accounting.MoveStateDraft,
	}, nil); !errors.Is(err, accounting.ErrMoveAlreadyReversed) {
		t.Fatalf("second reversal error = %v, want ErrMoveAlreadyReversed", err)
	}

	if valid, err := repository.VerifyAuditIntegrity(ctx); err != nil || !valid {
		t.Fatalf("VerifyAuditIntegrity() = %v, %v; want a valid chain", valid, err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE financial_audit_logs SET payload_after = '{"event":"tampered"}'::jsonb
		WHERE entity_id = $1 AND action = 'POST'`, moves[0].ID); err != nil {
		t.Fatalf("tamper with test audit payload: %v", err)
	}
	if valid, err := repository.VerifyAuditIntegrity(ctx); err == nil || valid {
		t.Fatalf("VerifyAuditIntegrity() after payload tampering = %v, %v; want integrity error", valid, err)
	}

	assertGaplessPostedNames(t, ctx, pool, concurrentMoves+3)
	var nextNumber int64
	if err := pool.QueryRow(ctx, `SELECT next_number FROM journals WHERE id = $1`, journalID).Scan(&nextNumber); err != nil {
		t.Fatal(err)
	}
	if nextNumber != concurrentMoves+4 {
		t.Fatalf("next_number = %d, want %d", nextNumber, concurrentMoves+4)
	}

	failedMove := integrationMove(journalID, period.ID, reportDate, debitAccount.ID, creditAccount.ID, 5)
	if err := repository.SaveDraftMove(ctx, failedMove); err != nil {
		t.Fatalf("SaveDraftMove(failure case): %v", err)
	}
	failedAudit := integrationAudit(failedMove.ID)
	failedAudit.IPAddress = "not-an-ip"
	if err := repository.PostMoveAtomic(ctx, failedMove, failedAudit); err == nil {
		t.Fatal("PostMoveAtomic() succeeded with an invalid audit IP")
	}
	if failedMove.State != accounting.MoveStateDraft || failedMove.Name != "" {
		t.Fatalf("failed transaction mutated draft move: %+v", failedMove)
	}
	if err := pool.QueryRow(ctx, `SELECT next_number FROM journals WHERE id = $1`, journalID).Scan(&nextNumber); err != nil {
		t.Fatal(err)
	}
	if nextNumber != concurrentMoves+4 {
		t.Fatalf("failed transaction consumed a sequence number: next_number = %d", nextNumber)
	}
	var failedState accounting.MoveState
	if err := pool.QueryRow(ctx, `SELECT state FROM account_moves WHERE id = $1`, failedMove.ID).Scan(&failedState); err != nil {
		t.Fatal(err)
	}
	if failedState != accounting.MoveStateDraft {
		t.Fatalf("database state after rollback = %s, want draft", failedState)
	}
	var failedAuditCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM financial_audit_logs WHERE entity_id = $1`, failedMove.ID,
	).Scan(&failedAuditCount); err != nil {
		t.Fatal(err)
	}
	if failedAuditCount != 0 {
		t.Fatalf("failed transaction persisted %d audit rows", failedAuditCount)
	}

	fromDate := reportDate
	toDate := reportDate
	trialBalance, err := repository.GetTrialBalanceData(ctx, fromDate, toDate)
	if err != nil {
		t.Fatalf("GetTrialBalanceData(): %v", err)
	}
	assertTrialBalanceRow(t, trialBalance, debitAccount.ID, "100", "0", "70", "1", "170", "1")
	assertTrialBalanceRow(t, trialBalance, creditAccount.ID, "0", "100", "1", "70", "1", "170")

	ledger, err := repository.GetGeneralLedgerLines(ctx, debitAccount.ID, fromDate, toDate)
	if err != nil {
		t.Fatalf("GetGeneralLedgerLines(): %v", err)
	}
	if len(ledger) != concurrentMoves+2 {
		t.Fatalf("general ledger has %d lines, want %d", len(ledger), concurrentMoves+2)
	}
	if got := ledger[0].RunningBalance.String(); got != "120" {
		t.Fatalf("first period running balance = %s, want 120", got)
	}
	if got := ledger[len(ledger)-1].RunningBalance.String(); got != "169" {
		t.Fatalf("ending running balance = %s, want 169", got)
	}
	if err := repository.UpdatePeriodState(ctx, period.ID, accounting.PeriodStateSoftClosed, uuid.New()); err != nil {
		t.Fatalf("UpdatePeriodState(open to soft closed): %v", err)
	}
	if err := repository.UpdatePeriodState(ctx, period.ID, accounting.PeriodStateHardClosed, uuid.New()); err != nil {
		t.Fatalf("UpdatePeriodState(soft to hard closed): %v", err)
	}
	if err := repository.UpdatePeriodState(ctx, period.ID, accounting.PeriodStateOpen, uuid.New()); !errors.Is(err, accounting.ErrInvalidPeriodTransition) {
		t.Fatalf("UpdatePeriodState(hard closed to open) error = %v, want ErrInvalidPeriodTransition", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fiscal_years SET state = 'closed' WHERE id = $1`, year.ID); err != nil {
		t.Fatalf("close fiscal year fixture: %v", err)
	}
	if err := repository.UpdatePeriodState(ctx, period.ID, accounting.PeriodStateOpen, uuid.New()); !errors.Is(err, accounting.ErrPeriodClosed) {
		t.Fatalf("UpdatePeriodState() for a closed year error = %v, want ErrPeriodClosed", err)
	}
}

func newIntegrationPool(ctx context.Context, t *testing.T, databaseURL string) (*pgxpool.Pool, error) {
	t.Helper()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to integration database: %w", err)
	}
	schema := "accounting_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		adminPool.Close()
		return nil, fmt.Errorf("create isolated integration schema: %w", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		return nil, fmt.Errorf("parse integration database URL: %w", err)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	config.MaxConns = 60
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		return nil, fmt.Errorf("create isolated integration pool: %w", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, dropErr := adminPool.Exec(context.Background(), "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		if dropErr != nil {
			t.Errorf("drop isolated integration schema: %v", dropErr)
		}
	})

	for _, name := range []string{
		"000002_create_accounting_schema.up.sql",
		"000003_accounting_governance.up.sql",
		"000004_accounting_controls.up.sql",
	} {
		migration, err := migrations.FS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		for _, statement := range strings.Split(string(migration), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := pool.Exec(ctx, statement); err != nil {
				return nil, fmt.Errorf("apply migration %s: %w", name, err)
			}
		}
	}
	return pool, nil
}

func integrationAccount(code, name string) *accounting.Account {
	now := time.Now().UTC()
	return &accounting.Account{
		ID: uuid.New(), Code: code, Name: name,
		Type: accounting.AccountTypeAsset, NormalBalance: accounting.NormalBalanceDebit,
		CurrencyID: "SAR", IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
}

func integrationMove(journalID, periodID uuid.UUID, date time.Time, debitAccountID, creditAccountID uuid.UUID, amount int64) *accounting.AccountMove {
	moveID := uuid.New()
	now := time.Now().UTC()
	amountValue := decimal.NewFromInt(amount)
	return &accounting.AccountMove{
		ID: moveID, JournalID: journalID, PeriodID: periodID, Date: date,
		State: accounting.MoveStateDraft, CreatedAt: now,
		Lines: []accounting.AccountMoveLine{
			{ID: uuid.New(), MoveID: moveID, AccountID: debitAccountID, Debit: amountValue, Credit: decimal.Zero, AmountCurrency: amountValue, CurrencyID: "SAR", Date: date, CreatedAt: now},
			{ID: uuid.New(), MoveID: moveID, AccountID: creditAccountID, Debit: decimal.Zero, Credit: amountValue, AmountCurrency: amountValue.Neg(), CurrencyID: "SAR", Date: date, CreatedAt: now},
		},
	}
}

func postIntegrationMove(t *testing.T, ctx context.Context, repository *PostgresRepository, move *accounting.AccountMove) {
	t.Helper()
	if err := repository.SaveDraftMove(ctx, move); err != nil {
		t.Fatalf("SaveDraftMove(): %v", err)
	}
	if err := repository.PostMoveAtomic(ctx, move, integrationAudit(move.ID)); err != nil {
		t.Fatalf("PostMoveAtomic(): %v", err)
	}
}

func integrationAudit(moveID uuid.UUID) *accounting.FinancialAuditLog {
	return &accounting.FinancialAuditLog{
		EntityType: "account_move", EntityID: moveID, Action: "POST",
		ActorID: uuid.New(), PayloadAfter: []byte(`{"event":"posted"}`), Timestamp: time.Now().UTC(),
	}
}

func assertGaplessPostedNames(t *testing.T, ctx context.Context, pool *pgxpool.Pool, wantCount int) {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT name FROM account_moves WHERE state = 'posted' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		count++
		want := fmt.Sprintf("GEN/%05d", count)
		if name != want {
			t.Fatalf("posted sequence at position %d = %q, want %q", count, name, want)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != wantCount {
		t.Fatalf("posted moves = %d, want %d", count, wantCount)
	}
}

func assertTrialBalanceRow(t *testing.T, rows []*accounting.TrialBalanceRow, accountID uuid.UUID, initialDebit, initialCredit, periodDebit, periodCredit, endingDebit, endingCredit string) {
	t.Helper()
	for _, row := range rows {
		if row.AccountID != accountID {
			continue
		}
		got := []string{
			row.InitialDebit.String(), row.InitialCredit.String(),
			row.PeriodDebit.String(), row.PeriodCredit.String(),
			row.EndingDebit.String(), row.EndingCredit.String(),
		}
		want := []string{initialDebit, initialCredit, periodDebit, periodCredit, endingDebit, endingCredit}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("trial balance for %s = %v, want %v", accountID, got, want)
		}
		return
	}
	t.Fatalf("trial balance missing account %s", accountID)
}
