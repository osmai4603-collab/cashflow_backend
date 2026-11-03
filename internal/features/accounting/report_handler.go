package accounting

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type ReportHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewReportHandler(uc *UseCases, logger *slog.Logger) *ReportHandler {
	return &ReportHandler{uc: uc, logger: handlerLogger(logger, "report")}
}

func (h *ReportHandler) HandleTrialBalance(w http.ResponseWriter, r *http.Request) {
	from, err := time.Parse(dateLayout, r.URL.Query().Get("from"))
	if err != nil {
		writeBadRequest(w, r, h.logger, errors.New("from must be in YYYY-MM-DD format"))
		return
	}
	to, err := time.Parse(dateLayout, r.URL.Query().Get("to"))
	if err != nil {
		writeBadRequest(w, r, h.logger, errors.New("to must be in YYYY-MM-DD format"))
		return
	}
	if from.Format(dateLayout) != r.URL.Query().Get("from") || to.Format(dateLayout) != r.URL.Query().Get("to") {
		writeBadRequest(w, r, h.logger, errors.New("report dates must be in YYYY-MM-DD format"))
		return
	}
	if from.After(to) {
		writeBadRequest(w, r, h.logger, errors.New("from must not be after to"))
		return
	}

	response, err := h.uc.GenerateTrialBalance.Execute(r.Context(), from, to)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
