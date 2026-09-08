package paymentusecase

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/payment"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test Mocks
// ─────────────────────────────────────────────────────────────────────────────

type mockPaymentRepo struct {
	payments        map[int64]*payment.Payment
	reconciliations map[int64][]payment.PaymentReconciliation
	lastID          int64
	lastSeq         int
}

func (m *mockPaymentRepo) CreateTransaction(ctx context.Context, t *payment.PaymentTransaction) error {
	return nil
}
func (m *mockPaymentRepo) GetTransactionByID(ctx context.Context, id int64) (*payment.PaymentTransaction, error) {
	return nil, platformerrors.NotFound("transaction not found")
}
func (m *mockPaymentRepo) GetTransactionByReference(ctx context.Context, ref string) (*payment.PaymentTransaction, error) {
	return nil, platformerrors.NotFound("transaction not found")
}
func (m *mockPaymentRepo) UpdateTransaction(ctx context.Context, t *payment.PaymentTransaction) error {
	return nil
}
func (m *mockPaymentRepo) GetProviderByCode(ctx context.Context, code string, companyID int64) (*payment.PaymentProvider, error) {
	return nil, platformerrors.NotFound("provider not found")
}

func newMockPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{
		payments:        make(map[int64]*payment.Payment),
		reconciliations: make(map[int64][]payment.PaymentReconciliation),
	}
}

func (m *mockPaymentRepo) CreatePayment(ctx context.Context, p *payment.Payment) error {
	m.lastID++
	p.ID = m.lastID
	cp := *p
	m.payments[p.ID] = &cp
	return nil
}

func (m *mockPaymentRepo) GetPaymentByID(ctx context.Context, id int64) (*payment.Payment, error) {
	p, ok := m.payments[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("payment %d not found", id))
	}
	cp := *p
	return &cp, nil
}

func (m *mockPaymentRepo) UpdatePayment(ctx context.Context, p *payment.Payment) error {
	if _, ok := m.payments[p.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("payment %d not found", p.ID))
	}
	cp := *p
	m.payments[p.ID] = &cp
	return nil
}

func (m *mockPaymentRepo) DeletePayment(ctx context.Context, id int64) error {
	delete(m.payments, id)
	return nil
}

func (m *mockPaymentRepo) ListPayments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[payment.Payment], error) {
	var items []payment.Payment
	for _, p := range m.payments {
		items = append(items, *p)
	}
	return pagination.PageResult[payment.Payment]{
		Items:      items,
		TotalItems: int64(len(items)),
		Page:       page.Page,
		Limit:      page.LimitClamped(),
	}, nil
}

func (m *mockPaymentRepo) NextSequence(ctx context.Context, year int) (string, error) {
	m.lastSeq++
	return fmt.Sprintf("PAY/%d/%05d", year, m.lastSeq), nil
}

func (m *mockPaymentRepo) CreateReconciliation(ctx context.Context, r *payment.PaymentReconciliation) error {
	m.reconciliations[r.PaymentID] = append(m.reconciliations[r.PaymentID], *r)
	return nil
}

func (m *mockPaymentRepo) GetReconciliationsByPaymentID(ctx context.Context, paymentID int64) ([]payment.PaymentReconciliation, error) {
	return m.reconciliations[paymentID], nil
}

func (m *mockPaymentRepo) GetReconciliationsByInvoiceID(ctx context.Context, invoiceID int64) ([]payment.PaymentReconciliation, error) {
	var res []payment.PaymentReconciliation
	for _, list := range m.reconciliations {
		for _, r := range list {
			if r.InvoiceID == invoiceID {
				res = append(res, r)
			}
		}
	}
	return res, nil
}

func (m *mockPaymentRepo) DeleteReconciliationsByPaymentID(ctx context.Context, paymentID int64) error {
	delete(m.reconciliations, paymentID)
	return nil
}

type mockAccountingService struct {
	moves    map[int64]*accounting.AccountMove
	journals map[int64]*accounting.Journal
	accounts map[string]*accounting.Account
	lastID   int64
}

