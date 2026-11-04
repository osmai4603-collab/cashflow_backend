package accounting

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	platformerr "cashflow_backend/internal/platform/errors"

	"github.com/google/uuid"
)

type MoveHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewMoveHandler(uc *UseCases, logger *slog.Logger) *MoveHandler {
	return &MoveHandler{uc: uc, logger: handlerLogger(logger, "move")}
}

func (h *MoveHandler) HandleCreateDraft(w http.ResponseWriter, r *http.Request) {
	var request CreateMoveRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}

	actorID, err := actorIDFromRequest(r)
	if err != nil {
		platformerr.WriteHTTPError(w, r, platformerr.Unauthorized("accounting.actor", "invalid authenticated user", err), h.logger)
		return
	}
	response, err := h.uc.CreateDraftMove.ExecuteAs(r.Context(), actorID, request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *MoveHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	moveID, err := parseRouteUUID(r, "id")
	if err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	response, err := h.uc.GetMove.Execute(r.Context(), moveID)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *MoveHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := MoveFilter{State: MoveState(query.Get("state")), Limit: 50}
	if value := query.Get("journal_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			writeBadRequest(w, r, h.logger, errors.New("journal_id must be a UUID"))
			return
		}
		filter.JournalID = &id
	}
	if value := query.Get("from"); value != "" {
		date, err := parseDate(value, "from")
		if err != nil {
			writeBadRequest(w, r, h.logger, err)
			return
		}
		filter.FromDate = &date
	}
	if value := query.Get("to"); value != "" {
		date, err := parseDate(value, "to")
		if err != nil {
			writeBadRequest(w, r, h.logger, err)
			return
		}
		filter.ToDate = &date
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			writeBadRequest(w, r, h.logger, errors.New("limit must be an integer between 1 and 200"))
			return
		}
		filter.Limit = limit
	}
	if value := query.Get("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil {
			writeBadRequest(w, r, h.logger, errors.New("offset must be a non-negative integer"))
			return
		}
		filter.Offset = offset
	}
	moves, total, err := h.uc.ListMoves.Execute(r.Context(), filter)
	if err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, ListMovesResponse{
		Moves: moves, Total: total, Limit: filter.Limit, Offset: filter.Offset,
	})
}

func (h *MoveHandler) HandleUpdateDraft(w http.ResponseWriter, r *http.Request) {
	moveID, err := parseRouteUUID(r, "id")
	if err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	var request CreateMoveRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}
	response, err := h.uc.UpdateDraftMove.Execute(r.Context(), moveID, request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *MoveHandler) HandleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	moveID, err := parseRouteUUID(r, "id")
	if err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := h.uc.DeleteDraftMove.Execute(r.Context(), moveID); err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MoveHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	moveID, err := parseRouteUUID(r, "id")
	if err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	actorID, err := actorIDFromRequest(r)
	if err != nil {
		platformerr.WriteHTTPError(w, r, platformerr.Unauthorized("accounting.actor", "invalid authenticated user", err), h.logger)
		return
	}
	response, err := h.uc.PostMove.Execute(r.Context(), moveID, actorID)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *MoveHandler) HandleReverse(w http.ResponseWriter, r *http.Request) {
	moveID, err := parseRouteUUID(r, "id")
	if err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	var request struct {
		Date   string `json:"date"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &request); err != nil {
		if !errors.Is(err, io.EOF) {
			writeBadRequest(w, r, h.logger, err)
			return
		}
	}

	reversalDate := time.Now().UTC()
	if request.Date != "" {
		reversalDate, err = time.Parse(dateLayout, request.Date)
		if err != nil || reversalDate.Format(dateLayout) != request.Date {
			writeBadRequest(w, r, h.logger, errors.New("date must be in YYYY-MM-DD format"))
			return
		}
	}
	actorID, err := actorIDFromRequest(r)
	if err != nil {
		platformerr.WriteHTTPError(w, r, platformerr.Unauthorized("accounting.actor", "invalid authenticated user", err), h.logger)
		return
	}

	response, err := h.uc.ReverseMove.Execute(r.Context(), moveID, actorID, reversalDate, request.Reason)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
