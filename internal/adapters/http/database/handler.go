package database

import (
	"net/http"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
)

type Handler struct {
	databaseName string
}

func NewHandler(databaseName string) *Handler {
	return &Handler{databaseName: databaseName}
}

func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{
		"databases": []string{h.databaseName},
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	response.Error(w, r, platformerrors.BadRequest(
		"cashflow_backend uses the configured cashflow database; database creation is managed outside this service",
	))
	return

}
