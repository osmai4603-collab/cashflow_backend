package attachment

import (
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Attachment represents a file attachment linked to any model (ir.attachment in Odoo).
type Attachment struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Filename    string       `json:"filename"`
	MimeType    string       `json:"mimetype"`
	FileSize    int64        `json:"file_size"`
	Checksum    string       `json:"checksum,omitempty"`
	StoragePath string       `json:"storage_path,omitempty"`
	ResModel    string       `json:"res_model,omitempty"`
	ResID       *int64       `json:"res_id,omitempty"`
	Description string       `json:"description,omitempty"`
	CompanyID   *int64       `json:"company_id,omitempty"`
	Active      bool         `json:"active"`
	Audit       audit.Fields `json:"audit"`
}

// Validate ensures the attachment entity satisfies all domain invariants.
func (a *Attachment) Validate() error {
	trimmedName := strings.TrimSpace(a.Name)
	if trimmedName == "" {
		return platformerrors.Validation("attachment name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if len(trimmedName) > 255 {
		return platformerrors.Validation("attachment name exceeds maximum length", map[string]string{
			"name": "must not exceed 255 characters",
		})
	}
	a.Name = trimmedName

	trimmedFilename := strings.TrimSpace(a.Filename)
	if trimmedFilename == "" {
		return platformerrors.Validation("filename is required", map[string]string{
			"filename": "cannot be empty",
		})
	}
	a.Filename = trimmedFilename

	if strings.TrimSpace(a.MimeType) == "" {
		a.MimeType = "application/octet-stream"
	}

	if a.FileSize < 0 {
		return platformerrors.Validation("file size cannot be negative", map[string]string{
			"file_size": "must be >= 0",
		})
	}

	return nil
}
