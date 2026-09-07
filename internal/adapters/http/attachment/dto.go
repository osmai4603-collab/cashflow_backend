package attachmenthttp

import (
	"time"

	"cashflow_backend/internal/domain/attachment"
	attachmentusecase "cashflow_backend/internal/usecase/attachment"
)

// CreateAttachmentRequest represents the JSON metadata payload used by the upload endpoint.
// File bytes are handled at the HTTP layer; this payload carries metadata only.
type CreateAttachmentRequest struct {
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

// ToInput maps the HTTP request DTO to the application usecase input.
func (r CreateAttachmentRequest) ToInput() attachmentusecase.CreateAttachmentInput {
	return attachmentusecase.CreateAttachmentInput{
		Name:        r.Name,
		Filename:    r.Filename,
		MimeType:    r.MimeType,
		FileSize:    r.FileSize,
		Checksum:    r.Checksum,
		StoragePath: r.StoragePath,
		ResModel:    r.ResModel,
		ResID:       r.ResID,
		Description: r.Description,
		CompanyID:   r.CompanyID,
	}
}

// AttachmentResponse formats attachment data for HTTP JSON client output.
type AttachmentResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	MimeType    string `json:"mimetype"`
	FileSize    int64  `json:"file_size"`
	Checksum    string `json:"checksum,omitempty"`
	StoragePath string `json:"storage_path,omitempty"`
	ResModel    string `json:"res_model,omitempty"`
	ResID       *int64 `json:"res_id,omitempty"`
	Description string `json:"description,omitempty"`
	CompanyID   *int64 `json:"company_id,omitempty"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ToAttachmentResponse converts a domain Attachment entity into AttachmentResponse DTO.
func ToAttachmentResponse(a *attachment.Attachment) AttachmentResponse {
	if a == nil {
		return AttachmentResponse{}
	}
	return AttachmentResponse{
		ID:          a.ID,
		Name:        a.Name,
		Filename:    a.Filename,
		MimeType:    a.MimeType,
		FileSize:    a.FileSize,
		Checksum:    a.Checksum,
		StoragePath: a.StoragePath,
		ResModel:    a.ResModel,
		ResID:       a.ResID,
		Description: a.Description,
		CompanyID:   a.CompanyID,
		Active:      a.Active,
		CreatedAt:   a.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   a.Audit.UpdatedAt.Format(time.RFC3339),
	}
}

// ToAttachmentResponseList converts a slice of domain attachments into response DTOs.
func ToAttachmentResponseList(items []attachment.Attachment) []AttachmentResponse {
	result := make([]AttachmentResponse, len(items))
	for i, item := range items {
		result[i] = ToAttachmentResponse(&item)
	}
	return result
}
