package bankstatementusecase_test

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"testing"
	"time"

	bankstatementstorage "cashflow_backend/internal/adapters/storage/bankstatement"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/bankstatement"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	bankstatementusecase "cashflow_backend/internal/usecase/bankstatement"
)

func int64Ptr(v int64) *int64 { return &v }

// ─────────────────────────────────────────────────────────────────────────────
// Fake AccountingService
// ─────────────────────────────────────────────────────────────────────────────

type fakeAccounting struct {
	journal        *accounting.Journal
	accounts       map[int64]*accounting.Account
	moves          map[int64]*accounting.AccountMove
	lines          map[int64]*accounting.AccountMoveLine
	nextMoveID     int64
	nextLineID     int64
	paymentUpdates []paymentUpdate
}

type paymentUpdate struct {
	moveID   int64
	state    accounting.PaymentState
	residual float64
}

func newFakeAccounting() *fakeAccounting {
	bank := int64Ptr(2)
	fa := &fakeAccounting{
		journal: &accounting.Journal{
			ID: 1, Name: "Bank", Code: "BNK1", Type: accounting.JournalTypeBank,
			DefaultAccountID: bank, SuspenseAccountID: bank,
		},
		accounts: map[int64]*accounting.Account{
			2: {ID: 2, Code: "102000", Name: "Bank Account"},
			3: {ID: 3, Code: "120000", Name: "Accounts Receivable", Type: accounting.AccountTypeAssetReceivable, Reconcile: true},
			6: {ID: 6, Code: "210000", Name: "Accounts Payable", Type: accounting.AccountTypeLiabilityPayable, Reconcile: true},
		},
		moves: make(map[int64]*accounting.AccountMove),
		lines: make(map[int64]*accounting.AccountMoveLine),
	}
	fa.nextMoveID = 100
	fa.nextLineID = 10
	return fa
}

// seedInvoice creates a posted invoice move with a single reconcilable receivable/payable line.
func (fa *fakeAccounting) seedInvoice(moveID, lineID int64, accountID int64, balance float64, partnerID *int64) {
	fa.moves[moveID] = &accounting.AccountMove{
		ID: moveID, Name: "INV/2026/00001", MoveType: accounting.MoveTypeOutInvoice,
		JournalID: 1, Date: time.Now(), State: accounting.MoveStatePosted,
		PaymentState: accounting.PaymentStateNotPaid, AmountTotal: math.Abs(balance),
		AmountResidual: math.Abs(balance), Active: true,
	}
	debit, credit := 0.0, 0.0
	if balance > 0 {
		debit = balance
	} else {
		credit = -balance
	}
	fa.lines[lineID] = &accounting.AccountMoveLine{
		ID: lineID, MoveID: moveID, AccountID: accountID, PartnerID: partnerID,
		Name: "Invoice", Debit: debit, Credit: credit, Balance: balance,
		Reconcile: true, Reconciled: false, AmountResidual: math.Abs(balance),
	}
	fa.moves[moveID].Lines = []accounting.AccountMoveLine{*fa.lines[lineID]}
}

func (fa *fakeAccounting) CreateJournalEntry(_ context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error) {
	fa.nextMoveID++
	moveID := fa.nextMoveID
	move := &accounting.AccountMove{
		ID: moveID, Name: "/", MoveType: accounting.MoveTypeEntry,
		JournalID: in.JournalID, Date: in.Date, State: accounting.MoveStateDraft,
		PaymentState: accounting.PaymentStateNotPaid, Active: true,
	}
	for _, l := range in.Lines {
		fa.nextLineID++
		line := &accounting.AccountMoveLine{
			ID: fa.nextLineID, MoveID: moveID, AccountID: l.AccountID, PartnerID: l.PartnerID,
			ProductID: l.ProductID, Name: l.Name, Debit: l.Debit, Credit: l.Credit,
			Balance: l.Debit - l.Credit, Reconcile: true, Reconciled: false,
			AmountResidual: math.Abs(l.Debit - l.Credit), StatementLineID: l.StatementLineID,
		}
		move.Lines = append(move.Lines, *line)
		fa.lines[line.ID] = line
	}
	move.AmountTotal = move.TotalDebit()
	fa.moves[moveID] = move
	return move, nil
}

func (fa *fakeAccounting) PostMove(_ context.Context, id int64) (*accounting.AccountMove, error) {
	m := fa.moves[id]
	m.State = accounting.MoveStatePosted
	m.Name = "BNK1/2026/00001"
	return m, nil
}

