package accountingusecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"cashflow_backend/internal/domain/accounting"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateAccountInput struct {
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	Type      accounting.AccountType `json:"type"`
	Reconcile bool                   `json:"reconcile"`
	Currency  string                 `json:"currency"`
	ParentID  *int64                 `json:"parent_id"`
	CompanyID *int64                 `json:"company_id"`
}

type UpdateAccountInput struct {
	Code      *string                 `json:"code"`
	Name      *string                 `json:"name"`
	Type      *accounting.AccountType `json:"type"`
	Reconcile *bool                   `json:"reconcile"`
	Currency  *string                 `json:"currency"`
	ParentID  *int64                  `json:"parent_id"`
	CompanyID *int64                  `json:"company_id"`
}

type CreateJournalInput struct {
	Name              string                 `json:"name"`
	Code              string                 `json:"code"`
	Type              accounting.JournalType `json:"type"`
	DefaultAccountID  *int64                 `json:"default_account_id"`
	SuspenseAccountID *int64                 `json:"suspense_account_id"`
	SequencePrefix    string                 `json:"sequence_prefix"`
	NextNumber        int                    `json:"next_number"`
}

type UpdateJournalInput struct {
	Name              *string                 `json:"name"`
	Code              *string                 `json:"code"`
	Type              *accounting.JournalType `json:"type"`
	DefaultAccountID  *int64                  `json:"default_account_id"`
	SuspenseAccountID *int64                  `json:"suspense_account_id"`
	SequencePrefix    *string                 `json:"sequence_prefix"`
}

type CreateTaxInput struct {
	Name            string              `json:"name"`
	Type            accounting.TaxType  `json:"type"`
	TypeTaxUse      accounting.TaxScope `json:"type_tax_use"`
	Amount          float64             `json:"amount"`
	AccountID       int64               `json:"account_id"`
	RefundAccountID *int64              `json:"refund_account_id"`
	PriceInclude    bool                `json:"price_include"`
}

type UpdateTaxInput struct {
	Name            *string              `json:"name"`
	Type            *accounting.TaxType  `json:"type"`
	TypeTaxUse      *accounting.TaxScope `json:"type_tax_use"`
	Amount          *float64             `json:"amount"`
	AccountID       *int64               `json:"account_id"`
	RefundAccountID *int64               `json:"refund_account_id"`
	PriceInclude    *bool                `json:"price_include"`
}

type CreatePaymentTermLineInput struct {
	ValueType   accounting.PaymentTermValueType `json:"value_type"`
	ValueAmount float64                         `json:"value_amount"`
	Days        int                             `json:"days"`
	DayOfMonth  int                             `json:"day_of_month"`
}

type CreatePaymentTermInput struct {
	Name  string                       `json:"name"`
	Note  string                       `json:"note"`
	Lines []CreatePaymentTermLineInput `json:"lines"`
}

type UpdatePaymentTermInput struct {
	Name  *string                      `json:"name"`
	Note  *string                      `json:"note"`
	Lines []CreatePaymentTermLineInput `json:"lines"`
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Definition
// ─────────────────────────────────────────────────────────────────────────────

// UseCase orchestrates business rules and application logic for the Core Accounting domain.
type UseCase struct {
	repo   accounting.Repository
	logger *slog.Logger
}

// New creates an initialized accounting UseCase instance.
func New(repo accounting.Repository, logger *slog.Logger) *UseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &UseCase{
		repo:   repo,
		logger: logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Accounts Management
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateAccount(ctx context.Context, in CreateAccountInput) (*accounting.Account, error) {
	acc := &accounting.Account{
		Code:      strings.TrimSpace(in.Code),
		Name:      strings.TrimSpace(in.Name),
		Type:      in.Type,
		Reconcile: in.Reconcile,
		Currency:  in.Currency,
		ParentID:  in.ParentID,
		CompanyID: in.CompanyID,
	}

	if err := acc.Validate(); err != nil {
		return nil, err
	}

	if in.ParentID != nil && *in.ParentID > 0 {
		if _, err := uc.repo.GetAccountByID(ctx, *in.ParentID); err != nil {
			return nil, platformerrors.Validation("parent account does not exist", map[string]string{
				"parent_id": fmt.Sprintf("account %d not found", *in.ParentID),
			})
		}
	}

	if err := uc.repo.CreateAccount(ctx, acc); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "account created", "id", acc.ID, "code", acc.Code, "name", acc.Name)
	return acc, nil
}

func (uc *UseCase) GetAccount(ctx context.Context, id int64) (*accounting.Account, error) {
	return uc.repo.GetAccountByID(ctx, id)
}

func (uc *UseCase) GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error) {
	return uc.repo.GetAccountByCode(ctx, code)
}

