package payment

import (
	"testing"
	"time"
)

func TestPayment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payment Payment
		wantErr bool
	}{
		{
			name: "valid inbound payment",
			payment: Payment{
				PartnerID:   1,
				JournalID:   3,
				Amount:      500.0,
				PaymentType: PaymentTypeInbound,
			},
			wantErr: false,
		},
		{
			name: "valid outbound payment",
			payment: Payment{
				PartnerID:     2,
				JournalID:     4,
				Amount:        350.50,
				PaymentType:   PaymentTypeOutbound,
				PartnerType:   PartnerTypeSupplier,
				PaymentMethod: PaymentMethodBankTransfer,
			},
			wantErr: false,
		},
		{
			name: "missing partner",
			payment: Payment{
				JournalID:   3,
				Amount:      100,
				PaymentType: PaymentTypeInbound,
			},
			wantErr: true,
		},
		{
			name: "missing journal",
			payment: Payment{
				PartnerID:   1,
				Amount:      100,
				PaymentType: PaymentTypeInbound,
			},
			wantErr: true,
		},
		{
			name: "zero or negative amount",
			payment: Payment{
				PartnerID:   1,
				JournalID:   3,
				Amount:      0,
				PaymentType: PaymentTypeInbound,
			},
			wantErr: true,
		},
		{
			name: "invalid payment type",
			payment: Payment{
				PartnerID:   1,
				JournalID:   3,
				Amount:      100,
				PaymentType: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payment.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if tt.payment.ResidualAmount != tt.payment.Amount {
					t.Errorf("ResidualAmount = %.4f, want %.4f", tt.payment.ResidualAmount, tt.payment.Amount)
				}
				if tt.payment.Name != "/" && tt.payment.Name == "" {
					t.Errorf("expected default name '/', got '%s'", tt.payment.Name)
				}
			}
		})
	}
}

func TestPayment_LifecycleAndReconciliation(t *testing.T) {
	p := &Payment{
		PartnerID:   10,
		JournalID:   3,
		Amount:      1000.0,
		PaymentType: PaymentTypeInbound,
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// 1. Post payment
	if err := p.Post("PAY/2026/00001"); err != nil {
		t.Fatalf("unexpected post error: %v", err)
	}
	if p.State != PaymentStatePosted {
		t.Errorf("expected state %s, got %s", PaymentStatePosted, p.State)
	}
	if p.Name != "PAY/2026/00001" {
		t.Errorf("expected name PAY/2026/00001, got %s", p.Name)
	}

	// Double posting should error
	if err := p.Post("PAY/2026/00002"); err == nil {
		t.Errorf("expected error on double post, got nil")
	}

	// 2. Partial reconciliation
	if err := p.AllocateReconciliation(400.0); err != nil {
		t.Fatalf("unexpected allocate error: %v", err)
	}
	if p.ReconciledAmount != 400.0 || p.ResidualAmount != 600.0 {
		t.Errorf("expected reconciled 400, residual 600; got %.2f, %.2f", p.ReconciledAmount, p.ResidualAmount)
	}
	if p.State != PaymentStatePosted {
		t.Errorf("expected state %s, got %s", PaymentStatePosted, p.State)
	}

	// Exceeding residual should error
	if err := p.AllocateReconciliation(700.0); err == nil {
		t.Errorf("expected error when allocating more than residual, got nil")
	}

	// 3. Complete reconciliation
	if err := p.AllocateReconciliation(600.0); err != nil {
		t.Fatalf("unexpected allocate error: %v", err)
	}
	if p.ResidualAmount != 0 || p.ReconciledAmount != 1000.0 {
		t.Errorf("expected reconciled 1000, residual 0; got %.2f, %.2f", p.ReconciledAmount, p.ResidualAmount)
	}
	if p.State != PaymentStateReconciled {
		t.Errorf("expected state %s, got %s", PaymentStateReconciled, p.State)
	}

	// 4. Unreconcile
	if err := p.Unreconcile(); err != nil {
		t.Fatalf("unexpected unreconcile error: %v", err)
	}
	if p.ResidualAmount != 1000.0 || p.ReconciledAmount != 0 {
		t.Errorf("expected restored residual 1000, got %.2f", p.ResidualAmount)
	}
	if p.State != PaymentStatePosted {
		t.Errorf("expected state %s, got %s", PaymentStatePosted, p.State)
	}

	// 5. Cancel
	if err := p.Cancel(); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if p.State != PaymentStateCancelled {
		t.Errorf("expected state %s, got %s", PaymentStateCancelled, p.State)
	}
}

func TestCategorizeAging(t *testing.T) {
	asOf := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		dueDate  time.Time
		amount   float64
		expected func(b AgingBucket) bool
	}{
		{
			name:    "current / future due date",
			dueDate: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
			amount:  150.0,
			expected: func(b AgingBucket) bool {
				return b.Current == 150.0 && b.Total == 150.0
			},
		},
		{
			name:    "1 to 30 days overdue",
			dueDate: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), // ~17 days
			amount:  200.0,
			expected: func(b AgingBucket) bool {
				return b.Days1_30 == 200.0 && b.Total == 200.0
			},
		},
		{
			name:    "31 to 60 days overdue",
			dueDate: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), // ~48 days
			amount:  300.0,
			expected: func(b AgingBucket) bool {
				return b.Days31_60 == 300.0 && b.Total == 300.0
			},
		},
		{
			name:    "61 to 90 days overdue",
			dueDate: time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), // ~78 days
			amount:  400.0,
			expected: func(b AgingBucket) bool {
				return b.Days61_90 == 400.0 && b.Total == 400.0
			},
		},
		{
			name:    "91 to 120 days overdue",
			dueDate: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), // ~109 days
			amount:  500.0,
			expected: func(b AgingBucket) bool {
				return b.Days91_120 == 500.0 && b.Total == 500.0
			},
		},
		{
			name:    "older than 120 days",
			dueDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), // > 200 days
			amount:  600.0,
			expected: func(b AgingBucket) bool {
				return b.Older == 600.0 && b.Total == 600.0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := CategorizeAging(asOf, tt.dueDate, tt.amount)
			if !tt.expected(b) {
				t.Errorf("CategorizeAging returned unexpected bucket: %+v", b)
			}
		})
	}

	// Test AgingBucket.Add
	b1 := AgingBucket{Current: 100, Days1_30: 50, Total: 150}
	b2 := AgingBucket{Current: 50, Older: 200, Total: 250}
	b1.Add(b2)

	if b1.Current != 150 || b1.Days1_30 != 50 || b1.Older != 200 || b1.Total != 400 {
		t.Errorf("AgingBucket.Add failed, got %+v", b1)
	}
}
