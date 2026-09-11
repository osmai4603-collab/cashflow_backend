package paymentusecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/payment"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// AccountingService abstracts the accounting operations required by the payment module.
type AccountingService interface {
	CreateJournalEntry(ctx context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error)
	PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
	CancelMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
	GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
	GetJournal(ctx context.Context, id int64) (*accounting.Journal, error)
	GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error)
	ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.AccountMove], error)
	UpdatePaymentStatus(ctx context.Context, id int64, state accounting.PaymentState, residual float64) error
}

// PartnerRepository abstracts partner retrieval.
type PartnerRepository interface {
	GetByID(ctx context.Context, id int64) (*partner.Partner, error)
}

type ProviderRegistry interface {
	Get(string) (payment.PaymentProviderInterface, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreatePaymentInput struct {
	PartnerID     int64                 `json:"partner_id"`
	Amount        float64               `json:"amount"`
	PaymentType   payment.PaymentType   `json:"payment_type"` // inbound, outbound
	PartnerType   payment.PartnerType   `json:"partner_type"` // customer, supplier
	PaymentMethod payment.PaymentMethod `json:"payment_method"`
	JournalID     int64                 `json:"journal_id"`
	Date          time.Time             `json:"date"`
	Ref           string                `json:"ref"`
	Currency      string                `json:"currency"`
	CompanyID     *int64                `json:"company_id"`
	InvoiceIDs    []int64               `json:"invoice_ids"`
	AutoReconcile bool                  `json:"auto_reconcile"`
}

type UpdatePaymentInput struct {
	PartnerID     *int64                 `json:"partner_id"`
	Amount        *float64               `json:"amount"`
	PaymentMethod *payment.PaymentMethod `json:"payment_method"`
	JournalID     *int64                 `json:"journal_id"`
	Date          *time.Time             `json:"date"`
	Ref           *string                `json:"ref"`
	InvoiceIDs    []int64                `json:"invoice_ids"`
}

type ReconcileInput struct {
	InvoiceIDs []int64 `json:"invoice_ids"` // If empty, auto-matches open invoices in FIFO order
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Definition
// ─────────────────────────────────────────────────────────────────────────────

type UseCase struct {
	repo              payment.Repository
	accountingService AccountingService
	partnerRepo       PartnerRepository
	logger            *slog.Logger
	providers         ProviderRegistry
}

// New constructs a new Payment UseCase.
func New(
	repo payment.Repository,
	accountingService AccountingService,
	partnerRepo PartnerRepository,
	logger *slog.Logger,
	providers ...ProviderRegistry,
) *UseCase {
	var registry ProviderRegistry
	if len(providers) > 0 {
		registry = providers[0]
	}
	return &UseCase{
		repo:              repo,
		accountingService: accountingService,
		partnerRepo:       partnerRepo,
		logger:            logger,
		providers:         registry,
	}
}

// CreatePayment initializes a new draft payment.
func (uc *UseCase) CreatePayment(ctx context.Context, in CreatePaymentInput) (*payment.Payment, error) {
	if in.PartnerID <= 0 {
		return nil, platformerrors.Validation("partner is required", map[string]string{
			"partner_id": "must reference a valid partner",
		})
	}
	if in.JournalID <= 0 {
		return nil, platformerrors.Validation("journal is required", map[string]string{
			"journal_id": "must reference a valid journal",
		})
	}

	// Validate partner exists
	if uc.partnerRepo != nil {
		if _, err := uc.partnerRepo.GetByID(ctx, in.PartnerID); err != nil {
			return nil, platformerrors.Validation("invalid partner", map[string]string{
				"partner_id": fmt.Sprintf("partner with id %d does not exist", in.PartnerID),
			})
		}
	}

	// Validate journal exists
	if uc.accountingService != nil {
		j, err := uc.accountingService.GetJournal(ctx, in.JournalID)
		if err != nil {
			return nil, platformerrors.Validation("invalid journal", map[string]string{
				"journal_id": fmt.Sprintf("journal with id %d does not exist", in.JournalID),
			})
		}
		if j.Type != accounting.JournalTypeCash && j.Type != accounting.JournalTypeBank {
			return nil, platformerrors.Validation("invalid journal type", map[string]string{
				"journal_id": "payment journal must be of type 'cash' or 'bank'",
			})
		}
	}

	if in.Date.IsZero() {
		in.Date = time.Now().UTC()
	}
	if in.Currency == "" {
		in.Currency = "USD"
	}

	p := &payment.Payment{
		Name:             "/",
		PaymentType:      in.PaymentType,
		PartnerType:      in.PartnerType,
		PartnerID:        in.PartnerID,
		Amount:           in.Amount,
		Currency:         in.Currency,
		PaymentMethod:    in.PaymentMethod,
		JournalID:        in.JournalID,
		Date:             in.Date,
		State:            payment.PaymentStateDraft,
		Ref:              strings.TrimSpace(in.Ref),
		InvoiceIDs:       in.InvoiceIDs,
		ResidualAmount:   in.Amount,
		ReconciledAmount: 0,
		CompanyID:        in.CompanyID,
		Active:           true,
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "payment created", "id", p.ID, "type", p.PaymentType, "amount", p.Amount)

	// Proactive auto-posting if requested
	if in.AutoReconcile {
		return uc.PostPayment(ctx, p.ID, true)
	}

	return p, nil
}

// GetPayment fetches a payment with its associated reconciliations.
func (uc *UseCase) GetPayment(ctx context.Context, id int64) (*payment.Payment, error) {
	p, err := uc.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	recons, err := uc.repo.GetReconciliationsByPaymentID(ctx, id)
	if err == nil && len(recons) > 0 {
		invIDs := make([]int64, 0, len(recons))
		for _, r := range recons {
			invIDs = append(invIDs, r.InvoiceID)
		}
		p.InvoiceIDs = invIDs
	}

	return p, nil
}

// UpdatePayment edits fields on an existing draft payment.
func (uc *UseCase) UpdatePayment(ctx context.Context, id int64, in UpdatePaymentInput) (*payment.Payment, error) {
	p, err := uc.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.State != payment.PaymentStateDraft {
		return nil, platformerrors.Conflict("only draft payments can be modified")
	}

	if in.PartnerID != nil && *in.PartnerID > 0 {
		p.PartnerID = *in.PartnerID
	}
	if in.Amount != nil && *in.Amount > 0 {
		p.Amount = *in.Amount
		p.ResidualAmount = *in.Amount
	}
	if in.PaymentMethod != nil {
		p.PaymentMethod = *in.PaymentMethod
	}
	if in.JournalID != nil && *in.JournalID > 0 {
		p.JournalID = *in.JournalID
	}
	if in.Date != nil && !in.Date.IsZero() {
		p.Date = *in.Date
	}
	if in.Ref != nil {
		p.Ref = strings.TrimSpace(*in.Ref)
	}
	if in.InvoiceIDs != nil {
		p.InvoiceIDs = in.InvoiceIDs
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdatePayment(ctx, p); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "payment updated", "id", p.ID)
	return p, nil
}

// DeletePayment removes a draft payment.
func (uc *UseCase) DeletePayment(ctx context.Context, id int64) error {
	p, err := uc.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return err
	}

	if p.State != payment.PaymentStateDraft {
		return platformerrors.Conflict("cannot delete a posted or cancelled payment; cancel it instead")
	}

	if err := uc.repo.DeletePayment(ctx, id); err != nil {
		return err
	}

	uc.logger.InfoContext(ctx, "payment deleted", "id", id)
	return nil
}

// ListPayments returns a paginated list of payments according to filters.
func (uc *UseCase) ListPayments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[payment.Payment], error) {
	return uc.repo.ListPayments(ctx, f, page)
}

// PostPayment confirms a payment, creates a balanced journal entry in accounting, and optionally reconciles.
func (uc *UseCase) PostPayment(ctx context.Context, id int64, autoReconcile bool) (*payment.Payment, error) {
	p, err := uc.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.State == payment.PaymentStatePosted || p.State == payment.PaymentStateReconciled {
		return p, nil
	}
	if p.State == payment.PaymentStateCancelled {
		return nil, platformerrors.Conflict("cannot post a cancelled payment")
	}

	if uc.accountingService == nil {
		return nil, platformerrors.Internal("accounting service is not available", nil)
	}

	// 1. Resolve Journal & Accounts
	journal, err := uc.accountingService.GetJournal(ctx, p.JournalID)
	if err != nil {
		return nil, err
	}

	var liquidityAccID int64
	if journal.DefaultAccountID != nil && *journal.DefaultAccountID > 0 {
		liquidityAccID = *journal.DefaultAccountID
	} else {
		targetCode := "101000" // Cash on Hand
		if journal.Type == accounting.JournalTypeBank {
			targetCode = "102000" // Bank Account
		}
		acc, err := uc.accountingService.GetAccountByCode(ctx, targetCode)
		if err != nil {
			if journal.Type == accounting.JournalTypeBank {
				liquidityAccID = 2
			} else {
				liquidityAccID = 1
			}
		} else {
			liquidityAccID = acc.ID
		}
	}

	var counterpartAccID int64
	if p.PaymentType == payment.PaymentTypeInbound {
		// Accounts Receivable
		arAcc, err := uc.accountingService.GetAccountByCode(ctx, "120000")
		if err != nil {
			counterpartAccID = 3
		} else {
			counterpartAccID = arAcc.ID
		}
	} else {
		// Accounts Payable
		apAcc, err := uc.accountingService.GetAccountByCode(ctx, "210000")
		if err != nil {
			counterpartAccID = 6
		} else {
			counterpartAccID = apAcc.ID
		}
	}

	// 2. Build double-entry lines
	var lines []accountingusecase.JournalEntryLineInput
	paymentLabel := p.Ref
	if paymentLabel == "" {
		paymentLabel = fmt.Sprintf("Payment for Partner #%d", p.PartnerID)
	}

	if p.PaymentType == payment.PaymentTypeInbound {
		// Inbound (Customer collection):
		// Debit: Liquidity (Bank / Cash)
		// Credit: Accounts Receivable
		lines = []accountingusecase.JournalEntryLineInput{
			{
				AccountID: liquidityAccID,
				PartnerID: &p.PartnerID,
				Name:      paymentLabel,
				Debit:     p.Amount,
				Credit:    0,
			},
			{
				AccountID: counterpartAccID,
				PartnerID: &p.PartnerID,
				Name:      paymentLabel,
				Debit:     0,
				Credit:    p.Amount,
			},
		}
	} else {
		// Outbound (Vendor payment):
		// Debit: Accounts Payable
		// Credit: Liquidity (Bank / Cash)
		lines = []accountingusecase.JournalEntryLineInput{
			{
				AccountID: counterpartAccID,
				PartnerID: &p.PartnerID,
				Name:      paymentLabel,
				Debit:     p.Amount,
				Credit:    0,
			},
			{
				AccountID: liquidityAccID,
				PartnerID: &p.PartnerID,
				Name:      paymentLabel,
				Debit:     0,
				Credit:    p.Amount,
			},
		}
	}

	// 3. Create & Post Accounting Move
	entry, err := uc.accountingService.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
		JournalID: p.JournalID,
		Date:      p.Date,
		Ref:       paymentLabel,
		Lines:     lines,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment journal entry: %w", err)
	}

	postedEntry, err := uc.accountingService.PostMove(ctx, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to post payment journal entry: %w", err)
	}

	// 4. Generate Sequence and Update Payment
	seq, err := uc.repo.NextSequence(ctx, p.Date.Year())
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment sequence: %w", err)
	}

	if err := p.Post(seq); err != nil {
		return nil, err
	}
	p.MoveID = &postedEntry.ID

	if err := uc.repo.UpdatePayment(ctx, p); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "payment posted", "id", p.ID, "sequence", p.Name, "move_id", postedEntry.ID)

	// 5. Automatic or targeted reconciliation
	if autoReconcile || len(p.InvoiceIDs) > 0 {
		return uc.ReconcilePayment(ctx, p.ID, ReconcileInput{InvoiceIDs: p.InvoiceIDs})
	}

	return p, nil
}

