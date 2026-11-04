package accounting

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCreateAccountUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	parent := &Account{ID: uuid.New(), IsActive: true}
	repo.accountsByID[parent.ID] = parent
	parentID := parent.ID

	response, err := NewCreateAccountUseCase(repo, nil).Execute(context.Background(), CreateAccountRequest{
		Code: " 1000 ", Name: " Cash ", Type: " ASSET ", ParentID: &parentID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.Code != "1000" || response.Name != "Cash" || response.NormalBalance != "debit" || response.CurrencyID != "SAR" {
		t.Fatalf("created account = %+v", response)
	}
	if response.ParentID == nil || *response.ParentID != parent.ID {
		t.Fatalf("parent id = %v, want %s", response.ParentID, parent.ID)
	}
	if _, err := NewCreateAccountUseCase(repo, nil).Execute(context.Background(), CreateAccountRequest{
		Code: "1000", Name: "Duplicate", Type: "asset",
	}); !errors.Is(err, ErrDuplicateAccountCode) {
		t.Fatalf("duplicate account error = %v, want ErrDuplicateAccountCode", err)
	}
}

func TestNewUseCasesWiresAllUseCasesAndFXProvider(t *testing.T) {
	provider := &mockFXRevaluationProvider{}
	useCases := NewUseCasesWithFXProvider(newMockAccountingRepository(), nil, provider, nil)
	if useCases.CreateAccount == nil || useCases.ListAccounts == nil || useCases.ListAccountsTree == nil ||
		useCases.GetMove == nil || useCases.CreateFiscalYear == nil ||
		useCases.ChangePeriodState == nil || useCases.CreateDraftMove == nil || useCases.PostMove == nil ||
		useCases.ReverseMove == nil || useCases.RevalueFX == nil || useCases.GenerateTrialBalance == nil ||
		useCases.IngestSubledgerEvent == nil || useCases.RevalueFX.provider != provider {
		t.Fatalf("use-case container is not fully wired: %+v", useCases)
	}
}

func TestNewUseCasesWithFXRatesWiresRepositoryProvider(t *testing.T) {
	repo := newMockAccountingRepository()
	rates := &mockFXRateProvider{rates: map[string]decimal.Decimal{"USD/SAR": decimal.RequireFromString("3.75")}}
	useCases, err := NewUseCasesWithFXRates(repo, nil, rates, FXRevaluationSettings{
		JournalID: uuid.New(), GainAccountID: uuid.New(), LossAccountID: uuid.New(), BaseCurrency: "sar",
	}, nil)
	if err != nil {
		t.Fatalf("NewUseCasesWithFXRates() error = %v", err)
	}
	if useCases.RevalueFX.provider == nil {
		t.Fatal("NewUseCasesWithFXRates() did not wire the FX provider")
	}
}

func TestNewUseCasesWithMakerCheckerValidatesConfigurableSettings(t *testing.T) {
	useCases, err := NewUseCasesWithMakerChecker(newMockAccountingRepository(), nil, MakerCheckerSettings{
		Threshold: decimal.NewFromInt(25000), FunctionalCurrency: "sar",
	}, nil)
	if err != nil {
		t.Fatalf("NewUseCasesWithMakerChecker() error = %v", err)
	}
	if got := useCases.PostMove.governance; !got.Threshold.Equal(decimal.NewFromInt(25000)) || got.FunctionalCurrency != "SAR" {
		t.Fatalf("maker-checker settings = %+v", got)
	}

	if _, err := NewUseCasesWithMakerChecker(newMockAccountingRepository(), nil, MakerCheckerSettings{
		Threshold: decimal.NewFromInt(-1), FunctionalCurrency: "SAR",
	}, nil); err == nil {
		t.Fatal("NewUseCasesWithMakerChecker() accepted a negative threshold")
	}
}

func TestListAccountsTreeUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	parent := &Account{ID: uuid.New(), Code: "1000", Name: "Cash"}
	child := &Account{ID: uuid.New(), Code: "1010", Name: "Bank", ParentID: &parent.ID}
	repo.accounts = []*Account{parent, child}

	tree, err := NewListAccountsTreeUseCase(repo, nil).Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(tree) != 1 || tree[0].ID != parent.ID || len(tree[0].Children) != 1 || tree[0].Children[0].ID != child.ID {
		t.Fatalf("account tree = %+v", tree)
	}

	parent.ParentID = &child.ID
	if _, err := NewListAccountsTreeUseCase(repo, nil).Execute(context.Background()); err == nil {
		t.Fatal("Execute() expected a cycle error")
	}
}

