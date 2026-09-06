package paymentstorage

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/domain/payment"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	// 1. Create
	p := &payment.Payment{
		Name:          "/",
		PaymentType:   payment.PaymentTypeInbound,
		PartnerID:     10,
		Amount:        500.0,
		JournalID:     3,
		PaymentMethod: payment.PaymentMethodBankTransfer,
		State:         payment.PaymentStateDraft,
		Active:        true,
	}
	if err := repo.CreatePayment(ctx, p); err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}
	if p.ID <= 0 {
		t.Errorf("expected positive ID, got %d", p.ID)
	}

	// 2. Get
	fetched, err := repo.GetPaymentByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get payment: %v", err)
	}
	if fetched.Amount != 500.0 || fetched.PartnerID != 10 {
		t.Errorf("fetched data mismatch: %+v", fetched)
	}

	// 3. Update
	fetched.Ref = "Updated Ref"
	fetched.Amount = 600.0
	if err := repo.UpdatePayment(ctx, fetched); err != nil {
		t.Fatalf("failed to update payment: %v", err)
	}
	updated, err := repo.GetPaymentByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get updated payment: %v", err)
	}
	if updated.Ref != "Updated Ref" || updated.Amount != 600.0 {
		t.Errorf("updated fields mismatch: %+v", updated)
	}

	// 4. Sequence
	seq1, err := repo.NextSequence(ctx, 2026)
	if err != nil {
		t.Fatalf("failed to generate sequence: %v", err)
	}
	seq2, err := repo.NextSequence(ctx, 2026)
	if err != nil {
		t.Fatalf("failed to generate second sequence: %v", err)
	}
	if seq1 != "PAY/2026/00001" || seq2 != "PAY/2026/00002" {
		t.Errorf("unexpected sequences: %s, %s", seq1, seq2)
	}

	// 5. List with Filter
	f := filter.NewFilter()
	f.Add("partner_id", filter.OpEqual, int64(10))
	res, err := repo.ListPayments(ctx, f, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("failed to list payments: %v", err)
	}
	if res.TotalItems != 1 || len(res.Items) != 1 {
		t.Errorf("expected 1 payment in filtered list, got %d", res.TotalItems)
	}

	// 6. Delete
	if err := repo.DeletePayment(ctx, p.ID); err != nil {
		t.Fatalf("failed to delete payment: %v", err)
	}
	if _, err := repo.GetPaymentByID(ctx, p.ID); err == nil {
		t.Errorf("expected error getting deleted payment, got nil")
	}
}

func TestMemoryRepo_Reconciliations(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	r1 := &payment.PaymentReconciliation{
		PaymentID:    1,
		InvoiceID:    101,
		Amount:       250.0,
		ReconciledAt: time.Now().UTC(),
	}
	r2 := &payment.PaymentReconciliation{
		PaymentID:    1,
		InvoiceID:    102,
		Amount:       150.0,
		ReconciledAt: time.Now().UTC(),
	}

	if err := repo.CreateReconciliation(ctx, r1); err != nil {
		t.Fatalf("failed to create recon 1: %v", err)
	}
	if err := repo.CreateReconciliation(ctx, r2); err != nil {
		t.Fatalf("failed to create recon 2: %v", err)
	}

	// Fetch by PaymentID
	pRecons, err := repo.GetReconciliationsByPaymentID(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get by payment id: %v", err)
	}
	if len(pRecons) != 2 {
		t.Errorf("expected 2 reconciliations, got %d", len(pRecons))
	}

	// Fetch by InvoiceID
	invRecons, err := repo.GetReconciliationsByInvoiceID(ctx, 101)
	if err != nil {
		t.Fatalf("failed to get by invoice id: %v", err)
	}
	if len(invRecons) != 1 || invRecons[0].Amount != 250.0 {
		t.Errorf("unexpected invoice reconciliations: %+v", invRecons)
	}

	// Delete
	if err := repo.DeleteReconciliationsByPaymentID(ctx, 1); err != nil {
		t.Fatalf("failed to delete reconciliations: %v", err)
	}
	pReconsAfter, err := repo.GetReconciliationsByPaymentID(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get after delete: %v", err)
	}
	if len(pReconsAfter) != 0 {
		t.Errorf("expected 0 reconciliations after delete, got %d", len(pReconsAfter))
	}
}
