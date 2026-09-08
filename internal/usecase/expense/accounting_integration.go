package expenseusecase

import (
	"context"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/expense"
	"cashflow_backend/internal/domain/hr"
	platformerrors "cashflow_backend/internal/platform/errors"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// PostExpenseInput contains the accounting configuration required to post an expense.
type PostExpenseInput struct {
	JournalID               int64
	CompanyPaymentAccountID *int64
}

// AccountingIntegration creates and posts the Odoo-style receipt move.
type AccountingIntegration struct {
	accounting *accountingusecase.UseCase
	hrRepo     hr.Repository
}

func NewAccountingIntegration(accountingUC *accountingusecase.UseCase, hrRepo hr.Repository) *AccountingIntegration {
	return &AccountingIntegration{accounting: accountingUC, hrRepo: hrRepo}
}

func (i *AccountingIntegration) PostExpense(ctx context.Context, value *expense.Expense, input PostExpenseInput) (*accounting.AccountMove, error) {
	if value == nil {
		return nil, platformerrors.Validation("expense is required", nil)
	}
	if value.State != expense.StateApproved {
		return nil, platformerrors.Conflict("only approved expenses can be posted")
	}
	if value.AccountMoveID != nil {
		return i.accounting.GetMove(ctx, *value.AccountMoveID)
	}
	if input.JournalID <= 0 {
		return nil, platformerrors.Validation("expense journal is required", map[string]string{"journal_id": "must be positive"})
	}
	if value.TotalAmount <= 0 {
		return nil, platformerrors.Validation("expense total must be positive", nil)
	}
	employee, err := i.hrRepo.GetEmployeeByID(ctx, value.EmployeeID)
	if err != nil {
		return nil, err
	}
	journal, err := i.accounting.GetJournal(ctx, input.JournalID)
	if err != nil {
		return nil, err
	}
	expenseAccountID := value.AccountID
	if expenseAccountID == nil || *expenseAccountID <= 0 {
		expenseAccountID = journal.DefaultAccountID
	}
	if expenseAccountID == nil || *expenseAccountID <= 0 {
		return nil, platformerrors.Validation("expense account is required", nil)
	}

	untaxed, taxTotal, taxLines, err := i.computeTaxes(ctx, value)
	if err != nil {
		return nil, err
	}
	value.UntaxedAmount = untaxed
	value.TaxAmount = taxTotal
	value.TotalAmount = untaxed + taxTotal
	if value.TotalAmount <= 0 {
		return nil, platformerrors.Validation("expense total must be positive", nil)
	}

	counterpart := input.CompanyPaymentAccountID
	var partnerID *int64
	if value.PaymentMode == expense.PaymentOwnAccount {
		payable, lookupErr := i.accounting.GetAccountByCode(ctx, "210000")
		if lookupErr != nil {
			return nil, lookupErr
		}
		counterpart = &payable.ID
		partnerID = employee.PartnerID
	} else if counterpart == nil || *counterpart <= 0 {
		return nil, platformerrors.Validation("company payment account is required", nil)
	}

	lines := []accountingusecase.JournalEntryLineInput{
		{AccountID: *expenseAccountID, PartnerID: partnerID, ProductID: value.ProductID, Name: strings.TrimSpace(value.Name), Debit: untaxed},
	}
	lines = append(lines, taxLines...)
	lines = append(lines, accountingusecase.JournalEntryLineInput{
		AccountID: *counterpart, PartnerID: partnerID, Name: strings.TrimSpace(value.Name), Credit: value.TotalAmount,
	})
	move, err := i.accounting.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
		JournalID: input.JournalID, MoveType: accounting.MoveTypeInReceipt, Date: value.Date, Ref: value.Name, Lines: lines,
	})
	if err != nil {
		return nil, err
	}
	posted, err := i.accounting.PostMove(ctx, move.ID)
	if err != nil {
		return nil, err
	}
	return posted, nil
}

func (i *AccountingIntegration) computeTaxes(ctx context.Context, value *expense.Expense) (float64, float64, []accountingusecase.JournalEntryLineInput, error) {
	base := value.TotalAmount
	if len(value.TaxIDs) > 0 && value.UnitAmount > 0 {
		quantity := value.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		base = value.UnitAmount * quantity
	}
	untaxed := base
	taxTotal := 0.0
	lines := make([]accountingusecase.JournalEntryLineInput, 0, len(value.TaxIDs))
	for _, taxID := range value.TaxIDs {
		tax, err := i.accounting.GetTax(ctx, taxID)
		if err != nil {
			return 0, 0, nil, err
		}
		result := tax.Compute(base)
		if tax.PriceInclude {
			untaxed = result.UntaxedAmount
			base = result.UntaxedAmount
		} else {
			untaxed = base
			base = result.TotalAmount
		}
		taxTotal += result.TaxAmount
		lines = append(lines, accountingusecase.JournalEntryLineInput{AccountID: tax.AccountID, Name: string(tax.Name), Debit: result.TaxAmount})
	}
	if len(value.TaxIDs) == 0 {
		untaxed = value.TotalAmount
	}
	if taxTotal == 0 && len(value.TaxIDs) > 0 {
		return 0, 0, nil, platformerrors.Validation(fmt.Sprintf("taxes on expense %d produced no tax amount", value.ID), nil)
	}
	return untaxed, taxTotal, lines, nil
}
