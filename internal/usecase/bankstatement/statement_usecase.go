package bankstatementusecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/bankstatement"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type AddStatementLineInput struct {
	Name           string    `json:"name"`
	Ref            string    `json:"ref"`
	Date           time.Time `json:"date"`
	Amount         float64   `json:"amount"`
	AmountCurrency float64   `json:"amount_currency"`
	Currency       string    `json:"currency"`
	PartnerID      *int64    `json:"partner_id"`
	AccountID      *int64    `json:"account_id"`
	Checked        bool      `json:"checked"`
	InternalIndex  string    `json:"internal_index"`
	ImportBatchID  string    `json:"import_batch_id"`
}

type CreateStatementInput struct {
	JournalID      int64                   `json:"journal_id"`
	PartnerID      *int64                  `json:"partner_id"`
	Date           time.Time               `json:"date"`
	BalanceStart   float64                 `json:"balance_start"`
	BalanceEndReal *float64                `json:"balance_end_real"`
	Currency       string                  `json:"currency"`
	Lines          []AddStatementLineInput `json:"lines"`
}

type UpdateStatementInput struct {
	PartnerID      *int64     `json:"partner_id"`
	Date           *time.Time `json:"date"`
	BalanceEndReal *float64   `json:"balance_end_real"`
}

type ReconcileResult struct {
	StatementLineID     int64                   `json:"statement_line_id"`
	CandidateLineID     int64                   `json:"candidate_line_id"`
	Amount              float64                 `json:"amount"`
	StatementReconciled bool                    `json:"statement_reconciled"`
	CandidateReconciled bool                    `json:"candidate_reconciled"`
	MatchingNumber      *string                 `json:"matching_number,omitempty"`
	PaymentState        accounting.PaymentState `json:"payment_state"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Statements
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateStatement(ctx context.Context, in CreateStatementInput) (*bankstatement.BankStatement, error) {
	if in.JournalID <= 0 {
		return nil, platformerrors.Validation("journal is required", map[string]string{"journal_id": "must reference a valid journal"})
	}
	if in.Date.IsZero() {
		in.Date = time.Now().UTC()
	}
	if in.Currency == "" {
		in.Currency = "USD"
	}

	journal, err := uc.accountingSvc.GetJournal(ctx, in.JournalID)
	if err != nil {
		return nil, platformerrors.Validation("invalid journal", map[string]string{"journal_id": fmt.Sprintf("journal with id %d does not exist", in.JournalID)})
	}
	if journal.Type != accounting.JournalTypeBank && journal.Type != accounting.JournalTypeCash {
		return nil, platformerrors.Validation("invalid journal type", map[string]string{
			"journal_id": "bank statements can only be created on 'bank' or 'cash' journals",
		})
	}

	name := ""
	if uc.sequences != nil {
		name, err = uc.sequences.NextStatementName(ctx, journal.Code, in.Date.Year())
		if err != nil {
			return nil, err
		}
	}
	if name == "" {
		name = fmt.Sprintf("%s Statement %04d", journal.Code, in.Date.Year())
	}

	st := &bankstatement.BankStatement{
		Name:           name,
		JournalID:      in.JournalID,
		PartnerID:      in.PartnerID,
		Date:           in.Date,
		BalanceStart:   roundTo4(in.BalanceStart),
		BalanceEndReal: in.BalanceEndReal,
		Currency:       in.Currency,
		State:          bankstatement.StatementStateOpen,
	}
	if err := uc.repo.CreateStatement(ctx, st); err != nil {
		return nil, err
	}

	if len(in.Lines) > 0 {
		if _, err := uc.AddLines(ctx, st.ID, in.Lines); err != nil {
			return nil, err
		}
	}

	uc.logger.InfoContext(ctx, "bank statement created", "id", st.ID, "name", st.Name)
	return uc.repo.GetStatementWithLines(ctx, st.ID)
}

func (uc *UseCase) GetStatement(ctx context.Context, id int64) (*bankstatement.BankStatement, error) {
	return uc.repo.GetStatementWithLines(ctx, id)
}

func (uc *UseCase) UpdateStatement(ctx context.Context, id int64, in UpdateStatementInput) (*bankstatement.BankStatement, error) {
	st, err := uc.repo.GetStatementByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if st.State != bankstatement.StatementStateOpen {
		return nil, platformerrors.Conflict("only open statements can be updated")
	}
	if in.PartnerID != nil {
		st.PartnerID = in.PartnerID
	}
	if in.Date != nil {
		st.Date = *in.Date
	}
	if in.BalanceEndReal != nil {
		st.BalanceEndReal = in.BalanceEndReal
	}
	if err := uc.repo.UpdateStatement(ctx, st); err != nil {
		return nil, err
	}
	return uc.repo.GetStatementWithLines(ctx, st.ID)
}

func (uc *UseCase) DeleteStatement(ctx context.Context, id int64) error {
	st, err := uc.repo.GetStatementByID(ctx, id)
	if err != nil {
		return err
	}
	if st.State != bankstatement.StatementStateOpen {
		return platformerrors.Conflict("only open statements can be deleted")
	}
	return uc.repo.DeleteStatement(ctx, id)
}

func (uc *UseCase) ListStatements(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[bankstatement.BankStatement], error) {
	return uc.repo.ListStatements(ctx, f, page)
}

// ─────────────────────────────────────────────────────────────────────────────
// Lines & Booking
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) AddLines(ctx context.Context, statementID int64, inputs []AddStatementLineInput) ([]bankstatement.BankStatementLine, error) {
	st, err := uc.repo.GetStatementByID(ctx, statementID)
	if err != nil {
		return nil, err
	}
	if st.State != bankstatement.StatementStateOpen {
		return nil, platformerrors.Conflict("lines can only be added to an open statement")
	}

	lines := make([]bankstatement.BankStatementLine, 0, len(inputs))
	for _, in := range inputs {
		if in.Amount == 0 {
			return nil, platformerrors.Validation("invalid line amount", map[string]string{"amount": "cannot be zero"})
		}
		if in.Date.IsZero() {
			in.Date = st.Date
		}
		if in.Currency == "" {
			in.Currency = st.Currency
		}
		lines = append(lines, bankstatement.BankStatementLine{
			Name:           in.Name,
			Ref:            in.Ref,
			Date:           in.Date,
			Amount:         roundTo4(in.Amount),
			AmountCurrency: roundTo4(in.AmountCurrency),
			Currency:       in.Currency,
			PartnerID:      in.PartnerID,
			AccountID:      in.AccountID,
			Checked:        in.Checked,
			InternalIndex:  in.InternalIndex,
			ImportBatchID:  in.ImportBatchID,
			AmountResidual: mathAbs(in.Amount),
		})
	}

	// Assign sequences + running balances, then persist (IDs assigned by the repo).
	st.Lines = lines
	st.ComputeTotals()
	if err := uc.repo.AddStatementLines(ctx, statementID, lines); err != nil {
		return nil, err
	}

	// Book each line as a posted dual-entry journal move.
	for i := range lines {
		if err := uc.bookLine(ctx, st, &lines[i]); err != nil {
			return nil, err
		}
	}

	// Recompute statement completeness and persist.
	st.Lines = lines
	st.ComputeCompleteness()
	if err := uc.repo.UpdateStatement(ctx, st); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "statement lines added", "statement_id", statementID, "count", len(lines))
	return lines, nil
}

// bookLine books a statement line into a posted journal entry:
// liquidity leg on the journal default account, counterpart leg on the
// provided account, the partner receivable/payable, or the suspense account.
func (uc *UseCase) bookLine(ctx context.Context, st *bankstatement.BankStatement, line *bankstatement.BankStatementLine) error {
	journal, err := uc.accountingSvc.GetJournal(ctx, st.JournalID)
	if err != nil {
		return err
	}
	if journal.DefaultAccountID == nil || *journal.DefaultAccountID <= 0 {
		return platformerrors.Validation("journal has no default account", map[string]string{
			"journal_id": fmt.Sprintf("journal %d must define a default (liquidity) account", st.JournalID),
		})
	}

	counterpartAccount, err := uc.resolveCounterpartAccount(ctx, line, journal)
	if err != nil {
		return err
	}

	amount := mathAbs(line.Amount)
	var (
		liquidityDebit, liquidityCredit     float64
		counterpartDebit, counterpartCredit float64
	)
	if line.Amount > 0 {
		liquidityDebit = amount
		counterpartCredit = amount
	} else {
		liquidityCredit = amount
		counterpartDebit = amount
	}

	entry, err := uc.accountingSvc.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
		JournalID: st.JournalID,
		Date:      line.Date,
		Ref:       line.Name,
		Lines: []accountingusecase.JournalEntryLineInput{
			{
				AccountID:       *journal.DefaultAccountID,
				Name:            line.Name,
				Debit:           liquidityDebit,
				Credit:          liquidityCredit,
				StatementLineID: &line.ID,
			},
			{
				AccountID: counterpartAccount,
				PartnerID: line.PartnerID,
				Name:      line.Name,
				Debit:     counterpartDebit,
				Credit:    counterpartCredit,
			},
		},
	})
	if err != nil {
		return err
	}

	if _, err := uc.accountingSvc.PostMove(ctx, entry.ID); err != nil {
		return err
	}

	line.MoveID = &entry.ID
	return uc.repo.UpdateStatementLine(ctx, line)
}

func (uc *UseCase) resolveCounterpartAccount(ctx context.Context, line *bankstatement.BankStatementLine, journal *accounting.Journal) (int64, error) {
	if line.AccountID != nil && *line.AccountID > 0 {
		return *line.AccountID, nil
	}
	// Money in -> customer receivable; money out -> vendor payable.
	if line.PartnerID != nil && *line.PartnerID > 0 {
		code := "120000"
		if line.Amount < 0 {
			code = "210000"
		}
		if acc, err := uc.accountingSvc.GetAccountByCode(ctx, code); err == nil {
			return acc.ID, nil
		}
	}
	if journal.SuspenseAccountID != nil && *journal.SuspenseAccountID > 0 {
		return *journal.SuspenseAccountID, nil
	}
	return 0, platformerrors.Validation("statement line needs an account", map[string]string{
		"account_id": "provide an account, a partner, or a journal suspense account to book against",
	})
}

func (uc *UseCase) ConfirmStatement(ctx context.Context, id int64) (*bankstatement.BankStatement, error) {
	st, err := uc.repo.GetStatementWithLines(ctx, id)
	if err != nil {
		return nil, err
	}
	if st.State == bankstatement.StatementStateConfirm {
		return st, nil
	}
	var nonBooked []string
	for _, l := range st.Lines {
		if !l.IsComplete() {
			nonBooked = append(nonBooked, fmt.Sprintf("line %d", l.ID))
		}
	}
	if len(nonBooked) > 0 {
		return nil, platformerrors.Validation("cannot confirm statement with unbooked lines", map[string]string{
			"lines": strings.Join(nonBooked, ", "),
		})
	}
	st.State = bankstatement.StatementStateConfirm
	if err := uc.repo.UpdateStatement(ctx, st); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "bank statement confirmed", "id", id)
	return uc.repo.GetStatementWithLines(ctx, id)
}

// ImportCSV parses a simplified CSV (utf-8) into statement lines and books them.
// Expected columns, in order: date, name, ref, amount, currency (header row optional).
func (uc *UseCase) ImportCSV(ctx context.Context, statementID int64, data []byte) ([]bankstatement.BankStatementLine, error) {
	st, err := uc.repo.GetStatementByID(ctx, statementID)
	if err != nil {
		return nil, err
	}

	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		return nil, platformerrors.BadRequest("could not parse CSV", err)
	}
	if len(rows) == 0 {
		return nil, platformerrors.Validation("csv is empty", map[string]string{"csv": "must contain at least one data row"})
	}

	var inputs []AddStatementLineInput
	for i, row := range rows {
		if i == 0 && len(row) > 0 && strings.Contains(strings.ToLower(row[0]), "date") {
			continue // header row
		}
		if len(row) < 4 {
			return nil, platformerrors.Validation("invalid csv row", map[string]string{
				"row": fmt.Sprintf("row %d must have at least 4 columns: date,name,ref,amount,currency", i+1),
			})
		}

		var date time.Time
		date, err = parseCSVDate(strings.TrimSpace(row[0]))
		if err != nil {
			date = st.Date
		}
		amount, err := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(row[3], ",", "")), 64)
		if err != nil {
			return nil, platformerrors.Validation(fmt.Sprintf("invalid amount on row %d", i+1), map[string]string{
				"amount": row[3],
			})
		}
		currency := st.Currency
		if len(row) > 4 && strings.TrimSpace(row[4]) != "" {
			currency = strings.TrimSpace(row[4])
		}

		inputs = append(inputs, AddStatementLineInput{
			Name:          strings.TrimSpace(row[1]),
			Ref:           strings.TrimSpace(row[2]),
			Date:          date,
			Amount:        amount,
			Currency:      currency,
			ImportBatchID: fmt.Sprintf("csv-%d", i+1),
		})
	}

	uc.logger.InfoContext(ctx, "csv import parsed", "statement_id", statementID, "rows", len(inputs))
	return uc.AddLines(ctx, statementID, inputs)
}

func parseCSVDate(v string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "02/01/2006", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format %q", v)
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