// ReconcilePayment matches a posted payment with open invoices and deducts residual amounts.
func (uc *UseCase) ReconcilePayment(ctx context.Context, id int64, in ReconcileInput) (*payment.Payment, error) {
	p, err := uc.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.State != payment.PaymentStatePosted && p.State != payment.PaymentStateReconciled {
		return nil, platformerrors.Conflict("only posted payments can be reconciled")
	}

	if p.ResidualAmount <= 0.0001 {
		return p, nil // already fully settled
	}

	var candidateInvoices []*accounting.AccountMove

	if len(in.InvoiceIDs) > 0 {
		for _, invID := range in.InvoiceIDs {
			inv, err := uc.accountingService.GetMove(ctx, invID)
			if err != nil {
				return nil, err
			}
			candidateInvoices = append(candidateInvoices, inv)
		}
	} else {
		// Auto-discover open invoices for this partner
		f := filter.NewFilter()
		f.Add("partner_id", filter.OpEqual, p.PartnerID)
		f.Add("state", filter.OpEqual, string(accounting.MoveStatePosted))

		res, err := uc.accountingService.ListMoves(ctx, f, pagination.PageRequest{Page: 1, Limit: 100})
		if err != nil {
			return nil, err
		}

		for i := range res.Items {
			inv := &res.Items[i]
			if inv.AmountResidual > 0.0001 {
				candidateInvoices = append(candidateInvoices, inv)
			}
		}

		// Sort candidate invoices FIFO by invoice date / due date
		sort.Slice(candidateInvoices, func(i, j int) bool {
			d1 := candidateInvoices[i].Date
			if candidateInvoices[i].InvoiceDueDate != nil {
				d1 = *candidateInvoices[i].InvoiceDueDate
			}
			d2 := candidateInvoices[j].Date
			if candidateInvoices[j].InvoiceDueDate != nil {
				d2 = *candidateInvoices[j].InvoiceDueDate
			}
			return d1.Before(d2)
		})
	}

	linkedInvoicesMap := make(map[int64]bool)
	for _, invID := range p.InvoiceIDs {
		linkedInvoicesMap[invID] = true
	}

	for _, inv := range candidateInvoices {
		if p.ResidualAmount <= 0.0001 {
			break
		}

		// Verify partner matches
		if inv.PartnerID == nil || *inv.PartnerID != p.PartnerID {
			continue
		}

		// Verify invoice type matches payment flow
		if p.PaymentType == payment.PaymentTypeInbound {
			if inv.MoveType != accounting.MoveTypeOutInvoice && inv.MoveType != accounting.MoveTypeOutRefund {
				continue
			}
		} else {
			if inv.MoveType != accounting.MoveTypeInInvoice && inv.MoveType != accounting.MoveTypeInRefund {
				continue
			}
		}

		if inv.State != accounting.MoveStatePosted || inv.AmountResidual <= 0.0001 {
			continue
		}

		// Calculate matching allocation
		alloc := math.Min(p.ResidualAmount, inv.AmountResidual)
		alloc = roundTo4(alloc)
		if alloc <= 0.0001 {
			continue
		}

		// Record reconciliation
		recon := &payment.PaymentReconciliation{
			PaymentID:    p.ID,
			InvoiceID:    inv.ID,
			Amount:       alloc,
			ReconciledAt: time.Now().UTC(),
		}
		if err := uc.repo.CreateReconciliation(ctx, recon); err != nil {
			return nil, err
		}

		// Update invoice settlement status
		newInvResidual := roundTo4(inv.AmountResidual - alloc)
		var newPaymentState accounting.PaymentState
		if newInvResidual <= 0.0001 {
			newPaymentState = accounting.PaymentStatePaid
			newInvResidual = 0
		} else {
			newPaymentState = accounting.PaymentStatePartial
		}

		if err := uc.accountingService.UpdatePaymentStatus(ctx, inv.ID, newPaymentState, newInvResidual); err != nil {
			return nil, fmt.Errorf("failed to update invoice %d payment status: %w", inv.ID, err)
		}

		// Update payment balance
		if err := p.AllocateReconciliation(alloc); err != nil {
			return nil, err
		}

		linkedInvoicesMap[inv.ID] = true
	}

	// Update list of linked invoice IDs on payment
	p.InvoiceIDs = make([]int64, 0, len(linkedInvoicesMap))
	for invID := range linkedInvoicesMap {
		p.InvoiceIDs = append(p.InvoiceIDs, invID)
	}
	sort.Slice(p.InvoiceIDs, func(i, j int) bool { return p.InvoiceIDs[i] < p.InvoiceIDs[j] })

	if err := uc.repo.UpdatePayment(ctx, p); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "payment reconciled",
		"id", p.ID,
		"reconciled_amount", p.ReconciledAmount,
		"residual_amount", p.ResidualAmount,
		"state", p.State,
	)

	return p, nil
}

