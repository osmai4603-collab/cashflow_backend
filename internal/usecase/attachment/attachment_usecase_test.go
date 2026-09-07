package attachmentusecase_test

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	attachmentstorage "cashflow_backend/internal/adapters/storage/attachment"
	"cashflow_backend/internal/platform/pagination"
	attachmentusecase "cashflow_backend/internal/usecase/attachment"
)

func setupTestUseCase() *attachmentusecase.AttachmentUseCase {
	repo := attachmentstorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return attachmentusecase.New(repo, logger)
}

func TestAttachmentUseCase_CreateAndGet(t *testing.T) {
	uc := setupTestUseCase()
	ctx := context.Background()

	resID := int64(42)
	a, err := uc.CreateAttachment(ctx, attachmentusecase.CreateAttachmentInput{
		Name:        "Invoice PDF",
		Filename:    "invoice.pdf",
		FileSize:    2048,
		Checksum:    "abc123",
		ResModel:    "sale.order",
		ResID:       &resID,
		Description: "Scanned invoice",
	})
	if err != nil {
		t.Fatalf("unexpected error creating attachment: %v", err)
	}
	if a.ID <= 0 {
		t.Errorf("expected positive attachment ID, got %d", a.ID)
	}
	if a.MimeType == "" {
		t.Errorf("expected default mimetype when omitted")
	}

	// Get
	fetched, err := uc.GetAttachment(ctx, a.ID)
	if err != nil {
		t.Fatalf("failed to get attachment: %v", err)
	}
	if fetched.Name != "Invoice PDF" {
		t.Errorf("expected Invoice PDF, got %q", fetched.Name)
	}

	// Invalid id
	_, err = uc.GetAttachment(ctx, 0)
	if err == nil {
		t.Fatalf("expected error for id=0, got nil")
	}

	// Validation failure (empty name)
	_, err = uc.CreateAttachment(ctx, attachmentusecase.CreateAttachmentInput{Filename: "x"})
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}
}

func TestAttachmentUseCase_ListByModelAndArchive(t *testing.T) {
	uc := setupTestUseCase()
	ctx := context.Background()

	for _, resID := range []int64{1, 1, 2} {
		_, err := uc.CreateAttachment(ctx, attachmentusecase.CreateAttachmentInput{
			Name:     "file",
			Filename: "f",
			ResModel: "stock.picking",
			ResID:    &resID,
		})
		if err != nil {
			t.Fatalf("failed to create attachment: %v", err)
		}
	}

	res, err := uc.ListByModel(ctx, "stock.picking", 1, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing attachments: %v", err)
	}
	if res.TotalItems != 2 {
		t.Errorf("expected 2 attachments for res_id=1, got %d", res.TotalItems)
	}

	// Archive one
	item := res.Items[0]
	if err := uc.ArchiveAttachment(ctx, item.ID); err != nil {
		t.Fatalf("failed to archive attachment: %v", err)
	}
	_, err = uc.GetAttachment(ctx, item.ID)
	if err == nil {
		t.Fatalf("expected not found after archive, got nil")
	}
}

func TestAttachmentUseCase_HashFileSHA1(t *testing.T) {
	uc := setupTestUseCase()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	sum, err := attachmentusecase.HashFileSHA1(path)
	if err != nil {
		t.Fatalf("unexpected error hashing file: %v", err)
	}
	h := sha1.Sum(content)
	if sum != hex.EncodeToString(h[:]) {
		t.Errorf("expected sha1 %x, got %s", h, sum)
	}

	// Auto-checksum when StoragePath provided
	ctx := context.Background()
	checksummed, err := uc.CreateAttachment(ctx, attachmentusecase.CreateAttachmentInput{
		Name:        "Hashed File",
		Filename:    "test.txt",
		FileSize:    int64(len(content)),
		StoragePath: path,
	})
	if err != nil {
		t.Fatalf("unexpected error creating attachment with checksum: %v", err)
	}
	if checksummed.Checksum != sum {
		t.Errorf("expected auto-computed checksum %s, got %q", sum, checksummed.Checksum)
	}
}
