package reporthttp

import (
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/report"
	"cashflow_backend/internal/platform/response"
	reportusecase "cashflow_backend/internal/usecase/report"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	generator report.ReportGenerator
	dashboard *reportusecase.DashboardUseCase
}

func NewHandler(generator report.ReportGenerator, dashboard *reportusecase.DashboardUseCase) *Handler {
	return &Handler{generator: generator, dashboard: dashboard}
}

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	companyID, _ := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)

	opts := report.ReportOptions{
		CompanyID: companyID,
	}

	if from := r.URL.Query().Get("date_from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			opts.DateFrom = &t
		}
	}
	if to := r.URL.Query().Get("date_to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			opts.DateTo = &t
		}
	}

	var config *report.Report
	switch code {
	case "PL":
		config = reportusecase.GetProfitAndLossConfig()
	case "BS":
		config = reportusecase.GetBalanceSheetConfig()
	case "TB":
		config = reportusecase.GetTrialBalanceConfig()
	case "INV_VAL":
		config = reportusecase.GetInventoryValuationConfig()
	case "SALE_ANALYSIS":
		config = reportusecase.GetSalesAnalysisConfig()
	default:
		response.JSON(w, http.StatusNotFound, map[string]string{"error": "Report not found"})
		return
	}

	res, err := h.generator.Generate(r.Context(), config, opts)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	companyID, _ := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	res, err := h.dashboard.GetOverviewDashboard(r.Context(), companyID)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, res)
}