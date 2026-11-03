package accounting_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/features/accounting"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCreateAccountRequest_Validate(t *testing.T) {
	t.Run("normalizes values and defaults the natural balance and currency", func(t *testing.T) {
		request := accounting.CreateAccountRequest{
			Code: " 1000 ",
			Name: " Cash ",
			Type: " ASSET ",
		}

		if err := request.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if request.Code != "1000" || request.Name != "Cash" || request.Type != "asset" {
			t.Fatalf("Validate() normalized request = %+v", request)
		}
		if request.NormalBalance != "debit" || request.CurrencyID != "SAR" {
			t.Fatalf("Validate() defaults = normal balance %q, currency %q", request.NormalBalance, request.CurrencyID)
		}
	})

	t.Run("defaults credit natural balance for liability accounts", func(t *testing.T) {
		request := accounting.CreateAccountRequest{Code: "2000", Name: "Payables", Type: "liability"}
		if err := request.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if request.NormalBalance != "credit" {
			t.Fatalf("normal balance = %q, want credit", request.NormalBalance)
		}
	})

	tests := []struct {
		name    string
		request accounting.CreateAccountRequest
		wantErr string
	}{
		{name: "missing code", request: accounting.CreateAccountRequest{Name: "Cash", Type: "asset"}, wantErr: "code is required"},
		{name: "missing name", request: accounting.CreateAccountRequest{Code: "1000", Type: "asset"}, wantErr: "name is required"},
		{name: "invalid account type", request: accounting.CreateAccountRequest{Code: "1000", Name: "Cash", Type: "unknown"}, wantErr: "invalid account_type: must be asset, liability, equity, income, or expense"},
		{name: "invalid normal balance", request: accounting.CreateAccountRequest{Code: "1000", Name: "Cash", Type: "asset", NormalBalance: "sideways"}, wantErr: "normal_balance must be debit or credit"},
		{name: "unsupported currency", request: accounting.CreateAccountRequest{Code: "1000", Name: "Cash", Type: "asset", CurrencyID: "XYZ"}, wantErr: "currency_id must be a supported ISO 4217 currency code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestCreateFiscalYearRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request accounting.CreateFiscalYearRequest
		wantErr string
	}{
		{name: "valid date range", request: accounting.CreateFiscalYearRequest{Year: 2024, StartDate: "2024-01-01", EndDate: "2024-12-31"}},
		{name: "year too low", request: accounting.CreateFiscalYearRequest{Year: 1999, StartDate: "2024-01-01", EndDate: "2024-12-31"}, wantErr: "year must be between 2000 and 2100"},
		{name: "year too high", request: accounting.CreateFiscalYearRequest{Year: 2101, StartDate: "2024-01-01", EndDate: "2024-12-31"}, wantErr: "year must be between 2000 and 2100"},
		{name: "invalid start date", request: accounting.CreateFiscalYearRequest{Year: 2024, StartDate: "2024-02-30", EndDate: "2024-12-31"}, wantErr: "start_date must be in YYYY-MM-DD format"},
		{name: "invalid date layout", request: accounting.CreateFiscalYearRequest{Year: 2024, StartDate: "2024-1-01", EndDate: "2024-12-31"}, wantErr: "start_date must be in YYYY-MM-DD format"},
		{name: "invalid end date", request: accounting.CreateFiscalYearRequest{Year: 2024, StartDate: "2024-01-01", EndDate: "not-a-date"}, wantErr: "end_date must be in YYYY-MM-DD format"},
		{name: "end date is not after start date", request: accounting.CreateFiscalYearRequest{Year: 2024, StartDate: "2024-01-01", EndDate: "2024-01-01"}, wantErr: "end_date must be strictly after start_date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestChangePeriodStateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		want    string
		wantErr string
	}{
		{name: "normalizes valid state", state: " SOFT_CLOSED ", want: "soft_closed"},
		{name: "accepts open state", state: "open", want: "open"},
		{name: "rejects invalid state", state: "closed", wantErr: "state must be open, soft_closed, or hard_closed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := accounting.ChangePeriodStateRequest{State: tt.state}
			err := request.Validate()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if request.State != tt.want {
				t.Fatalf("normalized state = %q, want %q", request.State, tt.want)
			}
		})
	}
}

