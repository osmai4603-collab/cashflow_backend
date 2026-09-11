package helpdeskhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/helpdesk"
	"cashflow_backend/internal/platform/response"
	helpdeskusecase "cashflow_backend/internal/usecase/helpdesk"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *helpdeskusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *helpdeskusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.useCase.ListTickets(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tickets)
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var t helpdesk.Ticket
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateTicket(r.Context(), &t)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	ticket, err := h.useCase.GetTicket(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ticket)
}

func (h *Handler) MoveToStage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct{ StageID int64 `json:"stage_id"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.MoveToStage(r.Context(), id, req.StageID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct{ UserID int64 `json:"user_id"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.AssignTicket(r.Context(), id, req.UserID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) Reply(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct{ Body string `json:"body"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.ReplyToTicket(r.Context(), id, req.Body); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ListArticles(w http.ResponseWriter, r *http.Request) {
	articles, err := h.useCase.ListArticles(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, articles)
}

func (h *Handler) GetArticle(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	article, err := h.useCase.GetArticleBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, article)
}

func (h *Handler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	var a helpdesk.KnowledgeArticle
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateArticle(r.Context(), &a)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var a helpdesk.KnowledgeArticle
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		response.Error(w, err)
		return
	}
	updated, err := h.useCase.UpdateArticle(r.Context(), id, &a)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, updated)
}
