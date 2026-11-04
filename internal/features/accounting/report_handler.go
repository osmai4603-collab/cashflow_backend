package accounting

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ReportHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewReportHandler(uc *UseCases, logger *slog.Logger) *ReportHandler {
	return &ReportHandler{uc: uc, logger: handlerLogger(logger, "report")}
}

func (h *ReportHandler) HandleTrialBalance(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseReportRange(w, r, h.logger)
	if !ok {
		return
	}

	response, err := h.uc.GenerateTrialBalance.Execute(r.Context(), from, to)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *ReportHandler) HandleIncomeStatement(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseReportRange(w, r, h.logger)
	if !ok {
		return
	}
	response, err := h.uc.GenerateIncomeStatement.Execute(r.Context(), from, to)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *ReportHandler) HandleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	value := r.URL.Query().Get("as_of")
	asOf, err := time.Parse(dateLayout, value)
	if err != nil || asOf.Format(dateLayout) != value {
		writeBadRequest(w, r, h.logger, errors.New("as_of must be in YYYY-MM-DD format"))
		return
	}
	response, err := h.uc.GenerateBalanceSheet.Execute(r.Context(), asOf)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *ReportHandler) HandleCashFlow(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseReportRange(w, r, h.logger)
	if !ok {
		return
	}
	response, err := h.uc.GenerateCashFlow.Execute(r.Context(), from, to)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *ReportHandler) HandleGeneralLedger(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(r.URL.Query().Get("account_id"))
	if err != nil {
		writeBadRequest(w, r, h.logger, errors.New("account_id must be a UUID"))
		return
	}
	from, to, ok := parseReportRange(w, r, h.logger)
	if !ok {
		return
	}
	response, err := h.uc.GenerateGeneralLedger.Execute(r.Context(), accountID, from, to)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func parseReportRange(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (time.Time, time.Time, bool) {
	fromValue, toValue := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	from, err := time.Parse(dateLayout, fromValue)
	if err != nil || from.Format(dateLayout) != fromValue {
		writeBadRequest(w, r, logger, errors.New("from must be in YYYY-MM-DD format"))
		return time.Time{}, time.Time{}, false
	}
	to, err := time.Parse(dateLayout, toValue)
	if err != nil || to.Format(dateLayout) != toValue {
		writeBadRequest(w, r, logger, errors.New("to must be in YYYY-MM-DD format"))
		return time.Time{}, time.Time{}, false
	}
	if from.After(to) {
		writeBadRequest(w, r, logger, errors.New("from must not be after to"))
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}