// CancelPayment revokes a payment, un-reconciles matched invoices, and cancels the journal entry.
func (uc *UseCase) CancelPayment(ctx context.Context, id int64) (*payment.Payment, error) {
	p, err := uc.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.State == payment.PaymentStateCancelled {
		return p, nil
	}

	// 1. Unlink & reverse reconciliations
	recons, err := uc.repo.GetReconciliationsByPaymentID(ctx, p.ID)
	if err == nil && len(recons) > 0 {
		for _, r := range recons {
			inv, err := uc.accountingService.GetMove(ctx, r.InvoiceID)
			if err == nil && inv != nil {
				restoredResidual := roundTo4(inv.AmountResidual + r.Amount)
				var restoredState accounting.PaymentState
				if restoredResidual >= inv.AmountTotal-0.0001 {
					restoredState = accounting.PaymentStateNotPaid
					restoredResidual = inv.AmountTotal
				} else {
					restoredState = accounting.PaymentStatePartial
				}
				_ = uc.accountingService.UpdatePaymentStatus(ctx, inv.ID, restoredState, restoredResidual)
			}
		}
		_ = uc.repo.DeleteReconciliationsByPaymentID(ctx, p.ID)
	}

	// 2. Cancel linked journal entry
	if p.MoveID != nil && *p.MoveID > 0 {
		if _, err := uc.accountingService.CancelMove(ctx, *p.MoveID); err != nil {
			uc.logger.WarnContext(ctx, "failed to cancel linked move during payment cancellation", "move_id", *p.MoveID, "error", err)
		}
	}

	// 3. Mark payment cancelled
	if err := p.Cancel(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdatePayment(ctx, p); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "payment cancelled", "id", p.ID)
	return p, nil
}

