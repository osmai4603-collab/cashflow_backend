package currencystorage

import (
	"context"
	"sort"
	"sync"
	"time"

	"cashflow_backend/internal/domain/currency"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRateRepo is a thread-safe in-memory implementation of currency.RateRepository.
type MemoryRateRepo struct {
	mu     sync.RWMutex
	rates  map[int64]*currency.CurrencyRate
	lastID int64
}

// NewMemoryRateRepo creates an initialized MemoryRateRepo.
func NewMemoryRateRepo() *MemoryRateRepo {
	return &MemoryRateRepo{
		rates: make(map[int64]*currency.CurrencyRate),
	}
}

func (r *MemoryRateRepo) Create(ctx context.Context, rate *currency.CurrencyRate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	rate.ID = r.lastID

	now := time.Now().UTC()
	if rate.Audit.CreatedAt.IsZero() {
		rate.Audit.CreatedAt = now
	}
	if rate.Audit.UpdatedAt.IsZero() {
		rate.Audit.UpdatedAt = now
	}

	clone := *rate
	r.rates[rate.ID] = &clone
	return nil
}

func (r *MemoryRateRepo) GetByID(ctx context.Context, id int64) (*currency.CurrencyRate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rate, exists := r.rates[id]
	if !exists {
		return nil, platformerrors.NotFound("currency rate not found")
	}

	clone := *rate
	return &clone, nil
}

func (r *MemoryRateRepo) GetLatestRate(ctx context.Context, currencyID int64, companyID *int64) (*currency.CurrencyRate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *currency.CurrencyRate
	for _, rate := range r.rates {
		if rate.CurrencyID != currencyID {
			continue
		}
		if companyID != nil {
			if rate.CompanyID == nil || *rate.CompanyID != *companyID {
				continue
			}
		}
		if latest == nil || rate.Date > latest.Date {
			clone := *rate
			latest = &clone
		}
	}

	if latest == nil {
		return nil, platformerrors.NotFound("no currency rate found")
	}
	return latest, nil
}

func (r *MemoryRateRepo) GetRateOnDate(ctx context.Context, currencyID int64, date string, companyID *int64) (*currency.CurrencyRate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best *currency.CurrencyRate
	for _, rate := range r.rates {
		if rate.CurrencyID != currencyID {
			continue
		}
		if companyID != nil {
			if rate.CompanyID == nil || *rate.CompanyID != *companyID {
				continue
			}
		}
		if rate.Date > date {
			continue
		}
		if best == nil || rate.Date > best.Date {
			clone := *rate
			best = &clone
		}
	}

	if best == nil {
		return nil, platformerrors.NotFound("no currency rate found on or before date")
	}
	return best, nil
}

func (r *MemoryRateRepo) ListByCurrency(ctx context.Context, currencyID int64, page pagination.PageRequest) (pagination.PageResult[currency.CurrencyRate], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*currency.CurrencyRate
	for _, rate := range r.rates {
		if rate.CurrencyID == currencyID {
			filtered = append(filtered, rate)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Date != filtered[j].Date {
			return filtered[i].Date > filtered[j].Date
		}
		return filtered[i].ID > filtered[j].ID
	})

	totalItems := int64(len(filtered))
	offset := page.Offset()
	limit := page.LimitClamped()

	var items []currency.CurrencyRate
	if offset < len(filtered) {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		for _, rate := range filtered[offset:end] {
			items = append(items, *rate)
		}
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}
