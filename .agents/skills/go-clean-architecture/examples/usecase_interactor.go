package cleanarch

import (
	"context"
	"fmt"
	"time"
)

// =============================================================================
// LAYER 2: USE CASE LAYER (Application Business Rules & Ports)
// =============================================================================
// Rules:
// - Imports domain package.
// - Declares interfaces (Ports) for persistence and external integrations.
// - Never imports net/http, sql, pgx, or third-party web frameworks.
// =============================================================================

// InvoiceRepository is a Driven (Secondary) Port defined by the usecase layer.
type InvoiceRepository interface {
	GetByID(ctx context.Context, id string) (*Invoice, error)
	Save(ctx context.Context, invoice *Invoice) error
	NextID(ctx context.Context) (string, error)
}

// PaymentEventPublisher is a Driven Port for domain event notification.
type PaymentEventPublisher interface {
	PublishInvoicePaid(ctx context.Context, invoiceID string, amount float64) error
}

// PayInvoiceCommand is the input boundary parameter.
type PayInvoiceCommand struct {
	InvoiceID string
	PaidAt    time.Time
}

// PayInvoiceUseCase coordinates the payment operation.
type PayInvoiceUseCase struct {
	repo      InvoiceRepository
	publisher PaymentEventPublisher
}

func NewPayInvoiceUseCase(repo InvoiceRepository, pub PaymentEventPublisher) *PayInvoiceUseCase {
	return &PayInvoiceUseCase{
		repo:      repo,
		publisher: pub,
	}
}

func (uc *PayInvoiceUseCase) Execute(ctx context.Context, cmd PayInvoiceCommand) error {
	invoice, err := uc.repo.GetByID(ctx, cmd.InvoiceID)
	if err != nil {
		return fmt.Errorf("retrieve invoice: %w", err)
	}
	if invoice == nil {
		return fmt.Errorf("invoice %s: not found", cmd.InvoiceID)
	}

	// Enforce domain logic
	if err := invoice.MarkAsPaid(cmd.PaidAt); err != nil {
		return fmt.Errorf("apply payment: %w", err)
	}

	// Persist via repository port
	if err := uc.repo.Save(ctx, invoice); err != nil {
		return fmt.Errorf("persist invoice: %w", err)
	}

	// Publish domain event port
	if uc.publisher != nil {
		_ = uc.publisher.PublishInvoicePaid(ctx, invoice.ID(), invoice.Amount())
	}

	return nil
}
