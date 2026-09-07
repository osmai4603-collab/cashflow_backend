package attachmentstorage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	attachmentstorage "cashflow_backend/internal/adapters/storage/attachment"
	"cashflow_backend/internal/domain/attachment"
	"cashflow_backend/internal/platform/pagination"
)

func TestAttachmentMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := attachmentstorage.NewMemoryRepo()

	resID := int64(7)
	a := &attachment.Attachment{
		Name:      "Contract",
		Filename:  "contract.pdf",
		FileSize:  1024,
		ResModel:  "crm.lead",
		ResID:     &resID,
		CompanyID: int64Ptr(1),
	}
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("unexpected error creating attachment: %v", err)
	}
	if a.ID <= 0 || !a.Active {
		t.Errorf("expected positive id and active=true")
	}

	fetched, err := repo.GetByID(ctx, a.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching attachment: %v", err)
	}
	if fetched.ResModel != "crm.lead" {
		t.Errorf("expected res_model crm.lead, got %q", fetched.ResModel)
	}

	fetched.Description = "Signed"
	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("unexpected error updating attachment: %v", err)
	}

	if err := repo.Delete(ctx, a.ID); err != nil {
		t.Fatalf("unexpected error deleting attachment: %v", err)
	}
	_, err = repo.GetByID(ctx, a.ID)
	if err == nil {
		t.Errorf("expected error getting archived attachment, got nil")
	}
}

func TestAttachmentMemoryRepo_ListByModel(t *testing.T) {
	ctx := context.Background()
	repo := attachmentstorage.NewMemoryRepo()

	res1, res2 := int64(100), int64(200)
	docs := []*attachment.Attachment{
		{Name: "a", Filename: "a", ResModel: "sale.order", ResID: &res1},
		{Name: "b", Filename: "b", ResModel: "sale.order", ResID: &res1},
		{Name: "c", Filename: "c", ResModel: "sale.order", ResID: &res2},
		{Name: "d", Filename: "d", ResModel: "stock.picking", ResID: &res1},
		{Name: "e", Filename: "e"},
	}
	for _, a := range docs {
		if err := repo.Create(ctx, a); err != nil {
			t.Fatalf("failed to seed attachment: %v", err)
		}
	}

	// Filter by model + res_id
	res, err := repo.ListByModel(ctx, "sale.order", res1, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing by model: %v", err)
	}
	if res.TotalItems != 2 {
		t.Errorf("expected 2 attachments, got %d", res.TotalItems)
	}

	// Filter by model only
	resAll, err := repo.ListByModel(ctx, "sale.order", 0, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing by model only: %v", err)
	}
	if resAll.TotalItems != 3 {
		t.Errorf("expected 3 sale.order attachments, got %d", resAll.TotalItems)
	}
}

func TestAttachmentMemoryRepo_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := attachmentstorage.NewMemoryRepo()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_ = repo.Create(ctx, &attachment.Attachment{Name: fmt.Sprintf("f%d", i), Filename: "f", ResModel: "x"})
		}(i)
	}
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = repo.ListByModel(ctx, "x", 0, pagination.PageRequest{Page: 1, Limit: 25})
		}()
	}
	wg.Wait()

	res, err := repo.ListByModel(ctx, "x", 0, pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error after concurrent access: %v", err)
	}
	if res.TotalItems != n {
		t.Errorf("expected %d attachments, got %d", n, res.TotalItems)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