func TestCreateFiscalYearUseCaseCreatesTwelvePeriods(t *testing.T) {
	repo := newMockAccountingRepository()
	response, err := NewCreateFiscalYearUseCase(repo, nil).Execute(context.Background(), CreateFiscalYearRequest{
		Year: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(response.Periods) != 12 || response.Periods[0].StartDate != "2026-01-01" ||
		response.Periods[0].EndDate != "2026-01-31" || response.Periods[11].EndDate != "2026-12-31" {
		t.Fatalf("fiscal periods = %+v", response.Periods)
	}
	if _, err := NewCreateFiscalYearUseCase(repo, nil).Execute(context.Background(), CreateFiscalYearRequest{
		Year: 2026, StartDate: "2026-01-15", EndDate: "2027-01-14",
	}); err == nil {
		t.Fatal("Execute() expected incomplete calendar months to be rejected")
	}
}

func TestChangePeriodStateUseCaseEnforcesCloseTransitions(t *testing.T) {
	repo := newMockAccountingRepository()
	period := &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
	repo.period = period
	repo.periodsByID[period.ID] = period

	response, err := NewChangePeriodStateUseCase(repo, nil).Execute(context.Background(), period.ID, uuid.New(), ChangePeriodStateRequest{State: "soft_closed"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.State != string(PeriodStateSoftClosed) || response.ClosedAt == nil {
		t.Fatalf("closed period response = %+v", response)
	}
	if _, err := NewChangePeriodStateUseCase(repo, nil).Execute(context.Background(), period.ID, uuid.New(), ChangePeriodStateRequest{State: "hard_closed"}); err != nil {
		t.Fatalf("soft close to hard close error = %v", err)
	}
	if _, err := NewChangePeriodStateUseCase(repo, nil).Execute(context.Background(), period.ID, uuid.New(), ChangePeriodStateRequest{State: "open"}); err == nil {
		t.Fatal("Execute() expected a hard-closed period to remain closed")
	}
}

func TestCreateDraftMoveUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	creatorID := uuid.New()
	date := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	debitAccount := &Account{ID: uuid.New(), IsActive: true}
	creditAccount := &Account{ID: uuid.New(), IsActive: true}
	repo.accountsByID[debitAccount.ID] = debitAccount
	repo.accountsByID[creditAccount.ID] = creditAccount
	repo.journal = &Journal{ID: uuid.New(), IsActive: true}
	repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}

	response, err := NewCreateDraftMoveUseCase(repo, nil).ExecuteAs(context.Background(), creatorID, CreateMoveRequest{
		JournalID: repo.journal.ID, Date: date.Format(dateLayout),
		Lines: []MoveLineRequest{
			{AccountID: debitAccount.ID, Debit: decimal.NewFromInt(25)},
			{AccountID: creditAccount.ID, Credit: decimal.NewFromInt(25)},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.State != string(MoveStateDraft) || len(response.Lines) != 2 || repo.draftCalls != 1 {
		t.Fatalf("draft response = %+v, save calls = %d", response, repo.draftCalls)
	}
	if repo.move.CreatedBy == nil || *repo.move.CreatedBy != creatorID {
		t.Fatalf("draft creator = %v, want %s", repo.move.CreatedBy, creatorID)
	}
}

func TestPostMoveUseCase(t *testing.T) {
	t.Run("posts balanced move and publishes event", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
		publisher := &mockEventPublisher{}
		response, err := NewPostMoveUseCase(repo, publisher, nil).Execute(context.Background(), repo.move.ID, uuid.New())
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if response.State != string(MoveStatePosted) || response.Name != "GEN/00001" || repo.postCalls != 1 {
			t.Fatalf("posted response = %+v, atomic calls = %d", response, repo.postCalls)
		}
		if len(publisher.events) != 1 || publisher.events[0] != "accounting.move.posted" {
			t.Fatalf("published events = %v", publisher.events)
		}
	})

	t.Run("rejects unbalanced move", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.move.Lines[0].Debit = decimal.NewFromInt(2)
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
		_, err := NewPostMoveUseCase(repo, nil, nil).Execute(context.Background(), repo.move.ID, uuid.New())
		if !errors.Is(err, ErrUnbalancedMove) || repo.postCalls != 0 {
			t.Fatalf("Execute() error = %v, atomic calls = %d", err, repo.postCalls)
		}
	})

	t.Run("rejects closed period", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateHardClosed}
		_, err := NewPostMoveUseCase(repo, nil, nil).Execute(context.Background(), repo.move.ID, uuid.New())
		if !errors.Is(err, ErrPeriodClosed) || repo.postCalls != 0 {
			t.Fatalf("Execute() error = %v, atomic calls = %d", err, repo.postCalls)
		}
	})
}

func TestPostMoveUseCaseMakerChecker(t *testing.T) {
	creatorID, checkerID := uuid.New(), uuid.New()
	settings := MakerCheckerSettings{
		Threshold: decimal.NewFromInt(20), FunctionalCurrency: "SAR",
	}

	t.Run("creator cannot post above threshold", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.move.CreatedBy = &creatorID
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
		uc := newPostMoveUseCase(repo, nil, settings, nil)

		_, err := uc.Execute(context.Background(), repo.move.ID, creatorID)
		if !errors.Is(err, ErrMakerCheckerViolation) || repo.postCalls != 0 {
			t.Fatalf("Execute() error = %v, atomic calls = %d", err, repo.postCalls)
		}
	})

	t.Run("different authenticated checker may post", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.move.CreatedBy = &creatorID
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
		uc := newPostMoveUseCase(repo, nil, settings, nil)

		if _, err := uc.Execute(context.Background(), repo.move.ID, checkerID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if repo.postCalls != 1 {
			t.Fatalf("atomic calls = %d, want 1", repo.postCalls)
		}
	})

	t.Run("missing actor cannot post above threshold", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.move.CreatedBy = &creatorID
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
		uc := newPostMoveUseCase(repo, nil, settings, nil)

		_, err := uc.Execute(context.Background(), repo.move.ID, uuid.Nil)
		if !errors.Is(err, ErrAccountingActorRequired) || repo.postCalls != 0 {
			t.Fatalf("Execute() error = %v, atomic calls = %d", err, repo.postCalls)
		}
	})

	t.Run("missing creator attribution cannot bypass the control", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
		uc := newPostMoveUseCase(repo, nil, settings, nil)

		_, err := uc.Execute(context.Background(), repo.move.ID, checkerID)
		if !errors.Is(err, ErrMakerCheckerCreatorRequired) || repo.postCalls != 0 {
			t.Fatalf("Execute() error = %v, atomic calls = %d", err, repo.postCalls)
		}
	})

	t.Run("equal to threshold does not require a second actor", func(t *testing.T) {
		repo := newMockAccountingRepository()
		repo.move = balancedMove(MoveStateDraft)
		repo.move.CreatedBy = &creatorID
		repo.move.Lines[0].Debit = decimal.NewFromInt(20)
		repo.move.Lines[1].Credit = decimal.NewFromInt(20)
		repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
		uc := newPostMoveUseCase(repo, nil, settings, nil)

		if _, err := uc.Execute(context.Background(), repo.move.ID, creatorID); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	})
}

func TestReverseMoveUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	repo.move = balancedMove(MoveStatePosted)
	repo.move.Name = "GEN/00001"
	repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
	response, err := NewReverseMoveUseCase(repo, nil, nil).Execute(
		context.Background(), repo.move.ID, uuid.New(), time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC), "correction",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.State != string(MoveStatePosted) || repo.reverseCalls != 1 ||
		repo.move.Lines[0].Debit.String() != "25" {
		t.Fatalf("reversal response = %+v", response)
	}
	if response.Lines[0].Debit.String() != "0" || response.Lines[0].Credit.String() != "25" {
		t.Fatalf("reversal line did not swap sides: %+v", response.Lines[0])
	}
}

func TestGenerateTrialBalanceUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	repo.trialBalance = []*TrialBalanceRow{
		{EndingDebit: decimal.NewFromInt(25), EndingCredit: decimal.Zero},
		{EndingDebit: decimal.Zero, EndingCredit: decimal.NewFromInt(25)},
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	response, err := NewGenerateTrialBalanceUseCase(repo, nil).Execute(context.Background(), from, to)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.IsBalanced || response.TotalDebit.String() != "25" || response.TotalCredit.String() != "25" {
		t.Fatalf("trial balance response = %+v", response)
	}
}

func TestRevalueFXUseCase(t *testing.T) {
	repo := newMockAccountingRepository()
	asset := &Account{ID: uuid.New(), Type: AccountTypeAsset, IsActive: true}
	gain := &Account{ID: uuid.New(), Type: AccountTypeIncome, IsActive: true}
	loss := &Account{ID: uuid.New(), Type: AccountTypeExpense, IsActive: true}
	repo.accountsByID[asset.ID] = asset
	repo.accountsByID[gain.ID] = gain
	repo.accountsByID[loss.ID] = loss
	repo.journal = &Journal{ID: uuid.New(), IsActive: true}
	repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
	provider := &mockFXRevaluationProvider{snapshot: &FXRevaluationSnapshot{
		JournalID: repo.journal.ID, GainAccountID: gain.ID, LossAccountID: loss.ID, BaseCurrency: "SAR",
		MonetaryPositions: []FXMonetaryPosition{{
			AccountID: asset.ID, AccountType: AccountTypeAsset, CurrencyID: "USD",
			ForeignAmount: decimal.NewFromInt(100), FunctionalCarryingAmount: decimal.NewFromInt(350),
			ClosingRate: decimal.NewFromFloat(3.75),
		}},
	}}
	response, err := NewRevalueFXUseCase(repo, provider, nil).Execute(context.Background(), time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.State != string(MoveStateDraft) || len(response.Lines) != 2 ||
		response.Lines[0].Debit.String() != "25" || response.Lines[1].Credit.String() != "25" {
		t.Fatalf("FX revaluation draft = %+v", response)
	}
	if _, err := NewRevalueFXUseCase(repo, nil, nil).Execute(context.Background(), time.Now()); err == nil {
		t.Fatal("Execute() expected an error when the FX provider is missing")
	}
}

func TestRepositoryFXRevaluationProviderBuildsSnapshot(t *testing.T) {
	repo := newMockAccountingRepository()
	assetID, liabilityID := uuid.New(), uuid.New()
	repo.fxBalances = []*FXMonetaryBalance{
		{
			AccountID: assetID, AccountType: AccountTypeAsset, CurrencyID: " usd ",
			ForeignAmount: decimal.NewFromInt(100), FunctionalCarryingAmount: decimal.NewFromInt(350),
		},
		{
			AccountID: liabilityID, AccountType: AccountTypeLiability, CurrencyID: "USD",
			ForeignAmount: decimal.NewFromInt(-100), FunctionalCarryingAmount: decimal.NewFromInt(-350),
		},
		{
			AccountID: uuid.New(), AccountType: AccountTypeAsset, CurrencyID: "SAR",
			ForeignAmount: decimal.NewFromInt(20), FunctionalCarryingAmount: decimal.NewFromInt(20),
		},
	}
	rates := &mockFXRateProvider{rates: map[string]decimal.Decimal{"USD/SAR": decimal.RequireFromString("3.75")}}
	provider, err := NewRepositoryFXRevaluationProvider(repo, rates, FXRevaluationSettings{
		JournalID: uuid.New(), GainAccountID: uuid.New(), LossAccountID: uuid.New(), BaseCurrency: "sar",
	})
	if err != nil {
		t.Fatalf("NewRepositoryFXRevaluationProvider() error = %v", err)
	}
	asOf := time.Date(2026, 11, 3, 17, 0, 0, 0, time.FixedZone("local", 2*60*60))
	snapshot, err := provider.GetFXRevaluationSnapshot(context.Background(), asOf)
	if err != nil {
		t.Fatalf("GetFXRevaluationSnapshot() error = %v", err)
	}
	if snapshot.BaseCurrency != "SAR" || len(snapshot.MonetaryPositions) != 2 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if !rates.lastAsOf.Equal(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("rate lookup date = %v", rates.lastAsOf)
	}
	if snapshot.MonetaryPositions[0].AccountID != assetID || snapshot.MonetaryPositions[1].AccountID != liabilityID ||
		snapshot.MonetaryPositions[1].ForeignAmount.String() != "-100" {
		t.Fatalf("signed positions = %+v", snapshot.MonetaryPositions)
	}
}

func TestRepositoryFXRevaluationProviderRejectsMissingRate(t *testing.T) {
	repo := newMockAccountingRepository()
	repo.fxBalances = []*FXMonetaryBalance{{
		AccountID: uuid.New(), AccountType: AccountTypeAsset, CurrencyID: "USD",
		ForeignAmount: decimal.NewFromInt(100), FunctionalCarryingAmount: decimal.NewFromInt(350),
	}}
	provider, err := NewRepositoryFXRevaluationProvider(repo, &mockFXRateProvider{}, FXRevaluationSettings{
		JournalID: uuid.New(), GainAccountID: uuid.New(), LossAccountID: uuid.New(), BaseCurrency: "SAR",
	})
	if err != nil {
		t.Fatalf("NewRepositoryFXRevaluationProvider() error = %v", err)
	}
	if _, err := provider.GetFXRevaluationSnapshot(context.Background(), time.Now()); err == nil {
		t.Fatal("GetFXRevaluationSnapshot() expected missing rate error")
	}
}

func TestRevalueFXUseCaseUsesCorrectGainAndLossSides(t *testing.T) {
	tests := []struct {
		name        string
		accountType AccountType
		rate        string
		wantDebit   uuid.UUID
		wantCredit  uuid.UUID
	}{
		{name: "asset gain", accountType: AccountTypeAsset, rate: "1.1"},
		{name: "asset loss", accountType: AccountTypeAsset, rate: "0.9"},
		{name: "liability gain", accountType: AccountTypeLiability, rate: "0.9"},
		{name: "liability loss", accountType: AccountTypeLiability, rate: "1.1"},
	}
	for i := range tests {
		positionID, gainID, lossID := uuid.New(), uuid.New(), uuid.New()
		switch tests[i].name {
		case "asset gain":
			tests[i].wantDebit, tests[i].wantCredit = positionID, gainID
		case "asset loss":
			tests[i].wantDebit, tests[i].wantCredit = lossID, positionID
		case "liability gain":
			tests[i].wantDebit, tests[i].wantCredit = positionID, gainID
		case "liability loss":
			tests[i].wantDebit, tests[i].wantCredit = lossID, positionID
		}
		t.Run(tests[i].name, func(t *testing.T) {
			repo := newMockAccountingRepository()
			position := &Account{ID: positionID, Type: tests[i].accountType, IsActive: true}
			gain := &Account{ID: gainID, Type: AccountTypeIncome, IsActive: true}
			loss := &Account{ID: lossID, Type: AccountTypeExpense, IsActive: true}
			repo.accountsByID[position.ID] = position
			repo.accountsByID[gain.ID] = gain
			repo.accountsByID[loss.ID] = loss
			repo.journal = &Journal{ID: uuid.New(), IsActive: true}
			repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
			foreignAmount := decimal.NewFromInt(100)
			carryingAmount := decimal.NewFromInt(100)
			if tests[i].accountType == AccountTypeLiability {
				foreignAmount = foreignAmount.Neg()
				carryingAmount = carryingAmount.Neg()
			}
			provider := &mockFXRevaluationProvider{snapshot: &FXRevaluationSnapshot{
				JournalID: repo.journal.ID, GainAccountID: gain.ID, LossAccountID: loss.ID, BaseCurrency: "SAR",
				MonetaryPositions: []FXMonetaryPosition{{
					AccountID: position.ID, AccountType: tests[i].accountType, CurrencyID: "USD",
					ForeignAmount: foreignAmount, FunctionalCarryingAmount: carryingAmount,
					ClosingRate: decimal.RequireFromString(tests[i].rate),
				}},
			}}
			if _, err := NewRevalueFXUseCase(repo, provider, nil).Execute(context.Background(), time.Now()); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if repo.move.Lines[0].AccountID != tests[i].wantDebit || repo.move.Lines[0].Debit.IsZero() ||
				repo.move.Lines[1].AccountID != tests[i].wantCredit || repo.move.Lines[1].Credit.IsZero() {
				t.Fatalf("adjustment lines = %+v, want debit account %s and credit account %s", repo.move.Lines, tests[i].wantDebit, tests[i].wantCredit)
			}
		})
	}
}

func TestIngestSubledgerEventUseCaseIsIdempotent(t *testing.T) {
	repo := newMockAccountingRepository()
	debitAccountID, creditAccountID, journalID := uuid.New(), uuid.New(), uuid.New()
	repo.accountsByID[debitAccountID] = &Account{ID: debitAccountID, Type: AccountTypeExpense, IsActive: true}
	repo.accountsByID[creditAccountID] = &Account{ID: creditAccountID, Type: AccountTypeAsset, IsActive: true}
	repo.eventMappings["goods_delivered"] = &SubledgerEventMapping{
		EventType: "goods_delivered", JournalID: journalID,
		DebitAccountID: debitAccountID, CreditAccountID: creditAccountID, IsActive: true,
	}
	repo.journal = &Journal{ID: journalID, IsActive: true}
	repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
	request := IngestSubledgerEventRequest{
		IdempotencyKey: "event-001", EventType: "goods_delivered", Date: "2026-11-03",
		Amount: decimal.NewFromInt(50), Currency: "SAR",
	}
	uc := NewIngestSubledgerEventUseCase(repo, nil)
	first, err := uc.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	second, err := uc.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if first.ID != second.ID || first.State != string(MoveStatePosted) || repo.postNewCalls != 1 {
		t.Fatalf("idempotent responses = %s and %s; atomic create calls = %d", first.ID, second.ID, repo.postNewCalls)
	}
}

func TestIngestSubledgerEventUseCaseRejectsUnmappedEvent(t *testing.T) {
	repo := newMockAccountingRepository()
	request := IngestSubledgerEventRequest{
		IdempotencyKey: "unmapped-event-001", EventType: "unconfigured_event",
		Date: "2026-11-03", Amount: decimal.NewFromInt(50), Currency: "SAR",
	}

	_, err := NewIngestSubledgerEventUseCase(repo, nil).Execute(context.Background(), request)
	if !errors.Is(err, ErrSubledgerMappingNotFound) {
		t.Fatalf("Execute() error = %v, want ErrSubledgerMappingNotFound", err)
	}
	if repo.postNewCalls != 0 {
		t.Fatalf("unmapped event attempted %d postings, want none", repo.postNewCalls)
	}
}

func TestIngestSubledgerEventUseCaseRejectsForeignCurrencyWithoutRate(t *testing.T) {
	repo := newMockAccountingRepository()
	debitAccountID, creditAccountID, journalID := uuid.New(), uuid.New(), uuid.New()
	repo.accountsByID[debitAccountID] = &Account{ID: debitAccountID, Type: AccountTypeExpense, IsActive: true}
	repo.accountsByID[creditAccountID] = &Account{ID: creditAccountID, Type: AccountTypeAsset, IsActive: true}
	repo.eventMappings["goods_delivered"] = &SubledgerEventMapping{
		EventType: "goods_delivered", JournalID: journalID,
		DebitAccountID: debitAccountID, CreditAccountID: creditAccountID, IsActive: true,
	}
	repo.journal = &Journal{ID: journalID, IsActive: true}
	repo.period = &FiscalPeriod{ID: uuid.New(), State: PeriodStateOpen}
	request := IngestSubledgerEventRequest{
		IdempotencyKey: "missing-rate-event-001", EventType: "goods_delivered",
		Date: "2026-11-03", Amount: decimal.NewFromInt(50), Currency: "EUR",
	}

	_, err := NewIngestSubledgerEventUseCase(repo, nil).Execute(context.Background(), request)
	if !errors.Is(err, ErrExchangeRateMissing) {
		t.Fatalf("Execute() error = %v, want ErrExchangeRateMissing", err)
	}
	if repo.postNewCalls != 0 {
		t.Fatalf("event without an exchange rate attempted %d postings, want none", repo.postNewCalls)
	}
}

func balancedMove(state MoveState) *AccountMove {
	date := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	return &AccountMove{
		ID: uuid.New(), JournalID: uuid.New(), Date: date, State: state,
		Lines: []AccountMoveLine{
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.NewFromInt(25)},
			{ID: uuid.New(), AccountID: uuid.New(), Credit: decimal.NewFromInt(25)},
		},
	}
}

type mockEventPublisher struct {
	events []string
	err    error
}

func (p *mockEventPublisher) Publish(_ context.Context, eventName string, _ any) error {
	p.events = append(p.events, eventName)
	return p.err
}

type mockFXRevaluationProvider struct {
	snapshot *FXRevaluationSnapshot
	err      error
}

func (p *mockFXRevaluationProvider) GetFXRevaluationSnapshot(context.Context, time.Time) (*FXRevaluationSnapshot, error) {
	return p.snapshot, p.err
}

type mockFXRateProvider struct {
	rates    map[string]decimal.Decimal
	err      error
	lastAsOf time.Time
}

func (p *mockFXRateProvider) GetClosingRate(_ context.Context, foreignCurrency, baseCurrency string, asOf time.Time) (decimal.Decimal, error) {
	p.lastAsOf = asOf
	if p.err != nil {
		return decimal.Zero, p.err
	}
	rate, ok := p.rates[foreignCurrency+"/"+baseCurrency]
	if !ok {
		return decimal.Zero, errors.New("closing rate not found")
	}
	return rate, nil
}