func (fa *fakeAccounting) GetMove(_ context.Context, id int64) (*accounting.AccountMove, error) {
	m := fa.moves[id]
	clone := *m
	clone.Lines = make([]accounting.AccountMoveLine, 0, len(m.Lines))
	for _, l := range m.Lines {
		clone.Lines = append(clone.Lines, *fa.lines[l.ID])
	}
	return &clone, nil
}

func (fa *fakeAccounting) GetJournal(_ context.Context, id int64) (*accounting.Journal, error) {
	return fa.journal, nil
}

func (fa *fakeAccounting) GetAccount(_ context.Context, id int64) (*accounting.Account, error) {
	return fa.accounts[id], nil
}

func (fa *fakeAccounting) GetAccountByCode(_ context.Context, code string) (*accounting.Account, error) {
	for _, a := range fa.accounts {
		if a.Code == code {
			return a, nil
		}
	}
	return nil, errors.New("account not found")
}

func (fa *fakeAccounting) UpdatePaymentStatus(_ context.Context, id int64, state accounting.PaymentState, residual float64) error {
	fa.paymentUpdates = append(fa.paymentUpdates, paymentUpdate{moveID: id, state: state, residual: residual})
	if m, ok := fa.moves[id]; ok {
		m.PaymentState = state
		m.AmountResidual = residual
	}
	return nil
}

func (fa *fakeAccounting) GetMoveLine(_ context.Context, id int64) (*accounting.AccountMoveLine, error) {
	return fa.lines[id], nil
}

func (fa *fakeAccounting) UpdateMoveLineReconcile(_ context.Context, id int64, reconciled bool, residual float64, matchingNumber *string) error {
	if l, ok := fa.lines[id]; ok {
		l.Reconciled = reconciled
		l.AmountResidual = residual
		l.MatchingNumber = matchingNumber
	}
	return nil
}