func (uc *UseCase) UpdateAccount(ctx context.Context, id int64, in UpdateAccountInput) (*accounting.Account, error) {
	acc, err := uc.repo.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Code != nil {
		acc.Code = strings.TrimSpace(*in.Code)
	}
	if in.Name != nil {
		acc.Name = strings.TrimSpace(*in.Name)
	}
	if in.Type != nil {
		acc.Type = *in.Type
	}
	if in.Reconcile != nil {
		acc.Reconcile = *in.Reconcile
	}
	if in.Currency != nil {
		acc.Currency = *in.Currency
	}
	if in.ParentID != nil {
		if *in.ParentID > 0 {
			if _, err := uc.repo.GetAccountByID(ctx, *in.ParentID); err != nil {
				return nil, platformerrors.Validation("parent account does not exist", map[string]string{
					"parent_id": fmt.Sprintf("account %d not found", *in.ParentID),
				})
			}
			acc.ParentID = in.ParentID
		} else {
			acc.ParentID = nil
		}
	}
	if in.CompanyID != nil {
		acc.CompanyID = in.CompanyID
	}

	if err := acc.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateAccount(ctx, acc); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "account updated", "id", acc.ID, "code", acc.Code)
	return acc, nil
}

func (uc *UseCase) DeleteAccount(ctx context.Context, id int64) error {
	if err := uc.repo.DeleteAccount(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "account deleted", "id", id)
	return nil
}

func (uc *UseCase) ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.Account], error) {
	return uc.repo.ListAccounts(ctx, f, page)
}

// ─────────────────────────────────────────────────────────────────────────────
// Journals Management
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateJournal(ctx context.Context, in CreateJournalInput) (*accounting.Journal, error) {
	j := &accounting.Journal{
		Name:              strings.TrimSpace(in.Name),
		Code:              strings.TrimSpace(strings.ToUpper(in.Code)),
		Type:              in.Type,
		DefaultAccountID:  in.DefaultAccountID,
		SuspenseAccountID: in.SuspenseAccountID,
		SequencePrefix:    in.SequencePrefix,
		NextNumber:        in.NextNumber,
	}

	if err := j.Validate(); err != nil {
		return nil, err
	}

	if in.DefaultAccountID != nil && *in.DefaultAccountID > 0 {
		if _, err := uc.repo.GetAccountByID(ctx, *in.DefaultAccountID); err != nil {
			return nil, platformerrors.Validation("default account does not exist", map[string]string{
				"default_account_id": fmt.Sprintf("account %d not found", *in.DefaultAccountID),
			})
		}
	}

	if in.SuspenseAccountID != nil && *in.SuspenseAccountID > 0 {
		if _, err := uc.repo.GetAccountByID(ctx, *in.SuspenseAccountID); err != nil {
			return nil, platformerrors.Validation("suspense account does not exist", map[string]string{
				"suspense_account_id": fmt.Sprintf("account %d not found", *in.SuspenseAccountID),
			})
		}
	}

	if err := uc.repo.CreateJournal(ctx, j); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "journal created", "id", j.ID, "code", j.Code, "name", j.Name)
	return j, nil
}

func (uc *UseCase) GetJournal(ctx context.Context, id int64) (*accounting.Journal, error) {
	return uc.repo.GetJournalByID(ctx, id)
}

