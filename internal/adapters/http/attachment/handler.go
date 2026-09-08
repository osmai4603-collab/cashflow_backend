package attachmenthttp

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	attachmentusecase "cashflow_backend/internal/usecase/attachment"

	"github.com/go-chi/chi/v5"
)

// UploadDir is the base directory where uploaded files are persisted.
var UploadDir = "uploads"

// Handler serves HTTP requests for the Attachment domain.
type Handler struct {
	useCase   attachmentusecase.UseCase
	logger    *slog.Logger
	uploadDir string
}

// NewHandler constructs a new Handler.
func NewHandler(useCase attachmentusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase:   useCase,
		logger:    logger,
		uploadDir: UploadDir,
	}
}

// upload saves multipart file bytes under the configured upload directory.
func (h *Handler) saveUploadedFile(r *http.Request) (name, filename, mimetype, storagePath string, size int64, err error) {
	_ = r.ParseMultipartForm(64 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		return "", "", "", "", 0, platformerrors.BadRequest("missing file part 'file'", err)
	}
	defer file.Close()

	filename = filepath.Base(header.Filename)
	mimetype = header.Header.Get("Content-Type")
	if mimetype == "" {
		mimetype = "application/octet-stream"
	}

	name = r.FormValue("name")
	if name == "" {
		name = filename
	}

	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		return "", "", "", "", 0, platformerrors.Internal("failed to create upload directory", err)
	}

	storagePath = filepath.Join(h.uploadDir, filename)
	dest, err := os.Create(storagePath)
	if err != nil {
		return "", "", "", "", 0, platformerrors.Internal("failed to create upload file", err)
	}
	defer dest.Close()

	size, err = io.Copy(dest, file)
	if err != nil {
		return "", "", "", "", 0, platformerrors.Internal("failed to write upload file", err)
	}

	return name, filename, mimetype, storagePath, size, nil
}

func parseOptionalInt64(formValue string) *int64 {
	trimmed := strings.TrimSpace(formValue)
	if trimmed == "" {
		return nil
	}
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

// Upload handles POST /api/v1/attachments (multipart/form-data with a 'file' part)
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	name, filename, mimetype, storagePath, size, err := h.saveUploadedFile(r)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	in := attachmentusecase.CreateAttachmentInput{
		Name:        name,
		Filename:    filename,
		MimeType:    mimetype,
		FileSize:    size,
		StoragePath: storagePath,
		ResModel:    r.FormValue("res_model"),
		ResID:       parseOptionalInt64(r.FormValue("res_id")),
		Description: r.FormValue("description"),
		CompanyID:   parseOptionalInt64(r.FormValue("company_id")),
	}

	created, err := h.useCase.CreateAttachment(r.Context(), in)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToAttachmentResponse(created))
}

// GetByID handles GET /api/v1/attachments/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseAttachmentID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid attachment ID in path", err))
		return
	}

	a, err := h.useCase.GetAttachment(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAttachmentResponse(a))
}

// ListByModel handles GET /api/v1/attachments?res_model=...&res_id=...
func (h *Handler) ListByModel(w http.ResponseWriter, r *http.Request) {
	resModel := strings.TrimSpace(r.URL.Query().Get("res_model"))
	if resModel == "" {
		response.Error(w, r, platformerrors.BadRequest("res_model query parameter is required"))
		return
	}

	var resID int64
	if rid := r.URL.Query().Get("res_id"); rid != "" {
		if id, err := strconv.ParseInt(rid, 10, 64); err == nil {
			resID = id
		}
	}

	pageReq := pagination.Parse(r)
	result, err := h.useCase.ListByModel(r.Context(), resModel, resID, pageReq)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToAttachmentResponseList(result.Items), result)
}

// Download handles GET /api/v1/attachments/{id}/download
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := parseAttachmentID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid attachment ID in path", err))
		return
	}

	a, err := h.useCase.GetAttachment(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	if a.StoragePath == "" {
		response.Error(w, r, platformerrors.NotFound("attachment has no stored file"))
		return
	}

	w.Header().Set("Content-Type", a.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(a.Filename))
	http.ServeFile(w, r, a.StoragePath)
}

// Archive handles DELETE /api/v1/attachments/{id} (soft-delete)
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	id, err := parseAttachmentID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid attachment ID in path", err))
		return
	}

	if err := h.useCase.ArchiveAttachment(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func parseAttachmentID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}
