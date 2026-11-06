package accounting

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCreateAndUpdateJournalUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	createUC := NewCreateJournalUseCase(repo, nil)
	updateUC := NewUpdateJournalUseCase(repo, nil)
	ctx := context.Background()

	// 1. Create journal
	req := CreateJournalRequest{
		Code:           "SAL1",
		Name:           "Sales Journal",
		Type:           "sale",
		SequencePrefix: "SAL/2026/",
	}
	res, err := createUC.Execute(ctx, req)
	if err != nil {
		t.Fatalf("expected create journal success, got: %v", err)
	}
	if res.Code != "SAL1" || res.SequencePrefix != "SAL/2026/" {
		t.Errorf("unexpected journal response: %+v", res)
	}

	// 2. Reject duplicate code
	_, err = createUC.Execute(ctx, req)
	if err != ErrDuplicateJournalCode {
		t.Errorf("expected ErrDuplicateJournalCode, got: %v", err)
	}

	// 3. Update journal
	upReq := UpdateJournalRequest{
		Name:           "Updated Sales Journal",
		SequencePrefix: "SAL/2026/V2/",
	}
	upRes, err := updateUC.Execute(ctx, res.ID, upReq)
	if err != nil {
		t.Fatalf("expected update journal success, got: %v", err)
	}
	if upRes.Name != "Updated Sales Journal" || upRes.SequencePrefix != "SAL/2026/V2/" {
		t.Errorf("unexpected updated journal: %+v", upRes)
	}
}

func TestAccountingConfigurationUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	getUC := NewGetAccountingConfigurationUseCase(repo)
	updateUC := NewUpdateAccountingConfigurationUseCase(repo, nil)
	ctx := context.Background()

	// 1. Get configuration when missing
	_, err := getUC.Execute(ctx)
	if err == nil {
		t.Fatal("expected error on missing configuration")
	}

	// 2. Setup mock accounts
	retainedAcc := &Account{ID: uuid.New(), Code: "300000", Name: "Retained Earnings", Type: AccountTypeEquity, IsActive: true}
	fxGainAcc := &Account{ID: uuid.New(), Code: "450000", Name: "FX Gain", Type: AccountTypeIncome, IsActive: true}
	fxLossAcc := &Account{ID: uuid.New(), Code: "550000", Name: "FX Loss", Type: AccountTypeExpense, IsActive: true}
	journal := &Journal{ID: uuid.New(), Code: "MISC", Name: "General", Type: "general", IsActive: true}

	repo.accountsByID[retainedAcc.ID] = retainedAcc
	repo.accountsByID[fxGainAcc.ID] = fxGainAcc
	repo.accountsByID[fxLossAcc.ID] = fxLossAcc
	repo.journal = journal

	// 3. Update configuration
	upReq := UpdateAccountingConfigurationRequest{
		RetainedEarningsAccountID: retainedAcc.ID,
		FXGainAccountID:           fxGainAcc.ID,
		FXLossAccountID:           fxLossAcc.ID,
		FXReconciliationJournalID: journal.ID,
		FunctionalCurrency:        "SAR",
	}
	upRes, err := updateUC.Execute(ctx, upReq)
	if err != nil {
		t.Fatalf("expected update config success, got: %v", err)
	}
	if upRes.FunctionalCurrency != "SAR" {
		t.Errorf("expected functional currency SAR, got %s", upRes.FunctionalCurrency)
	}

	// 4. Get configuration returns updated
	getRes, err := getUC.Execute(ctx)
	if err != nil {
		t.Fatalf("expected get config success, got: %v", err)
	}
	if getRes.RetainedEarningsAccountID != retainedAcc.ID {
		t.Errorf("unexpected retained earnings account: %s", getRes.RetainedEarningsAccountID)
	}
}

