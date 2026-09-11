package marketinghttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/marketing"
	"cashflow_backend/internal/platform/response"
	marketingusecase "cashflow_backend/internal/usecase/marketing"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *marketingusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *marketingusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.useCase.ListCampaigns(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, campaigns)
}

func (h *Handler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var c marketing.MarketingCampaign
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateCampaign(r.Context(), &c)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) ListMailingLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.useCase.ListMailingLists(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, lists)
}

func (h *Handler) CreateMailingList(w http.ResponseWriter, r *http.Request) {
	var l marketing.MailingList
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateMailingList(r.Context(), &l)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) ImportContacts(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	// Simplified: in real app, parse multipart form CSV
	if err := h.useCase.ImportContacts(r.Context(), id, nil); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ListBlacklist(w http.ResponseWriter, r *http.Request) {
	blacklist, err := h.useCase.ListBlacklist(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, blacklist)
}

func (h *Handler) AddToBlacklist(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email string `json:"email"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.AddToBlacklist(r.Context(), req.Email); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateMassMailing(w http.ResponseWriter, r *http.Request) {
	var m marketing.MassMailing
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateMassMailing(r.Context(), &m)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) SendTestMailing(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct{ Email string `json:"email"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.SendTestMailing(r.Context(), id, req.Email); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ScheduleMailing(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.ScheduleMailing(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CancelMailing(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.CancelMailing(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) GetMailingStats(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	stats, err := h.useCase.GetMailingStats(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}

func (h *Handler) TrackOpen(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	h.useCase.TrackOpen(r.Context(), code)
	w.Header().Set("Content-Type", "image/gif")
	w.Write([]byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;"))
}

func (h *Handler) TrackClick(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	redirectURL, err := h.useCase.TrackClick(r.Context(), code)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if err := h.useCase.Unsubscribe(r.Context(), token); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Unsubscribed successfully"})
}

func (h *Handler) ListAutomations(w http.ResponseWriter, r *http.Request) {
	autos, err := h.useCase.ListAutomations(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, autos)
}

func (h *Handler) CreateAutomation(w http.ResponseWriter, r *http.Request) {
	var a marketing.MarketingAutomation
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateAutomation(r.Context(), &a)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) UpdateAutomation(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var a marketing.MarketingAutomation
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		response.Error(w, err)
		return
	}
	updated, err := h.useCase.UpdateAutomation(r.Context(), id, &a)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) TriggerTestAutomation(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.TriggerTestAutomation(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