func newMockAccountingService() *mockAccountingService {
	m := &mockAccountingService{
		moves:    make(map[int64]*accounting.AccountMove),
		journals: make(map[int64]*accounting.Journal),
		accounts: make(map[string]*accounting.Account),
	}
	m.journals[3] = &accounting.Journal{
		ID:   3,
		Name: "Bank",
		Code: "BNK1",
		Type: accounting.JournalTypeBank,
	}
	m.journals[4] = &accounting.Journal{
		ID:   4,
		Name: "Cash",
		Code: "CSH1",
		Type: accounting.JournalTypeCash,
	}
	m.accounts["101000"] = &accounting.Account{ID: 1, Code: "101000", Name: "Cash"}
	m.accounts["102000"] = &accounting.Account{ID: 2, Code: "102000", Name: "Bank"}
	m.accounts["120000"] = &accounting.Account{ID: 3, Code: "120000", Name: "AR"}
	m.accounts["210000"] = &accounting.Account{ID: 6, Code: "210000", Name: "AP"}
	return m
}

func (s *mockAccountingService) CreateJournalEntry(ctx context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error) {
	s.lastID++
	move := &accounting.AccountMove{
		ID:        s.lastID,
		Name:      "/",
		MoveType:  accounting.MoveTypeEntry,
		JournalID: in.JournalID,
		Date:      in.Date,
		State:     accounting.MoveStateDraft,
		Ref:       in.Ref,
	}
	s.moves[move.ID] = move
	return move, nil
}

func (s *mockAccountingService) PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	m, ok := s.moves[id]
	if !ok {
		return nil, platformerrors.NotFound("move not found")
	}
	m.State = accounting.MoveStatePosted
	m.Name = fmt.Sprintf("ENTRY/2026/%05d", id)
	return m, nil
}

func (s *mockAccountingService) CancelMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	m, ok := s.moves[id]
	if !ok {
		return nil, platformerrors.NotFound("move not found")
	}
	m.State = accounting.MoveStateCancel
	return m, nil
}

func (s *mockAccountingService) GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	m, ok := s.moves[id]
	if !ok {
		return nil, platformerrors.NotFound("move not found")
	}
	return m, nil
}

func (s *mockAccountingService) GetJournal(ctx context.Context, id int64) (*accounting.Journal, error) {
	j, ok := s.journals[id]
	if !ok {
		return nil, platformerrors.NotFound("journal not found")
	}
	return j, nil
}

func (s *mockAccountingService) GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error) {
	a, ok := s.accounts[code]
	if !ok {
		return nil, platformerrors.NotFound("account not found")
	}
	return a, nil
}

func (s *mockAccountingService) ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.AccountMove], error) {
	var items []accounting.AccountMove
	for _, m := range s.moves {
		items = append(items, *m)
	}
	return pagination.PageResult[accounting.AccountMove]{
		Items:      items,
		TotalItems: int64(len(items)),
		Page:       page.Page,
		Limit:      page.LimitClamped(),
	}, nil
}

func (s *mockAccountingService) UpdatePaymentStatus(ctx context.Context, id int64, state accounting.PaymentState, residual float64) error {
	m, ok := s.moves[id]
	if !ok {
		return platformerrors.NotFound("move not found")
	}
	m.PaymentState = state
	m.AmountResidual = residual
	return nil
}

type mockPartnerRepo struct {
	partners map[int64]*partner.Partner
}

func newMockPartnerRepo() *mockPartnerRepo {
	return &mockPartnerRepo{
		partners: map[int64]*partner.Partner{
			10: {ID: 10, Name: "Acme Corp", IsCustomer: true},
			20: {ID: 20, Name: "Global Supplier", IsSupplier: true},
		},
	}
}