func (fa *fakeAccounting) ListReconcilableMoveLines(_ context.Context, partnerID *int64, excludeLineIDs []int64, limit int) ([]accounting.AccountMoveLine, error) {
	exclude := make(map[int64]struct{}, len(excludeLineIDs))
	for _, id := range excludeLineIDs {
		exclude[id] = struct{}{}
	}
	var result []accounting.AccountMoveLine
	for _, l := range fa.lines {
		if _, skip := exclude[l.ID]; skip {
			continue
		}
		m := fa.moves[l.MoveID]
		if m == nil || m.State != accounting.MoveStatePosted || !m.Active {
			continue
		}
		if l.StatementLineID != nil {
			continue
		}
		if !l.Reconcile || l.Reconciled || l.AmountResidual <= 0.004 {
			continue
		}
		if partnerID != nil && (l.PartnerID == nil || *l.PartnerID != *partnerID) {
			continue
		}
		result = append(result, *l)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

type staticSequence struct{ name string }

func (s staticSequence) NextStatementName(_ context.Context, _ string, _ int) (string, error) {
	return s.name, nil
}

func newUseCase() (*bankstatementusecase.UseCase, *fakeAccounting, bankstatement.Repository) {
	fa := newFakeAccounting()
	repo := bankstatementstorage.NewMemoryRepo()
	uc := bankstatementusecase.New(repo, fa, staticSequence{name: "BNK1 Statement 2026"}, slog.New(slog.DiscardHandler))
	return uc, fa, repo
}

// ─────────────────────────────────────────────────────────────────────────────
// Statement lifecycle & booking
// ─────────────────────────────────────────────────────────────────────────────

func TestCreateStatementBooksAndPostsLines(t *testing.T) {
	uc, fa, repo := newUseCase()
	date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	st, err := uc.CreateStatement(context.Background(), bankstatementusecase.CreateStatementInput{
		JournalID: 1,
		Date:      date,
		Currency:  "USD",
		Lines: []bankstatementusecase.AddStatementLineInput{
			{Name: "Customer payment", Amount: 100, PartnerID: int64Ptr(5), Date: date},
			{Name: "Supplier payment", Amount: -40, PartnerID: int64Ptr(7), Date: date},
		},
	})
	if err != nil {
		t.Fatalf("CreateStatement failed: %v", err)
	}
	if st.ID == 0 || len(st.Lines) != 2 {
		t.Fatalf("unexpected statement: %+v", st)
	}
	for _, l := range st.Lines {
		if l.MoveID == nil || *l.MoveID <= 0 {
			t.Fatalf("line %d was not booked", l.ID)
		}
	}

	// The liquidity line of each booked move must reference the statement line.
	for _, l := range st.Lines {
		move, err := fa.GetMove(context.Background(), *l.MoveID)
		if err != nil {
			t.Fatalf("failed to load move: %v", err)
		}
		foundLiquidity := false
		for _, line := range move.Lines {
			if line.StatementLineID != nil && *line.StatementLineID == l.ID {
				foundLiquidity = true
				// money in => debit on bank; money out => credit on bank
				if l.Amount > 0 && line.Debit != l.Amount {
					t.Fatalf("expected liquidity debit %v, got %v", l.Amount, line.Debit)
				}
				if l.Amount < 0 && line.Credit != -l.Amount {
					t.Fatalf("expected liquidity credit %v, got %v", -l.Amount, line.Credit)
				}
			}
		}
		if !foundLiquidity {
			t.Fatalf("line %d has no liquidity line referencing it", l.ID)
		}
	}

	// Confirm on fully-booked statement succeeds.
	confirmed, err := uc.ConfirmStatement(context.Background(), st.ID)
	if err != nil {
		t.Fatalf("ConfirmStatement failed: %v", err)
	}
	if confirmed.State != bankstatement.StatementStateConfirm {
		t.Fatalf("expected confirmed state, got %s", confirmed.State)
	}

	_ = repo
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconciliation
// ─────────────────────────────────────────────────────────────────────────────

func TestReconcileLineFullAndUndo(t *testing.T) {
	uc, fa, repo := newUseCase()
	date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	// Seed a posted customer invoice of 100 on AR for partner 5.
	fa.seedInvoice(200, 20, 3, 100, int64Ptr(5))

	st, err := uc.CreateStatement(context.Background(), bankstatementusecase.CreateStatementInput{
		JournalID: 1,
		Date:      date,
		Currency:  "USD",
		Lines: []bankstatementusecase.AddStatementLineInput{
			{Name: "Customer payment", Amount: 100, PartnerID: int64Ptr(5), Date: date},
		},
	})
	if err != nil {
		t.Fatalf("CreateStatement failed: %v", err)
	}
	line := st.Lines[0]

	candidates, err := uc.FindCandidates(context.Background(), line.ID, 10)
	if err != nil {
		t.Fatalf("FindCandidates failed: %v", err)
	}
	if len(candidates) != 1 || !candidates[0].ExactMatch {
		t.Fatalf("expected exactly one exact candidate, got %+v", candidates)
	}

	result, err := uc.ReconcileLine(context.Background(), line.ID, candidates[0].MoveLine.ID)
	if err != nil {
		t.Fatalf("ReconcileLine failed: %v", err)
	}
	if !result.StatementReconciled || !result.CandidateReconciled {
		t.Fatalf("expected full reconcile, got %+v", result)
	}
	if result.PaymentState != accounting.PaymentStatePaid {
		t.Fatalf("expected invoice to be paid, got %s", result.PaymentState)
	}
	if result.MatchingNumber == nil || *result.MatchingNumber == "" {
		t.Fatalf("expected a matching number to be assigned")
	}

	// FullReconcile must be persisted.
	_, err = repo.GetFullReconcileByMatchingNumber(context.Background(), *result.MatchingNumber)
	if err != nil {
		t.Fatalf("full reconcile not persisted: %v", err)
	}

	// Statement line must now be marked reconciled.
	final, err := uc.GetStatement(context.Background(), st.ID)
	if err != nil {
		t.Fatalf("GetStatement failed: %v", err)
	}
	if !final.Lines[0].Reconciled || final.Lines[0].AmountResidual != 0 {
		t.Fatalf("statement line should be reconciled with residual 0, got %+v", final.Lines[0])
	}

	// Undo must restore everything.
	if err := uc.UndoReconcile(context.Background(), line.ID); err != nil {
		t.Fatalf("UndoReconcile failed: %v", err)
	}
	after, err := uc.GetStatement(context.Background(), st.ID)
	if err != nil {
		t.Fatalf("GetStatement failed: %v", err)
	}
	if after.Lines[0].Reconciled || math.Abs(after.Lines[0].AmountResidual-100) > 0.004 {
		t.Fatalf("statement line should be unreconciled with residual 100, got %+v", after.Lines[0])
	}
	cand, err := fa.GetMoveLine(context.Background(), candidates[0].MoveLine.ID)
	if err != nil {
		t.Fatalf("GetMoveLine failed: %v", err)
	}
	if cand.Reconciled || math.Abs(cand.AmountResidual-100) > 0.004 {
		t.Fatalf("candidate residual should be restored to 100, got %+v", cand)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CSV import
// ─────────────────────────────────────────────────────────────────────────────

func TestImportCSVBooksRows(t *testing.T) {
	uc, _, _ := newUseCase()
	st, err := uc.CreateStatement(context.Background(), bankstatementusecase.CreateStatementInput{
		JournalID: 1,
		Date:      time.Now(),
		Currency:  "USD",
	})
	if err != nil {
		t.Fatalf("CreateStatement failed: %v", err)
	}

	csvData := []byte("date,name,ref,amount,currency\n2026-03-01,Cafe,ref-1,12.50,USD\n2026-03-02,COOP,ref-2,-8.00,USD\n")
	lines, err := uc.ImportCSV(context.Background(), st.ID, csvData)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 imported lines, got %d", len(lines))
	}
	for _, l := range lines {
		if l.MoveID == nil {
			t.Fatalf("imported line %d was not booked", l.ID)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Partial reconciliation then undo
// ─────────────────────────────────────────────────────────────────────────────

func TestReconcileLinePartialAndUndo(t *testing.T) {
	uc, fa, _ := newUseCase()
	date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	// Invoice of 150 leaves 50 outstanding after a 100 payment.
	fa.seedInvoice(200, 20, 3, 150, int64Ptr(5))

	st, err := uc.CreateStatement(context.Background(), bankstatementusecase.CreateStatementInput{
		JournalID: 1,
		Date:      date,
		Currency:  "USD",
		Lines: []bankstatementusecase.AddStatementLineInput{
			{Name: "Customer payment", Amount: 100, PartnerID: int64Ptr(5), Date: date},
		},
	})
	if err != nil {
		t.Fatalf("CreateStatement failed: %v", err)
	}
	line := st.Lines[0]

	result, err := uc.ReconcileLine(context.Background(), line.ID, 20)
	if err != nil {
		t.Fatalf("ReconcileLine failed: %v", err)
	}
	if !result.StatementReconciled || result.CandidateReconciled {
		t.Fatalf("expected statement settled but candidate partially reconciled, got %+v", result)
	}
	if result.PaymentState != accounting.PaymentStatePartial {
		t.Fatalf("expected payment_state partial, got %s", result.PaymentState)
	}
	if result.MatchingNumber != nil {
		t.Fatalf("partial reconcile must not assign a matching number, got %q", *result.MatchingNumber)
	}

	cand, err := fa.GetMoveLine(context.Background(), 20)
	if err != nil {
		t.Fatalf("GetMoveLine failed: %v", err)
	}
	if math.Abs(cand.AmountResidual-50) > 0.004 {
		t.Fatalf("expected candidate residual 50, got %v", cand.AmountResidual)
	}

	// Undo restores the candidate residual back to 150.
	if err := uc.UndoReconcile(context.Background(), line.ID); err != nil {
		t.Fatalf("UndoReconcile failed: %v", err)
	}
	cand, _ = fa.GetMoveLine(context.Background(), 20)
	if math.Abs(cand.AmountResidual-150) > 0.004 {
		t.Fatalf("expected candidate residual restored to 150, got %v", cand.AmountResidual)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Auto reconcile
// ─────────────────────────────────────────────────────────────────────────────

func TestAutoReconcileMatchesModel(t *testing.T) {
	uc, fa, _ := newUseCase()
	date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	// Auto-reconcile model matching everything.
	if _, err := uc.CreateReconcileModel(context.Background(), &bankstatement.ReconcileModel{
		Name:            "Any customer payment",
		IsAutoReconcile: true,
		Active:          true,
	}); err != nil {
		t.Fatalf("CreateReconcileModel failed: %v", err)
	}

	fa.seedInvoice(200, 20, 3, 100, int64Ptr(5))

	st, err := uc.CreateStatement(context.Background(), bankstatementusecase.CreateStatementInput{
		JournalID: 1,
		Date:      date,
		Currency:  "USD",
		Lines: []bankstatementusecase.AddStatementLineInput{
			{Name: "Customer payment", Amount: 100, PartnerID: int64Ptr(5), Date: date},
		},
	})
	if err != nil {
		t.Fatalf("CreateStatement failed: %v", err)
	}

	count, err := uc.AutoReconcile(context.Background(), st.ID)
	if err != nil {
		t.Fatalf("AutoReconcile failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 reconciled line by auto reconcile, got %d", count)
	}

	final, err := uc.GetStatement(context.Background(), st.ID)
	if err != nil {
		t.Fatalf("GetStatement failed: %v", err)
	}
	if !final.Lines[0].Reconciled {
		t.Fatalf("expected statement line reconciled after auto reconcile, got %+v", final.Lines[0])
	}
}