// GetReceivableAging compiles the Aged Receivable report across customer invoices.
func (uc *UseCase) GetReceivableAging(ctx context.Context, asOfDate time.Time, partnerID *int64) (*payment.AgingReport, error) {
	return uc.generateAgingReport(ctx, "receivable", accounting.MoveTypeOutInvoice, asOfDate, partnerID)
}

// GetPayableAging compiles the Aged Payable report across vendor bills.
func (uc *UseCase) GetPayableAging(ctx context.Context, asOfDate time.Time, partnerID *int64) (*payment.AgingReport, error) {
	return uc.generateAgingReport(ctx, "payable", accounting.MoveTypeInInvoice, asOfDate, partnerID)
}

func (uc *UseCase) generateAgingReport(
	ctx context.Context,
	reportType string,
	targetMoveType accounting.MoveType,
	asOfDate time.Time,
	partnerID *int64,
) (*payment.AgingReport, error) {
	if uc.accountingService == nil {
		return nil, platformerrors.Internal("accounting service is not available", nil)
	}

	if asOfDate.IsZero() {
		asOfDate = time.Now().UTC()
	}

	f := filter.NewFilter()
	f.Add("move_type", filter.OpEqual, string(targetMoveType))
	f.Add("state", filter.OpEqual, string(accounting.MoveStatePosted))
	if partnerID != nil && *partnerID > 0 {
		f.Add("partner_id", filter.OpEqual, *partnerID)
	}

	res, err := uc.accountingService.ListMoves(ctx, f, pagination.PageRequest{Page: 1, Limit: 1000})
	if err != nil {
		return nil, err
	}

	report := &payment.AgingReport{
		ReportType: reportType,
		AsOfDate:   asOfDate,
	}

	partnerMap := make(map[int64]*payment.PartnerAgingItem)

	for _, inv := range res.Items {
		if inv.MoveType != targetMoveType {
			continue
		}
		if inv.AmountResidual <= 0.0001 {
			continue
		}
		if inv.PartnerID == nil {
			continue
		}
		pid := *inv.PartnerID

		refDate := inv.Date
		if inv.InvoiceDueDate != nil && !inv.InvoiceDueDate.IsZero() {
			refDate = *inv.InvoiceDueDate
		} else if inv.InvoiceDate != nil && !inv.InvoiceDate.IsZero() {
			refDate = *inv.InvoiceDate
		}

		bucket := payment.CategorizeAging(asOfDate, refDate, inv.AmountResidual)
		report.Total.Add(bucket)

		item, exists := partnerMap[pid]
		if !exists {
			pName := fmt.Sprintf("Partner #%d", pid)
			if uc.partnerRepo != nil {
				if pObj, err := uc.partnerRepo.GetByID(ctx, pid); err == nil && pObj != nil {
					pName = pObj.Name
				}
			}
			item = &payment.PartnerAgingItem{
				PartnerID:   pid,
				PartnerName: pName,
			}
			partnerMap[pid] = item
		}
		item.Buckets.Add(bucket)
		item.InvoiceCount++
	}

	for _, item := range partnerMap {
		report.Partners = append(report.Partners, *item)
	}

	sort.Slice(report.Partners, func(i, j int) bool {
		return report.Partners[i].Buckets.Total > report.Partners[j].Buckets.Total
	})

	return report, nil
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
