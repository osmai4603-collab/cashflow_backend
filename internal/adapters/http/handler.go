package httpadapter

import (
	"log/slog"
	"net/http"

	"cashflow_backend/internal/platform/response"
)

// BaseHandler provides shared HTTP handling capabilities and the root service discovery endpoint.
type BaseHandler struct {
	serviceName string
	version     string
	logger      *slog.Logger
}

// NewBaseHandler constructs a new BaseHandler.
func NewBaseHandler(serviceName, version string, logger *slog.Logger) *BaseHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &BaseHandler{
		serviceName: serviceName,
		version:     version,
		logger:      logger,
	}
}

// Root handles GET / returning service metadata and status.
func (h *BaseHandler) Root(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, ServiceInfoResponse{
		Service: h.serviceName,
		Version: h.version,
		Status:  "healthy",
	})
}