func (uc *UseCase) UpdateJournal(ctx context.Context, id int64, in UpdateJournalInput) (*accounting.Journal, error) {
	j, err := uc.repo.GetJournalByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		j.Name = strings.TrimSpace(*in.Name)
	}
	if in.Code != nil {
		j.Code = strings.TrimSpace(strings.ToUpper(*in.Code))
	}
	if in.Type != nil {
		j.Type = *in.Type
	}
	if in.DefaultAccountID != nil {
		if *in.DefaultAccountID > 0 {
			if _, err := uc.repo.GetAccountByID(ctx, *in.DefaultAccountID); err != nil {
				return nil, platformerrors.Validation("default account does not exist", map[string]string{
					"default_account_id": fmt.Sprintf("account %d not found", *in.DefaultAccountID),
				})
			}
			j.DefaultAccountID = in.DefaultAccountID
		} else {
			j.DefaultAccountID = nil
		}
	}
	if in.SuspenseAccountID != nil {
		if *in.SuspenseAccountID > 0 {
			if _, err := uc.repo.GetAccountByID(ctx, *in.SuspenseAccountID); err != nil {
				return nil, platformerrors.Validation("suspense account does not exist", map[string]string{
					"suspense_account_id": fmt.Sprintf("account %d not found", *in.SuspenseAccountID),
				})
			}
			j.SuspenseAccountID = in.SuspenseAccountID
		} else {
			j.SuspenseAccountID = nil
		}
	}
	if in.SequencePrefix != nil {
		j.SequencePrefix = *in.SequencePrefix
	}

	if err := j.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateJournal(ctx, j); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "journal updated", "id", j.ID, "code", j.Code)
	return j, nil
}

func (uc *UseCase) DeleteJournal(ctx context.Context, id int64) error {
	if err := uc.repo.DeleteJournal(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "journal deleted", "id", id)
	return nil
}

func (uc *UseCase) ListJournals(ctx context.Context) ([]accounting.Journal, error) {
	return uc.repo.ListJournals(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Taxes Management
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateTax(ctx context.Context, in CreateTaxInput) (*accounting.Tax, error) {
	t := &accounting.Tax{
		Name:            strings.TrimSpace(in.Name),
		Type:            in.Type,
		TypeTaxUse:      in.TypeTaxUse,
		Amount:          in.Amount,
		AccountID:       in.AccountID,
		RefundAccountID: in.RefundAccountID,
		PriceInclude:    in.PriceInclude,
	}

	if err := t.Validate(); err != nil {
		return nil, err
	}

	if _, err := uc.repo.GetAccountByID(ctx, in.AccountID); err != nil {
		return nil, platformerrors.Validation("tax account does not exist", map[string]string{
			"account_id": fmt.Sprintf("account %d not found", in.AccountID),
		})
	}

	if in.RefundAccountID != nil && *in.RefundAccountID > 0 {
		if _, err := uc.repo.GetAccountByID(ctx, *in.RefundAccountID); err != nil {
			return nil, platformerrors.Validation("tax refund account does not exist", map[string]string{
				"refund_account_id": fmt.Sprintf("account %d not found", *in.RefundAccountID),
			})
		}
	}

	if err := uc.repo.CreateTax(ctx, t); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "tax created", "id", t.ID, "name", t.Name, "amount", t.Amount)
	return t, nil
}

func (uc *UseCase) GetTax(ctx context.Context, id int64) (*accounting.Tax, error) {
	return uc.repo.GetTaxByID(ctx, id)
}

func (uc *UseCase) UpdateTax(ctx context.Context, id int64, in UpdateTaxInput) (*accounting.Tax, error) {
	t, err := uc.repo.GetTaxByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		t.Name = strings.TrimSpace(*in.Name)
	}
	if in.Type != nil {
		t.Type = *in.Type
	}
	if in.TypeTaxUse != nil {
		t.TypeTaxUse = *in.TypeTaxUse
	}
	if in.Amount != nil {
		t.Amount = *in.Amount
	}
	if in.AccountID != nil {
		if _, err := uc.repo.GetAccountByID(ctx, *in.AccountID); err != nil {
			return nil, platformerrors.Validation("tax account does not exist", map[string]string{
				"account_id": fmt.Sprintf("account %d not found", *in.AccountID),
			})
		}
		t.AccountID = *in.AccountID
	}
	if in.RefundAccountID != nil {
		if *in.RefundAccountID > 0 {
			if _, err := uc.repo.GetAccountByID(ctx, *in.RefundAccountID); err != nil {
				return nil, platformerrors.Validation("tax refund account does not exist", map[string]string{
					"refund_account_id": fmt.Sprintf("account %d not found", *in.RefundAccountID),
				})
			}
			t.RefundAccountID = in.RefundAccountID
		} else {
			t.RefundAccountID = nil
		}
	}
	if in.PriceInclude != nil {
		t.PriceInclude = *in.PriceInclude
	}

	if err := t.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateTax(ctx, t); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "tax updated", "id", t.ID, "name", t.Name)
	return t, nil
}

