package accountinghttp

import (
	"net/http"

	"cashflow_backend/internal/domain/accounting"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"

	"github.com/go-chi/chi/v5"
)

// ProcessMoveEDI triggers electronic document generation for an invoice.
func (h *Handler) ProcessMoveEDI(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid move ID", err))
		return
	}

	format := accounting.EDIFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = accounting.EDIFormatZatcaPhase1
	}

	doc, err := h.useCase.ProcessMoveEDI(r.Context(), id, format)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToEDIDocumentResponse(doc))
}

// GetMoveEDIDocuments lists all electronic documents for a move.
func (h *Handler) GetMoveEDIDocuments(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid move ID", err))
		return
	}

	docs, err := h.useCase.GetMoveEDIDocuments(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	res := make([]EDIDocumentResponse, len(docs))
	for i, d := range docs {
		res[i] = ToEDIDocumentResponse(&d)
	}

	response.JSON(w, http.StatusOK, res)
}
