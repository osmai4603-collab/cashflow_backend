package attachmentusecase

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log/slog"
	"os"

	"cashflow_backend/internal/domain/attachment"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
)

// CreateAttachmentInput defines input parameters for registering an attachment.
type CreateAttachmentInput struct {
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	MimeType    string `json:"mimetype"`
	FileSize    int64  `json:"file_size"`
	Checksum    string `json:"checksum"`
	StoragePath string `json:"storage_path"`
	ResModel    string `json:"res_model"`
	ResID       *int64 `json:"res_id"`
	Description string `json:"description"`
	CompanyID   *int64 `json:"company_id"`
}

// UpdateAttachmentInput defines input parameters for modifying an attachment.
type UpdateAttachmentInput struct {
	Name        *string `json:"name"`
	Filename    *string `json:"filename"`
	MimeType    *string `json:"mimetype"`
	ResModel    *string `json:"res_model"`
	ResID       *int64  `json:"res_id"`
	Description *string `json:"description"`
}

// UseCase defines the application interface for Attachment domain operations.
type UseCase interface {
	CreateAttachment(ctx context.Context, in CreateAttachmentInput) (*attachment.Attachment, error)
	GetAttachment(ctx context.Context, id int64) (*attachment.Attachment, error)
	UpdateAttachment(ctx context.Context, id int64, in UpdateAttachmentInput) (*attachment.Attachment, error)
	ArchiveAttachment(ctx context.Context, id int64) error
	ListByModel(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[attachment.Attachment], error)
}

// AttachmentUseCase implements the UseCase interface.
type AttachmentUseCase struct {
	repo   attachment.Repository
	logger *slog.Logger
}

// New constructs a new AttachmentUseCase.
func New(repo attachment.Repository, logger *slog.Logger) *AttachmentUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &AttachmentUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *AttachmentUseCase) CreateAttachment(ctx context.Context, in CreateAttachmentInput) (*attachment.Attachment, error) {
	if in.MimeType == "" {
		in.MimeType = "application/octet-stream"
	}
	if in.Checksum == "" && in.StoragePath != "" {
		checksum, err := HashFileSHA1(in.StoragePath)
		if err != nil {
			uc.logger.Warn("failed to compute file checksum", "error", err, "path", in.StoragePath)
		} else {
			in.Checksum = checksum
		}
	}

	a := &attachment.Attachment{
		Name:        in.Name,
		Filename:    in.Filename,
		MimeType:    in.MimeType,
		FileSize:    in.FileSize,
		Checksum:    in.Checksum,
		StoragePath: in.StoragePath,
		ResModel:    in.ResModel,
		ResID:       in.ResID,
		Description: in.Description,
		CompanyID:   in.CompanyID,
		Active:      true,
		Audit:       audit.NewFields(ctx),
	}

	if err := a.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, a); err != nil {
		uc.logger.Error("failed to create attachment", "error", err, "name", a.Name)
		return nil, err
	}

	uc.logger.Info("attachment created successfully", "id", a.ID, "name", a.Name)
	return a, nil
}

func (uc *AttachmentUseCase) GetAttachment(ctx context.Context, id int64) (*attachment.Attachment, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid attachment id")
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *AttachmentUseCase) UpdateAttachment(ctx context.Context, id int64, in UpdateAttachmentInput) (*attachment.Attachment, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid attachment id")
	}

	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		a.Name = *in.Name
	}
	if in.Filename != nil {
		a.Filename = *in.Filename
	}
	if in.MimeType != nil {
		a.MimeType = *in.MimeType
	}
	if in.ResModel != nil {
		a.ResModel = *in.ResModel
	}
	if in.ResID != nil {
		a.ResID = in.ResID
	}
	if in.Description != nil {
		a.Description = *in.Description
	}

	a.Audit.Touch(ctx)

	if err := a.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, a); err != nil {
		uc.logger.Error("failed to update attachment", "error", err, "id", id)
		return nil, err
	}

	uc.logger.Info("attachment updated successfully", "id", a.ID)
	return a, nil
}

func (uc *AttachmentUseCase) ArchiveAttachment(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid attachment id")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("attachment archived successfully", "id", id)
	return nil
}

func (uc *AttachmentUseCase) ListByModel(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[attachment.Attachment], error) {
	return uc.repo.ListByModel(ctx, resModel, resID, page)
}

// HashFileSHA1 computes the SHA-1 checksum of a file on disk.
func HashFileSHA1(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", platformerrors.Internal("failed to open attachment file", err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", platformerrors.Internal("failed to hash attachment file", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