func (uc *UseCase) DeleteTax(ctx context.Context, id int64) error {
	if err := uc.repo.DeleteTax(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "tax deleted", "id", id)
	return nil
}

func (uc *UseCase) ListTaxes(ctx context.Context, scope *accounting.TaxScope) ([]accounting.Tax, error) {
	return uc.repo.ListTaxes(ctx, scope)
}

func (uc *UseCase) ComputeTax(ctx context.Context, taxID int64, amount float64) (*accounting.TaxCalculationResult, error) {
	t, err := uc.repo.GetTaxByID(ctx, taxID)
	if err != nil {
		return nil, err
	}
	res := t.Compute(amount)
	return &res, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Payment Terms Management
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreatePaymentTerm(ctx context.Context, in CreatePaymentTermInput) (*accounting.PaymentTerm, error) {
	pt := &accounting.PaymentTerm{
		Name:  strings.TrimSpace(in.Name),
		Note:  in.Note,
		Lines: make([]accounting.PaymentTermLine, len(in.Lines)),
	}

	for i, l := range in.Lines {
		pt.Lines[i] = accounting.PaymentTermLine{
			ValueType:   l.ValueType,
			ValueAmount: l.ValueAmount,
			Days:        l.Days,
			DayOfMonth:  l.DayOfMonth,
		}
	}

	if err := pt.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreatePaymentTerm(ctx, pt); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "payment term created", "id", pt.ID, "name", pt.Name)
	return pt, nil
}

func (uc *UseCase) GetPaymentTerm(ctx context.Context, id int64) (*accounting.PaymentTerm, error) {
	return uc.repo.GetPaymentTermByID(ctx, id)
}

func (uc *UseCase) UpdatePaymentTerm(ctx context.Context, id int64, in UpdatePaymentTermInput) (*accounting.PaymentTerm, error) {
	pt, err := uc.repo.GetPaymentTermByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		pt.Name = strings.TrimSpace(*in.Name)
	}
	if in.Note != nil {
		pt.Note = *in.Note
	}
	if in.Lines != nil {
		pt.Lines = make([]accounting.PaymentTermLine, len(in.Lines))
		for i, l := range in.Lines {
			pt.Lines[i] = accounting.PaymentTermLine{
				PaymentTermID: pt.ID,
				ValueType:     l.ValueType,
				ValueAmount:   l.ValueAmount,
				Days:          l.Days,
				DayOfMonth:    l.DayOfMonth,
			}
		}
	}

	if err := pt.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdatePaymentTerm(ctx, pt); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "payment term updated", "id", pt.ID, "name", pt.Name)
	return pt, nil
}

func (uc *UseCase) DeletePaymentTerm(ctx context.Context, id int64) error {
	if err := uc.repo.DeletePaymentTerm(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "payment term deleted", "id", id)
	return nil
}

func (uc *UseCase) ListPaymentTerms(ctx context.Context) ([]accounting.PaymentTerm, error) {
	return uc.repo.ListPaymentTerms(ctx)
}