func TestAuditIntegrityAndLogUseCases(t *testing.T) {
	repo := newMockAccountingRepository()
	verifyUC := NewVerifyAuditIntegrityUseCase(repo)
	listUC := NewListAuditLogsUseCase(repo)
	ctx := context.Background()

	// 1. Verify integrity
	res, err := verifyUC.Execute(ctx)
	if err != nil {
		t.Fatalf("expected verify integrity success, got: %v", err)
	}
	if !res.IsValid {
		t.Error("expected audit chain to be valid")
	}

	// 2. List audit logs
	logsRes, err := listUC.Execute(ctx, 10, 0)
	if err != nil {
		t.Fatalf("expected list audit logs success, got: %v", err)
	}
	if logsRes.Limit != 10 {
		t.Errorf("expected limit 10, got %d", logsRes.Limit)
	}
}

func TestSubledgerMappingsAndEventsUseCases(t *testing.T) {
	repo := newMockAccountingRepository()
	listMappingsUC := NewListSubledgerMappingsUseCase(repo)
	saveMappingUC := NewSaveSubledgerMappingUseCase(repo)
	listEventsUC := NewListSubledgerEventsUseCase(repo)
	ctx := context.Background()

	journal := &Journal{ID: uuid.New(), Code: "BILL", Name: "Purchase", Type: "purchase", IsActive: true}
	debitAcc := &Account{ID: uuid.New(), Code: "500000", Name: "Purchases", Type: AccountTypeExpense, IsActive: true}
	creditAcc := &Account{ID: uuid.New(), Code: "201000", Name: "AP Vendors", Type: AccountTypeLiability, IsActive: true}

	repo.journal = journal
	repo.accountsByID[debitAcc.ID] = debitAcc
	repo.accountsByID[creditAcc.ID] = creditAcc

	// 1. Save mapping
	req := SaveSubledgerEventMappingRequest{
		EventType:       "vendor_bill_received",
		JournalID:       journal.ID,
		DebitAccountID:  debitAcc.ID,
		CreditAccountID: creditAcc.ID,
		Description:     "Vendor bill received from procurement",
		IsActive:        true,
	}
	saveRes, err := saveMappingUC.Execute(ctx, req)
	if err != nil {
		t.Fatalf("expected save mapping success, got: %v", err)
	}
	if saveRes.EventType != "vendor_bill_received" {
		t.Errorf("unexpected event type: %s", saveRes.EventType)
	}

	// 2. List mappings
	mappings, err := listMappingsUC.Execute(ctx)
	if err != nil {
		t.Fatalf("expected list mappings success, got: %v", err)
	}
	if len(mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(mappings))
	}

	// 3. List events
	events, err := listEventsUC.Execute(ctx, 50, 0)
	if err != nil {
		t.Fatalf("expected list events success, got: %v", err)
	}
	if events.Limit != 50 {
		t.Errorf("expected limit 50, got %d", events.Limit)
	}
}

func TestBulkPostMovesUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	logger := slog.Default()
	postMoveUC := newPostMoveUseCase(repo, nil, MakerCheckerSettings{}, logger)
	bulkUC := NewBulkPostMovesUseCase(postMoveUC, logger)
	ctx := context.Background()

	now := time.Now().UTC()
	period := &FiscalPeriod{ID: uuid.New(), FiscalYearID: uuid.New(), State: PeriodStateOpen, StartDate: now.Add(-24 * time.Hour), EndDate: now.Add(24 * time.Hour)}
	repo.period = period

	move1 := &AccountMove{
		ID:        uuid.New(),
		JournalID: uuid.New(),
		Date:      now,
		State:     MoveStateDraft,
		Lines: []AccountMoveLine{
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.NewFromInt(100), Credit: decimal.Zero},
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.Zero, Credit: decimal.NewFromInt(100)},
		},
	}
	repo.move = move1

	actorID := uuid.New()
	res, err := bulkUC.Execute(ctx, actorID, []uuid.UUID{move1.ID})
	if err != nil {
		t.Fatalf("expected bulk post success, got: %v", err)
	}
	if res.PostedCount != 1 {
		t.Errorf("expected 1 posted move, got %d", res.PostedCount)
	}
}
