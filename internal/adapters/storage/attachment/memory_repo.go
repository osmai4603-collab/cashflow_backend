package attachmentstorage

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/attachment"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory implementation of attachment.Repository.
type MemoryRepo struct {
	mu          sync.RWMutex
	attachments map[int64]*attachment.Attachment
	lastID      int64
}

// NewMemoryRepo creates an initialized MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		attachments: make(map[int64]*attachment.Attachment),
	}
}

func (r *MemoryRepo) Create(ctx context.Context, a *attachment.Attachment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	a.ID = r.lastID
	a.Active = true

	now := time.Now().UTC()
	if a.Audit.CreatedAt.IsZero() {
		a.Audit.CreatedAt = now
	}
	if a.Audit.UpdatedAt.IsZero() {
		a.Audit.UpdatedAt = now
	}

	clone := *a
	r.attachments[a.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*attachment.Attachment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, exists := r.attachments[id]
	if !exists || !a.Active {
		return nil, platformerrors.NotFound("attachment not found")
	}

	clone := *a
	return &clone, nil
}

func (r *MemoryRepo) Update(ctx context.Context, a *attachment.Attachment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.attachments[a.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("attachment not found")
	}

	a.Audit.UpdatedAt = time.Now().UTC()
	clone := *a
	r.attachments[a.ID] = &clone
	return nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.attachments[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("attachment not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListByModel(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[attachment.Attachment], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*attachment.Attachment
	for _, a := range r.attachments {
		if a.Active && strings.EqualFold(a.ResModel, resModel) && (resID <= 0 || (a.ResID != nil && *a.ResID == resID)) {
			filtered = append(filtered, a)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Audit.CreatedAt.Equal(filtered[j].Audit.CreatedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].Audit.CreatedAt.After(filtered[j].Audit.CreatedAt)
	})

	totalItems := int64(len(filtered))
	offset := page.Offset()
	limit := page.LimitClamped()

	var items []attachment.Attachment
	if offset < len(filtered) {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		for _, a := range filtered[offset:end] {
			items = append(items, *a)
		}
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}
