package paymentstorage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/payment"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo provides a thread-safe, comprehensive in-memory implementation of payment.Repository.
type MemoryRepo struct {
	mu              sync.RWMutex
	payments        map[int64]*payment.Payment
	reconciliations map[int64][]payment.PaymentReconciliation // paymentID -> []reconciliations
	providers       map[int64]*payment.PaymentProvider
	transactions    map[int64]*payment.PaymentTransaction
	seqCounters     map[int]int64
	lastPaymentID   int64
	lastReconID     int64
	lastProviderID  int64
	lastTransID     int64
}

// NewMemoryRepo initializes an empty MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	r := &MemoryRepo{
		payments:        make(map[int64]*payment.Payment),
		reconciliations: make(map[int64][]payment.PaymentReconciliation),
		providers:       make(map[int64]*payment.PaymentProvider),
		transactions:    make(map[int64]*payment.PaymentTransaction),
		seqCounters:     make(map[int]int64),
	}
	r.seedProviders()
	return r
}

func (r *MemoryRepo) seedProviders() {
	r.providers[1] = &payment.PaymentProvider{ID: 1, Name: "Stripe", Code: "stripe", Active: true, CompanyID: 1}
	r.providers[2] = &payment.PaymentProvider{ID: 2, Name: "PayPal", Code: "paypal", Active: true, CompanyID: 1}
	r.lastProviderID = 2
}

// ─────────────────────────────────────────────────────────────────────────────
// Payments CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreatePayment(ctx context.Context, p *payment.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastPaymentID++
	p.ID = r.lastPaymentID
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	clone := *p
	r.payments[p.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetPaymentByID(ctx context.Context, id int64) (*payment.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.payments[id]
	if !ok || !p.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("payment with id %d not found", id))
	}

	clone := *p
	return &clone, nil
}

func (r *MemoryRepo) UpdatePayment(ctx context.Context, p *payment.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.payments[p.ID]
	if !ok || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("payment with id %d not found", p.ID))
	}

	p.UpdatedAt = time.Now().UTC()
	clone := *p
	r.payments[p.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeletePayment(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.payments[id]
	if !ok || !p.Active {
		return platformerrors.NotFound(fmt.Sprintf("payment with id %d not found", id))
	}

	p.Active = false
	p.UpdatedAt = time.Now().UTC()
	delete(r.reconciliations, id)
	return nil
}

func (r *MemoryRepo) ListPayments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[payment.Payment], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []payment.Payment
	for _, p := range r.payments {
		if !p.Active {
			continue
		}

		if !r.matchesFilter(p, f) {
			continue
		}

		result = append(result, *p)
	}

	// Sorting
	sort.Slice(result, func(i, j int) bool {
		if strings.ToLower(page.SortOrder) == "asc" {
			return result[i].ID < result[j].ID
		}
		return result[i].ID > result[j].ID
	})

	totalItems := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= len(result) {
		return pagination.NewPageResult([]payment.Payment{}, totalItems, page), nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	paged := result[offset:end]
	return pagination.NewPageResult(paged, totalItems, page), nil
}

func (r *MemoryRepo) NextSequence(ctx context.Context, year int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seqCounters[year]++
	return fmt.Sprintf("PAY/%04d/%05d", year, r.seqCounters[year]), nil
}

func (r *MemoryRepo) matchesFilter(p *payment.Payment, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 {
		return true
	}

	for _, c := range f.Criteria {
		switch c.Field {
		case "partner_id":
			var pid int64
			switch v := c.Value.(type) {
			case int64:
				pid = v
			case int:
				pid = int64(v)
			case float64:
				pid = int64(v)
			}
			if p.PartnerID != pid {
				return false
			}
		case "journal_id":
			var jid int64
			switch v := c.Value.(type) {
			case int64:
				jid = v
			case int:
				jid = int64(v)
			case float64:
				jid = int64(v)
			}
			if p.JournalID != jid {
				return false
			}
		case "payment_type":
			if string(p.PaymentType) != fmt.Sprintf("%v", c.Value) {
				return false
			}
		case "state":
			if string(p.State) != fmt.Sprintf("%v", c.Value) {
				return false
			}
		case "payment_method":
			if string(p.PaymentMethod) != fmt.Sprintf("%v", c.Value) {
				return false
			}
		}
	}
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconciliations
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateReconciliation(ctx context.Context, rec *payment.PaymentReconciliation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastReconID++
	rec.ID = r.lastReconID
	if rec.ReconciledAt.IsZero() {
		rec.ReconciledAt = time.Now().UTC()
	}

	clone := *rec
	r.reconciliations[rec.PaymentID] = append(r.reconciliations[rec.PaymentID], clone)
	return nil
}

func (r *MemoryRepo) GetReconciliationsByPaymentID(ctx context.Context, paymentID int64) ([]payment.PaymentReconciliation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := r.reconciliations[paymentID]
	res := make([]payment.PaymentReconciliation, len(list))
	copy(res, list)
	return res, nil
}

func (r *MemoryRepo) GetReconciliationsByInvoiceID(ctx context.Context, invoiceID int64) ([]payment.PaymentReconciliation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []payment.PaymentReconciliation
	for _, list := range r.reconciliations {
		for _, item := range list {
			if item.InvoiceID == invoiceID {
				res = append(res, item)
			}
		}
	}
	return res, nil
}

func (r *MemoryRepo) DeleteReconciliationsByPaymentID(ctx context.Context, paymentID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.reconciliations, paymentID)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// External Transactions
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateTransaction(ctx context.Context, t *payment.PaymentTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastTransID++
	t.ID = r.lastTransID
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	clone := *t
	r.transactions[t.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetTransactionByID(ctx context.Context, id int64) (*payment.PaymentTransaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.transactions[id]
	if !ok {
		return nil, platformerrors.NotFound("transaction not found")
	}
	clone := *t
	return &clone, nil
}

func (r *MemoryRepo) GetTransactionByReference(ctx context.Context, ref string) (*payment.PaymentTransaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, t := range r.transactions {
		if t.Reference == ref {
			clone := *t
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("transaction not found")
}

func (r *MemoryRepo) UpdateTransaction(ctx context.Context, t *payment.PaymentTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.transactions[t.ID]; !ok {
		return platformerrors.NotFound("transaction not found")
	}

	t.UpdatedAt = time.Now().UTC()
	clone := *t
	r.transactions[t.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetProviderByCode(ctx context.Context, code string, companyID int64) (*payment.PaymentProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.providers {
		if p.Code == code && p.CompanyID == companyID {
			clone := *p
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("provider not found")
}
