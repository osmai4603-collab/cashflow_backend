package reporthttp

import (
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/report"
	reportusecase "cashflow_backend/internal/usecase/report"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	generator report.ReportGenerator
	dashboard *reportusecase.DashboardUseCase
}

func NewHandler(generator report.ReportGenerator, dashboard *reportusecase.DashboardUseCase) *Handler {
	return &Handler{generator: generator, dashboard: dashboard}
}

func (h *Handler) GetReport(c *gin.Context) {
	code := c.Param("code")
	companyID, _ := strconv.ParseInt(c.Query("company_id"), 10, 64)
	if companyID == 0 {
		// Fallback or error
	}

	opts := report.ReportOptions{
		CompanyID: companyID,
	}

	if from := c.Query("date_from"); from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err == nil {
			opts.DateFrom = &t
		}
	}
	if to := c.Query("date_to"); to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err == nil {
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	res, err := h.generator.Generate(c.Request.Context(), config, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetDashboard(c *gin.Context) {
	companyID, _ := strconv.ParseInt(c.Query("company_id"), 10, 64)
	res, err := h.dashboard.GetOverviewDashboard(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
