package bankstatementhttp

import (
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/bankstatement"
	bankstatementusecase "cashflow_backend/internal/usecase/bankstatement"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request DTOs
// ─────────────────────────────────────────────────────────────────────────────

type AddStatementLineRequest struct {
	Name           string     `json:"name"`
	Ref            string     `json:"ref"`
	Date           *time.Time `json:"date"`
	Amount         float64    `json:"amount"`
	AmountCurrency float64    `json:"amount_currency"`
	Currency       string     `json:"currency"`
	PartnerID      *int64     `json:"partner_id"`
	AccountID      *int64     `json:"account_id"`
	Checked        bool       `json:"checked"`
	InternalIndex  string     `json:"internal_index"`
	ImportBatchID  string     `json:"import_batch_id"`
}

func (r AddStatementLineRequest) ToInput() bankstatementusecase.AddStatementLineInput {
	var d time.Time
	if r.Date != nil {
		d = *r.Date
	}
	return bankstatementusecase.AddStatementLineInput{
		Name:           r.Name,
		Ref:            r.Ref,
		Date:           d,
		Amount:         r.Amount,
		AmountCurrency: r.AmountCurrency,
		Currency:       r.Currency,
		PartnerID:      r.PartnerID,
		AccountID:      r.AccountID,
		Checked:        r.Checked,
		InternalIndex:  r.InternalIndex,
		ImportBatchID:  r.ImportBatchID,
	}
}

type CreateStatementRequest struct {
	JournalID      int64                     `json:"journal_id"`
	PartnerID      *int64                    `json:"partner_id"`
	Date           *time.Time                `json:"date"`
	BalanceStart   float64                   `json:"balance_start"`
	BalanceEndReal *float64                  `json:"balance_end_real"`
	Currency       string                    `json:"currency"`
	Lines          []AddStatementLineRequest `json:"lines"`
}

func (r CreateStatementRequest) ToInput() bankstatementusecase.CreateStatementInput {
	var d time.Time
	if r.Date != nil {
		d = *r.Date
	}
	lines := make([]bankstatementusecase.AddStatementLineInput, 0, len(r.Lines))
	for _, l := range r.Lines {
		lines = append(lines, l.ToInput())
	}
	return bankstatementusecase.CreateStatementInput{
		JournalID:      r.JournalID,
		PartnerID:      r.PartnerID,
		Date:           d,
		BalanceStart:   r.BalanceStart,
		BalanceEndReal: r.BalanceEndReal,
		Currency:       r.Currency,
		Lines:          lines,
	}
}

type UpdateStatementRequest struct {
	PartnerID      *int64     `json:"partner_id"`
	Date           *time.Time `json:"date"`
	BalanceEndReal *float64   `json:"balance_end_real"`
}

func (r UpdateStatementRequest) ToInput() bankstatementusecase.UpdateStatementInput {
	return bankstatementusecase.UpdateStatementInput{
		PartnerID:      r.PartnerID,
		Date:           r.Date,
		BalanceEndReal: r.BalanceEndReal,
	}
}

type AddLinesRequest struct {
	Lines []AddStatementLineRequest `json:"lines"`
}

type ReconcileLineRequest struct {
	CandidateLineID int64 `json:"candidate_line_id"`
}

type ReconcileModelRequest struct {
	Name            string                              `json:"name"`
	Sequence        int                                 `json:"sequence"`
	IsAutoReconcile bool                                `json:"is_auto_reconcile"`
	MatchNature     bankstatement.ReconcileMatchNature  `json:"match_nature"`
	MatchAmount     *bankstatement.ReconcileMatchAmount `json:"match_amount"`
	MatchAmountMin  *float64                            `json:"match_amount_min"`
	MatchAmountMax  *float64                            `json:"match_amount_max"`
	MatchLabel      *bankstatement.ReconcileMatchLabel  `json:"match_label"`
	MatchLabelParam string                              `json:"match_label_param"`
	MatchJournalIDs []int64                             `json:"match_journal_ids"`
	MatchPartnerIDs []int64                             `json:"match_partner_ids"`
	MappedPartnerID *int64                              `json:"mapped_partner_id"`
	Lines           []ReconcileModelLineRequest         `json:"lines"`
}

type ReconcileModelLineRequest struct {
	AmountType bankstatement.ReconcileLineAmountType `json:"amount_type"`
	Amount     string                                `json:"amount"`
	AccountID  int64                                 `json:"account_id"`
	Label      string                                `json:"label"`
	TaxIDs     []int64                               `json:"tax_ids"`
}

func (r ReconcileModelRequest) ToModel() *bankstatement.ReconcileModel {
	lines := make([]bankstatement.ReconcileModelLine, 0, len(r.Lines))
	for _, l := range r.Lines {
		lines = append(lines, bankstatement.ReconcileModelLine{
			AmountType: l.AmountType,
			Amount:     l.Amount,
			AccountID:  l.AccountID,
			Label:      l.Label,
			TaxIDs:     l.TaxIDs,
		})
	}
	return &bankstatement.ReconcileModel{
		Name:            r.Name,
		Sequence:        r.Sequence,
		IsAutoReconcile: r.IsAutoReconcile,
		MatchNature:     r.MatchNature,
		MatchAmount:     r.MatchAmount,
		MatchAmountMin:  r.MatchAmountMin,
		MatchAmountMax:  r.MatchAmountMax,
		MatchLabel:      r.MatchLabel,
		MatchLabelParam: r.MatchLabelParam,
		MatchJournalIDs: r.MatchJournalIDs,
		MatchPartnerIDs: r.MatchPartnerIDs,
		MappedPartnerID: r.MappedPartnerID,
		Lines:           lines,
	}
}

type CashRoundingRequest struct {
	Name            string                         `json:"name"`
	RoundingMethod  bankstatement.RoundingMethod   `json:"rounding_method"`
	Rounding        float64                        `json:"rounding"`
	Strategy        bankstatement.RoundingStrategy `json:"strategy"`
	ProfitAccountID *int64                         `json:"profit_account_id"`
	LossAccountID   *int64                         `json:"loss_account_id"`
}

func (r CashRoundingRequest) ToCashRounding() *bankstatement.CashRounding {
	return &bankstatement.CashRounding{
		Name:            r.Name,
		RoundingMethod:  r.RoundingMethod,
		Rounding:        r.Rounding,
		Strategy:        r.Strategy,
		ProfitAccountID: r.ProfitAccountID,
		LossAccountID:   r.LossAccountID,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Response DTOs
// ─────────────────────────────────────────────────────────────────────────────

type StatementLineResponse struct {
	ID             int64   `json:"id"`
	StatementID    int64   `json:"statement_id"`
	Name           string  `json:"name"`
	Ref            string  `json:"ref"`
	Sequence       int     `json:"sequence"`
	Date           string  `json:"date"`
	Amount         float64 `json:"amount"`
	AmountCurrency float64 `json:"amount_currency"`
	Currency       string  `json:"currency"`
	PartnerID      *int64  `json:"partner_id"`
	AccountID      *int64  `json:"account_id"`
	MoveID         *int64  `json:"move_id"`
	JournalID      *int64  `json:"journal_id"`
	Checked        bool    `json:"checked"`
	RunningBalance float64 `json:"running_balance"`
	AmountResidual float64 `json:"amount_residual"`
	Reconciled     bool    `json:"reconciled"`
	MatchingNumber *string `json:"matching_number,omitempty"`
	InternalIndex  string  `json:"internal_index"`
	ImportBatchID  string  `json:"import_batch_id"`
}

func ToStatementLineResponse(l *bankstatement.BankStatementLine) StatementLineResponse {
	resp := StatementLineResponse{
		ID:             l.ID,
		StatementID:    l.StatementID,
		Name:           l.Name,
		Ref:            l.Ref,
		Sequence:       l.Sequence,
		Date:           l.Date.Format("2006-01-02"),
		Amount:         l.Amount,
		AmountCurrency: l.AmountCurrency,
		Currency:       l.Currency,
		PartnerID:      l.PartnerID,
		AccountID:      l.AccountID,
		MoveID:         l.MoveID,
		JournalID:      l.JournalID,
		Checked:        l.Checked,
		RunningBalance: l.RunningBalance,
		AmountResidual: l.AmountResidual,
		Reconciled:     l.Reconciled,
		MatchingNumber: l.MatchingNumber,
		InternalIndex:  l.InternalIndex,
		ImportBatchID:  l.ImportBatchID,
	}
	if l.Ref == "" {
		resp.Ref = l.Ref
	}
	return resp
}

type StatementResponse struct {
	ID                 int64                        `json:"id"`
	Name               string                       `json:"name"`
	JournalID          int64                        `json:"journal_id"`
	PartnerID          *int64                       `json:"partner_id"`
	Date               string                       `json:"date"`
	BalanceStart       float64                      `json:"balance_start"`
	BalanceEnd         float64                      `json:"balance_end"`
	BalanceEndReal     *float64                     `json:"balance_end_real"`
	Currency           string                       `json:"currency"`
	State              bankstatement.StatementState `json:"state"`
	IsComplete         bool                         `json:"is_complete"`
	IsValid            bool                         `json:"is_valid"`
	ProblemDescription string                       `json:"problem_description"`
	Lines              []StatementLineResponse      `json:"lines"`
}

func ToStatementResponse(s *bankstatement.BankStatement) StatementResponse {
	resp := StatementResponse{
		ID:                 s.ID,
		Name:               s.Name,
		JournalID:          s.JournalID,
		PartnerID:          s.PartnerID,
		Date:               s.Date.Format("2006-01-02"),
		BalanceStart:       s.BalanceStart,
		BalanceEnd:         s.BalanceEnd,
		BalanceEndReal:     s.BalanceEndReal,
		Currency:           s.Currency,
		State:              s.State,
		IsComplete:         s.IsComplete,
		IsValid:            s.IsValid,
		ProblemDescription: s.ProblemDescription,
		Lines:              make([]StatementLineResponse, 0, len(s.Lines)),
	}
	for i := range s.Lines {
		resp.Lines = append(resp.Lines, ToStatementLineResponse(&s.Lines[i]))
	}
	return resp
}

type CandidateResponse struct {
	MoveLine        accounting.AccountMoveLine `json:"move_line"`
	SuggestedAmount float64                    `json:"suggested_amount"`
	ExactMatch      bool                       `json:"exact_match"`
}

type ReconcileResultResponse struct {
	StatementLineID     int64                   `json:"statement_line_id"`
	CandidateLineID     int64                   `json:"candidate_line_id"`
	Amount              float64                 `json:"amount"`
	StatementReconciled bool                    `json:"statement_reconciled"`
	CandidateReconciled bool                    `json:"candidate_reconciled"`
	MatchingNumber      *string                 `json:"matching_number,omitempty"`
	PaymentState        accounting.PaymentState `json:"payment_state"`
}

func ToReconcileResultResponse(r *bankstatementusecase.ReconcileResult) ReconcileResultResponse {
	return ReconcileResultResponse{
		StatementLineID:     r.StatementLineID,
		CandidateLineID:     r.CandidateLineID,
		Amount:              r.Amount,
		StatementReconciled: r.StatementReconciled,
		CandidateReconciled: r.CandidateReconciled,
		MatchingNumber:      r.MatchingNumber,
		PaymentState:        r.PaymentState,
	}
}

type ReconcileModelResponse struct {
	ID              int64                               `json:"id"`
	Name            string                              `json:"name"`
	Sequence        int                                 `json:"sequence"`
	IsAutoReconcile bool                                `json:"is_auto_reconcile"`
	MatchNature     bankstatement.ReconcileMatchNature  `json:"match_nature"`
	MatchAmount     *bankstatement.ReconcileMatchAmount `json:"match_amount"`
	MatchAmountMin  *float64                            `json:"match_amount_min"`
	MatchAmountMax  *float64                            `json:"match_amount_max"`
	MatchLabel      *bankstatement.ReconcileMatchLabel  `json:"match_label"`
	MatchLabelParam string                              `json:"match_label_param"`
	MatchJournalIDs []int64                             `json:"match_journal_ids"`
	MatchPartnerIDs []int64                             `json:"match_partner_ids"`
	MappedPartnerID *int64                              `json:"mapped_partner_id"`
	Lines           []ReconcileModelLineResponse        `json:"lines"`
}

type ReconcileModelLineResponse struct {
	ID               int64                                 `json:"id"`
	ReconcileModelID int64                                 `json:"reconcile_model_id"`
	AmountType       bankstatement.ReconcileLineAmountType `json:"amount_type"`
	Amount           string                                `json:"amount"`
	AccountID        int64                                 `json:"account_id"`
	Label            string                                `json:"label"`
	TaxIDs           []int64                               `json:"tax_ids"`
}

func ToReconcileModelResponse(m *bankstatement.ReconcileModel) ReconcileModelResponse {
	lines := make([]ReconcileModelLineResponse, 0, len(m.Lines))
	for _, l := range m.Lines {
		lines = append(lines, ReconcileModelLineResponse{
			ID:               l.ID,
			ReconcileModelID: l.ReconcileModelID,
			AmountType:       l.AmountType,
			Amount:           l.Amount,
			AccountID:        l.AccountID,
			Label:            l.Label,
			TaxIDs:           l.TaxIDs,
		})
	}
	return ReconcileModelResponse{
		ID:              m.ID,
		Name:            m.Name,
		Sequence:        m.Sequence,
		IsAutoReconcile: m.IsAutoReconcile,
		MatchNature:     m.MatchNature,
		MatchAmount:     m.MatchAmount,
		MatchAmountMin:  m.MatchAmountMin,
		MatchAmountMax:  m.MatchAmountMax,
		MatchLabel:      m.MatchLabel,
		MatchLabelParam: m.MatchLabelParam,
		MatchJournalIDs: m.MatchJournalIDs,
		MatchPartnerIDs: m.MatchPartnerIDs,
		MappedPartnerID: m.MappedPartnerID,
		Lines:           lines,
	}
}

type CashRoundingResponse struct {
	ID              int64                          `json:"id"`
	Name            string                         `json:"name"`
	RoundingMethod  bankstatement.RoundingMethod   `json:"rounding_method"`
	Rounding        float64                        `json:"rounding"`
	Strategy        bankstatement.RoundingStrategy `json:"strategy"`
	ProfitAccountID *int64                         `json:"profit_account_id"`
	LossAccountID   *int64                         `json:"loss_account_id"`
	Active          bool                           `json:"active"`
}

func ToCashRoundingResponse(cr *bankstatement.CashRounding) CashRoundingResponse {
	return CashRoundingResponse{
		ID:              cr.ID,
		Name:            cr.Name,
		RoundingMethod:  cr.RoundingMethod,
		Rounding:        cr.Rounding,
		Strategy:        cr.Strategy,
		ProfitAccountID: cr.ProfitAccountID,
		LossAccountID:   cr.LossAccountID,
		Active:          cr.Active,
	}
}
