package websitehttp

import (
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/platform/response"
	websiteusecase "cashflow_backend/internal/usecase/website"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *websiteusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *websiteusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) GetSite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	site, err := h.useCase.GetSiteByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, site)
}

func (h *Handler) GetPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	domain := r.Host
	page, err := h.useCase.RenderPage(r.Context(), domain, slug)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, page)
}

func (h *Handler) GetMenus(w http.ResponseWriter, r *http.Request) {
	domain := r.Host
	menus, err := h.useCase.GetNavigation(r.Context(), domain)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, menus)
}