func (m *mockPartnerRepo) GetByID(ctx context.Context, id int64) (*partner.Partner, error) {
	p, ok := m.partners[id]
	if !ok {
		return nil, platformerrors.NotFound("partner not found")
	}
	return p, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestPaymentUseCase_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pRepo := newMockPaymentRepo()
	accSvc := newMockAccountingService()
	partRepo := newMockPartnerRepo()

	uc := New(pRepo, accSvc, partRepo, logger)

	// Setup an open customer invoice in accounting
	invPartnerID := int64(10)
	dueDate := time.Now().Add(10 * 24 * time.Hour)
	invoice := &accounting.AccountMove{
		ID:             101,
		Name:           "INV/2026/00001",
		MoveType:       accounting.MoveTypeOutInvoice,
		PartnerID:      &invPartnerID,
		Date:           time.Now(),
		InvoiceDueDate: &dueDate,
		State:          accounting.MoveStatePosted,
		PaymentState:   accounting.PaymentStateNotPaid,
		AmountTotal:    1000.0,
		AmountResidual: 1000.0,
	}
	accSvc.moves[invoice.ID] = invoice

	// 1. Create Payment Draft
	created, err := uc.CreatePayment(ctx, CreatePaymentInput{
		PartnerID:     10,
		Amount:        600.0,
		PaymentType:   payment.PaymentTypeInbound,
		PaymentMethod: payment.PaymentMethodBankTransfer,
		JournalID:     3,
		Ref:           "INV/2026/00001 Partial",
		InvoiceIDs:    []int64{invoice.ID},
	})
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}
	if created.State != payment.PaymentStateDraft {
		t.Errorf("expected draft state, got %s", created.State)
	}
	if created.ResidualAmount != 600.0 {
		t.Errorf("expected residual 600, got %.2f", created.ResidualAmount)
	}

	// 2. Post Payment (with auto-reconcile on invoice)
	posted, err := uc.PostPayment(ctx, created.ID, true)
	if err != nil {
		t.Fatalf("failed to post payment: %v", err)
	}
	if posted.State != payment.PaymentStateReconciled {
		t.Errorf("expected reconciled state after full allocation of 600, got %s", posted.State)
	}
	if posted.MoveID == nil {
		t.Errorf("expected MoveID to be set, got nil")
	}
	if posted.Name != "PAY/2026/00001" {
		t.Errorf("expected sequence PAY/2026/00001, got %s", posted.Name)
	}

	// Check invoice state in accounting
	if invoice.AmountResidual != 400.0 {
		t.Errorf("expected invoice residual 400, got %.2f", invoice.AmountResidual)
	}
	if invoice.PaymentState != accounting.PaymentStatePartial {
		t.Errorf("expected invoice payment state partial, got %s", invoice.PaymentState)
	}

	// 3. Create second payment to settle the rest
	created2, err := uc.CreatePayment(ctx, CreatePaymentInput{
		PartnerID:     10,
		Amount:        400.0,
		PaymentType:   payment.PaymentTypeInbound,
		PaymentMethod: payment.PaymentMethodCash,
		JournalID:     4,
		Ref:           "INV/2026/00001 Final",
		InvoiceIDs:    []int64{invoice.ID},
	})
	if err != nil {
		t.Fatalf("failed to create payment 2: %v", err)
	}

	posted2, err := uc.PostPayment(ctx, created2.ID, true)
	if err != nil {
		t.Fatalf("failed to post payment 2: %v", err)
	}
	if posted2.State != payment.PaymentStateReconciled {
		t.Errorf("expected state reconciled, got %s", posted2.State)
	}

	// Now invoice should be fully paid!
	if invoice.AmountResidual != 0.0 {
		t.Errorf("expected invoice residual 0, got %.2f", invoice.AmountResidual)
	}
	if invoice.PaymentState != accounting.PaymentStatePaid {
		t.Errorf("expected invoice payment state paid, got %s", invoice.PaymentState)
	}

	// 4. Cancel payment 2: residual should restore to 400 and invoice state back to partial
	cancelled2, err := uc.CancelPayment(ctx, posted2.ID)
	if err != nil {
		t.Fatalf("failed to cancel payment: %v", err)
	}
	if cancelled2.State != payment.PaymentStateCancelled {
		t.Errorf("expected cancelled state, got %s", cancelled2.State)
	}
	if invoice.AmountResidual != 400.0 {
		t.Errorf("expected restored invoice residual 400, got %.2f", invoice.AmountResidual)
	}
	if invoice.PaymentState != accounting.PaymentStatePartial {
		t.Errorf("expected restored invoice state partial, got %s", invoice.PaymentState)
	}
}