func TestCreateMoveRequest_Validate(t *testing.T) {
	newValidRequest := func() accounting.CreateMoveRequest {
		return accounting.CreateMoveRequest{
			JournalID: uuid.New(),
			Date:      "2025-06-15",
			Lines: []accounting.MoveLineRequest{
				{AccountID: uuid.New(), Debit: decimal.NewFromInt(25), CurrencyID: " usd "},
				{AccountID: uuid.New(), Credit: decimal.NewFromInt(25)},
			},
		}
	}

	t.Run("accepts a balanced move and normalizes currencies", func(t *testing.T) {
		request := newValidRequest()
		if err := request.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if request.Lines[0].CurrencyID != "USD" || request.Lines[1].CurrencyID != "SAR" {
			t.Fatalf("normalized currencies = %q, %q", request.Lines[0].CurrencyID, request.Lines[1].CurrencyID)
		}
	})

	tests := []struct {
		name    string
		mutate  func(*accounting.CreateMoveRequest)
		wantErr string
	}{
		{name: "missing journal", mutate: func(r *accounting.CreateMoveRequest) { r.JournalID = uuid.Nil }, wantErr: "journal_id is required"},
		{name: "invalid date", mutate: func(r *accounting.CreateMoveRequest) { r.Date = "2025-2-15" }, wantErr: "date must be in YYYY-MM-DD format"},
		{name: "too few lines", mutate: func(r *accounting.CreateMoveRequest) { r.Lines = r.Lines[:1] }, wantErr: "a move must contain at least 2 lines"},
		{name: "missing account", mutate: func(r *accounting.CreateMoveRequest) { r.Lines[0].AccountID = uuid.Nil }, wantErr: "account_id is required on all lines"},
		{name: "negative amount", mutate: func(r *accounting.CreateMoveRequest) { r.Lines[0].Debit = decimal.NewFromInt(-25) }, wantErr: "negative debit or credit amounts are not permitted"},
		{name: "zero amount line", mutate: func(r *accounting.CreateMoveRequest) { r.Lines[0].Debit = decimal.Zero }, wantErr: "line must specify non-zero debit or credit"},
		{name: "both debit and credit on one line", mutate: func(r *accounting.CreateMoveRequest) { r.Lines[0].Credit = decimal.NewFromInt(1) }, wantErr: "line cannot have both debit and credit; split into separate lines"},
		{name: "unsupported currency", mutate: func(r *accounting.CreateMoveRequest) { r.Lines[0].CurrencyID = "XYZ" }, wantErr: "currency_id must be a supported ISO 4217 currency code"},
		{name: "unbalanced totals", mutate: func(r *accounting.CreateMoveRequest) { r.Lines[1].Credit = decimal.NewFromInt(24) }, wantErr: "total debit must equal total credit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := newValidRequest()
			tt.mutate(&request)
			err := request.Validate()
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestIngestSubledgerEventRequest_Validate(t *testing.T) {
	newValidRequest := func() accounting.IngestSubledgerEventRequest {
		return accounting.IngestSubledgerEventRequest{
			IdempotencyKey: " key-123 ",
			EventType:      " order_delivered ",
			Date:           "2025-06-15",
			Amount:         decimal.NewFromInt(25),
			Currency:       "usd",
		}
	}

	t.Run("normalizes required strings and currency", func(t *testing.T) {
		request := newValidRequest()
		if err := request.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if request.IdempotencyKey != "key-123" || request.EventType != "order_delivered" || request.Currency != "USD" {
			t.Fatalf("Validate() normalized request = %+v", request)
		}
	})

	tests := []struct {
		name    string
		mutate  func(*accounting.IngestSubledgerEventRequest)
		wantErr string
	}{
		{name: "missing idempotency key", mutate: func(r *accounting.IngestSubledgerEventRequest) { r.IdempotencyKey = " " }, wantErr: "idempotency_key is required"},
		{name: "missing event type", mutate: func(r *accounting.IngestSubledgerEventRequest) { r.EventType = " " }, wantErr: "event_type is required"},
		{name: "zero amount", mutate: func(r *accounting.IngestSubledgerEventRequest) { r.Amount = decimal.Zero }, wantErr: "amount must be strictly positive"},
		{name: "negative amount", mutate: func(r *accounting.IngestSubledgerEventRequest) { r.Amount = decimal.NewFromInt(-1) }, wantErr: "amount must be strictly positive"},
		{name: "invalid date", mutate: func(r *accounting.IngestSubledgerEventRequest) { r.Date = "2025-02-30" }, wantErr: "date must be in YYYY-MM-DD format"},
		{name: "unsupported currency", mutate: func(r *accounting.IngestSubledgerEventRequest) { r.Currency = "XYZ" }, wantErr: "currency must be a supported ISO 4217 currency code"},
		{name: "missing currency", mutate: func(r *accounting.IngestSubledgerEventRequest) { r.Currency = "" }, wantErr: "currency must be a supported ISO 4217 currency code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := newValidRequest()
			tt.mutate(&request)
			err := request.Validate()
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestDTO_Mappers(t *testing.T) {
	t.Run("account response", func(t *testing.T) {
		parentID := uuid.New()
		createdAt := time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)
		account := &accounting.Account{
			ID:             uuid.New(),
			Code:           "1000",
			Name:           "Cash",
			Type:           accounting.AccountTypeAsset,
			NormalBalance:  accounting.NormalBalanceDebit,
			CurrencyID:     "SAR",
			ParentID:       &parentID,
			IsReconcilable: true,
			IsActive:       true,
			CreatedAt:      createdAt,
		}

		response := accounting.ToAccountResponse(account)
		if response.ID != account.ID || response.Code != account.Code || response.Type != "asset" || response.CreatedAt != createdAt {
			t.Fatalf("ToAccountResponse() = %+v", response)
		}
		if response.ParentID == nil || *response.ParentID != parentID {
			t.Fatalf("parent id = %v, want %s", response.ParentID, parentID)
		}
		*response.ParentID = uuid.New()
		if *account.ParentID != parentID {
			t.Fatal("response parent ID aliases the entity")
		}
	})

	t.Run("fiscal period response", func(t *testing.T) {
		closedAt := time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC)
		period := &accounting.FiscalPeriod{
			ID:           uuid.New(),
			FiscalYearID: uuid.New(),
			PeriodNumber: 3,
			Name:         "March 2025",
			StartDate:    time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      time.Date(2025, time.March, 31, 0, 0, 0, 0, time.UTC),
			State:        accounting.PeriodStateSoftClosed,
			ClosedAt:     &closedAt,
		}

		response := accounting.ToFiscalPeriodResponse(period)
		if response.StartDate != "2025-03-01" || response.EndDate != "2025-03-31" || response.State != "soft_closed" {
			t.Fatalf("ToFiscalPeriodResponse() = %+v", response)
		}
		if response.ClosedAt == nil || !response.ClosedAt.Equal(closedAt) {
			t.Fatalf("closed at = %v, want %v", response.ClosedAt, closedAt)
		}
		*response.ClosedAt = response.ClosedAt.Add(time.Hour)
		if !period.ClosedAt.Equal(closedAt) {
			t.Fatal("response closed_at aliases the entity")
		}
	})

	t.Run("move response", func(t *testing.T) {
		reversedMoveID := uuid.New()
		costCenterID := uuid.New()
		postedAt := time.Date(2025, time.June, 15, 12, 0, 0, 0, time.UTC)
		move := &accounting.AccountMove{
			ID:             uuid.New(),
			Name:           "MVE/2025/0001",
			JournalID:      uuid.New(),
			Date:           time.Date(2025, time.June, 15, 0, 0, 0, 0, time.UTC),
			State:          accounting.MoveStatePosted,
			ReversedMoveID: &reversedMoveID,
			PostedAt:       &postedAt,
			Lines: []accounting.AccountMoveLine{{
				ID:             uuid.New(),
				AccountID:      uuid.New(),
				Debit:          decimal.NewFromInt(25),
				AmountCurrency: decimal.NewFromInt(25),
				CurrencyID:     "SAR",
				CostCenterID:   &costCenterID,
			}},
		}

		response := accounting.ToMoveResponse(move)
		if response.Date != "2025-06-15" || response.State != "posted" || len(response.Lines) != 1 {
			t.Fatalf("ToMoveResponse() = %+v", response)
		}
		if response.ReversedMoveID == nil || *response.ReversedMoveID != reversedMoveID || response.Lines[0].CostCenterID == nil || *response.Lines[0].CostCenterID != costCenterID {
			t.Fatalf("mapped optional IDs = %+v", response)
		}
		if response.PostedAt == nil || !response.PostedAt.Equal(postedAt) {
			t.Fatalf("posted at = %v, want %v", response.PostedAt, postedAt)
		}
		*response.Lines[0].CostCenterID = uuid.New()
		if *move.Lines[0].CostCenterID != costCenterID {
			t.Fatal("response cost center ID aliases the entity")
		}

		emptyMoveResponse := accounting.ToMoveResponse(&accounting.AccountMove{})
		if emptyMoveResponse.Lines == nil || len(emptyMoveResponse.Lines) != 0 {
			t.Fatalf("empty move lines = %#v, want non-nil empty slice", emptyMoveResponse.Lines)
		}
	})

	t.Run("trial balance row response", func(t *testing.T) {
		row := &accounting.TrialBalanceRow{
			AccountID:     uuid.New(),
			AccountCode:   "1000",
			AccountName:   "Cash",
			InitialDebit:  decimal.NewFromInt(10),
			InitialCredit: decimal.NewFromInt(2),
			PeriodDebit:   decimal.NewFromInt(5),
			PeriodCredit:  decimal.NewFromInt(1),
			EndingDebit:   decimal.NewFromInt(13),
			EndingCredit:  decimal.NewFromInt(0),
		}
		response := accounting.ToTrialBalanceRowResponse(row)
		if response.AccountID != row.AccountID || response.AccountName != row.AccountName || !response.EndingDebit.Equal(row.EndingDebit) {
			t.Fatalf("ToTrialBalanceRowResponse() = %+v", response)
		}
	})

	t.Run("trial balance response", func(t *testing.T) {
		fromDate := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		toDate := time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)
		row := &accounting.TrialBalanceRow{
			AccountID:   uuid.New(),
			AccountCode: "1000",
			AccountName: "Cash",
			EndingDebit: decimal.NewFromInt(25),
		}

		response := accounting.ToTrialBalanceResponse(
			fromDate,
			toDate,
			decimal.NewFromInt(25),
			decimal.NewFromInt(25),
			[]*accounting.TrialBalanceRow{row},
		)
		if response.FromDate != "2025-01-01" || response.ToDate != "2025-12-31" ||
			!response.TotalDebit.Equal(decimal.NewFromInt(25)) ||
			!response.TotalCredit.Equal(decimal.NewFromInt(25)) ||
			!response.IsBalanced || len(response.Accounts) != 1 ||
			response.Accounts[0].AccountID != row.AccountID {
			t.Fatalf("ToTrialBalanceResponse() = %+v", response)
		}

		unbalanced := accounting.ToTrialBalanceResponse(fromDate, toDate, decimal.NewFromInt(25), decimal.NewFromInt(24), nil)
		if unbalanced.IsBalanced {
			t.Fatal("ToTrialBalanceResponse() marked unequal totals as balanced")
		}
		if unbalanced.Accounts == nil {
			t.Fatal("ToTrialBalanceResponse() accounts = nil, want non-nil empty slice")
		}
	})

	t.Run("general ledger line response", func(t *testing.T) {
		line := &accounting.GeneralLedgerLineRow{
			MoveID:         uuid.New(),
			MoveName:       "MVE/2025/0001",
			Date:           time.Date(2025, time.June, 15, 9, 0, 0, 0, time.UTC),
			Ref:            "invoice-1",
			Debit:          decimal.NewFromInt(25),
			RunningBalance: decimal.NewFromInt(25),
		}
		response := accounting.ToGeneralLedgerLineResponse(line)
		if response.Date != "2025-06-15" || response.MoveID != line.MoveID || !response.RunningBalance.Equal(line.RunningBalance) {
			t.Fatalf("ToGeneralLedgerLineResponse() = %+v", response)
		}
	})
}