func TestPaymentUseCase_AgingReports(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pRepo := newMockPaymentRepo()
	accSvc := newMockAccountingService()
	partRepo := newMockPartnerRepo()

	uc := New(pRepo, accSvc, partRepo, logger)

	asOf := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)

	// Add receivable invoices for Customer 10
	partnerID10 := int64(10)
	currentDue := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	overdue15 := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)

	accSvc.moves[201] = &accounting.AccountMove{
		ID:             201,
		Name:           "INV/2026/0001",
		MoveType:       accounting.MoveTypeOutInvoice,
		PartnerID:      &partnerID10,
		Date:           time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		InvoiceDueDate: &currentDue,
		State:          accounting.MoveStatePosted,
		PaymentState:   accounting.PaymentStateNotPaid,
		AmountTotal:    500.0,
		AmountResidual: 500.0,
	}

	accSvc.moves[202] = &accounting.AccountMove{
		ID:             202,
		Name:           "INV/2026/0002",
		MoveType:       accounting.MoveTypeOutInvoice,
		PartnerID:      &partnerID10,
		Date:           time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		InvoiceDueDate: &overdue15,
		State:          accounting.MoveStatePosted,
		PaymentState:   accounting.PaymentStateNotPaid,
		AmountTotal:    300.0,
		AmountResidual: 300.0,
	}

	// Add payable bill for Supplier 20
	partnerID20 := int64(20)
	overdue45 := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	accSvc.moves[301] = &accounting.AccountMove{
		ID:             301,
		Name:           "BILL/2026/0001",
		MoveType:       accounting.MoveTypeInInvoice,
		PartnerID:      &partnerID20,
		Date:           time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		InvoiceDueDate: &overdue45,
		State:          accounting.MoveStatePosted,
		PaymentState:   accounting.PaymentStateNotPaid,
		AmountTotal:    750.0,
		AmountResidual: 750.0,
	}

	// 1. Test Aged Receivable
	recAging, err := uc.GetReceivableAging(ctx, asOf, nil)
	if err != nil {
		t.Fatalf("GetReceivableAging failed: %v", err)
	}
	if recAging.Total.Current != 500.0 {
		t.Errorf("expected current 500, got %.2f", recAging.Total.Current)
	}
	if recAging.Total.Days1_30 != 300.0 {
		t.Errorf("expected days 1-30 300, got %.2f", recAging.Total.Days1_30)
	}
	if recAging.Total.Total != 800.0 {
		t.Errorf("expected total 800, got %.2f", recAging.Total.Total)
	}
	if len(recAging.Partners) != 1 || recAging.Partners[0].PartnerName != "Acme Corp" {
		t.Errorf("unexpected partner in receivable aging: %+v", recAging.Partners)
	}

	// 2. Test Aged Payable
	payAging, err := uc.GetPayableAging(ctx, asOf, nil)
	if err != nil {
		t.Fatalf("GetPayableAging failed: %v", err)
	}
	if payAging.Total.Days31_60 != 750.0 {
		t.Errorf("expected days 31-60 750, got %.2f", payAging.Total.Days31_60)
	}
	if payAging.Total.Total != 750.0 {
		t.Errorf("expected total 750, got %.2f", payAging.Total.Total)
	}
	if len(payAging.Partners) != 1 || payAging.Partners[0].PartnerName != "Global Supplier" {
		t.Errorf("unexpected partner in payable aging: %+v", payAging.Partners)
	}
}
