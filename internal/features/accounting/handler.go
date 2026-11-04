package accounting

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	platformctx "cashflow_backend/internal/platform/context"
	platformerr "cashflow_backend/internal/platform/errors"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func RegisterRoutes(r chi.Router, uc *UseCases, logger *slog.Logger) {
	if uc == nil {
		registerUnavailableRoutes(r, logger)
		return
	}

	accountHandler := NewAccountHandler(uc, logger)
	periodHandler := NewPeriodHandler(uc, logger)
	moveHandler := NewMoveHandler(uc, logger)
	reportHandler := NewReportHandler(uc, logger)
	subledgerHandler := NewSubledgerHandler(uc, logger)
	reconciliationHandler := NewReconciliationHandler(uc, logger)

	r.Route("/api/v1/accounting", func(sub chi.Router) {
		sub.Post("/accounts", accountHandler.HandleCreate)
		sub.Get("/accounts", accountHandler.HandleList)
		sub.Get("/accounts/tree", accountHandler.HandleTree)
		sub.Put("/accounts/{id}", accountHandler.HandleUpdate)

		sub.Post("/fiscal-years", periodHandler.HandleCreateYear)
		sub.Get("/fiscal-years", periodHandler.HandleListYears)
		sub.Post("/fiscal-years/{id}/close", periodHandler.HandleCloseYear)
		sub.Post("/fiscal-periods/{id}/state", periodHandler.HandleChangeState)
		sub.Get("/journals", periodHandler.HandleListJournals)

		sub.Post("/moves", moveHandler.HandleCreateDraft)
		sub.Get("/moves", moveHandler.HandleList)
		sub.Get("/moves/{id}", moveHandler.HandleGet)
		sub.Put("/moves/{id}", moveHandler.HandleUpdateDraft)
		sub.Delete("/moves/{id}", moveHandler.HandleDeleteDraft)
		sub.Post("/moves/{id}/post", moveHandler.HandlePost)
		sub.Post("/moves/{id}/reverse", moveHandler.HandleReverse)

		sub.Get("/reports/trial-balance", reportHandler.HandleTrialBalance)
		sub.Get("/reports/income-statement", reportHandler.HandleIncomeStatement)
		sub.Get("/reports/balance-sheet", reportHandler.HandleBalanceSheet)
		sub.Get("/reports/cash-flow", reportHandler.HandleCashFlow)
		sub.Get("/reports/general-ledger", reportHandler.HandleGeneralLedger)
		sub.Post("/subledger/events", subledgerHandler.HandleIngest)
		sub.Post("/reconcile", reconciliationHandler.HandleReconcile)
		sub.Post("/unreconcile", reconciliationHandler.HandleUnreconcile)
		sub.Get("/unreconciled-lines", reconciliationHandler.HandleListUnreconciled)
	})
}

func registerUnavailableRoutes(r chi.Router, logger *slog.Logger) {
	unavailable := func(w http.ResponseWriter, r *http.Request) {
		err := platformerr.Unavailable("accounting", "accounting storage is not configured", nil)
		platformerr.WriteHTTPError(w, r, err, logger)
	}
	r.Route("/api/v1/accounting", func(sub chi.Router) {
		for _, route := range []struct {
			method string
			path   string
		}{
			{http.MethodPost, "/accounts"},
			{http.MethodGet, "/accounts"},
			{http.MethodGet, "/accounts/tree"},
			{http.MethodPut, "/accounts/{id}"},
			{http.MethodPost, "/fiscal-years"},
			{http.MethodGet, "/fiscal-years"},
			{http.MethodPost, "/fiscal-years/{id}/close"},
			{http.MethodPost, "/fiscal-periods/{id}/state"},
			{http.MethodGet, "/journals"},
			{http.MethodPost, "/moves"},
			{http.MethodGet, "/moves"},
			{http.MethodGet, "/moves/{id}"},
			{http.MethodPut, "/moves/{id}"},
			{http.MethodDelete, "/moves/{id}"},
			{http.MethodPost, "/moves/{id}/post"},
			{http.MethodPost, "/moves/{id}/reverse"},
			{http.MethodGet, "/reports/trial-balance"},
			{http.MethodGet, "/reports/income-statement"},
			{http.MethodGet, "/reports/balance-sheet"},
			{http.MethodGet, "/reports/cash-flow"},
			{http.MethodGet, "/reports/general-ledger"},
			{http.MethodPost, "/subledger/events"},
			{http.MethodPost, "/reconcile"},
			{http.MethodPost, "/unreconcile"},
			{http.MethodGet, "/unreconciled-lines"},
		} {
			sub.Method(route.method, route.path, http.HandlerFunc(unavailable))
		}
	})
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request contains multiple JSON values")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeBadRequest(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	appErr := platformerr.Invalid("accounting.request", "invalid request payload", err)
	platformerr.WriteHTTPError(w, r, appErr, logger)
}

func writeValidationError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	appErr := platformerr.Invalid("accounting.validation", err.Error(), err)
	platformerr.WriteHTTPErrorWithStatus(w, r, appErr, logger, http.StatusUnprocessableEntity)
}

func writeAccountingError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, ErrAccountNotFound), errors.Is(err, ErrPeriodNotFound),
		errors.Is(err, ErrJournalNotFound), errors.Is(err, ErrMoveNotFound):
		err = platformerr.NotFound("accounting", "requested accounting resource was not found", err)
	case errors.Is(err, ErrDuplicateAccountCode), errors.Is(err, ErrPeriodClosed),
		errors.Is(err, ErrMoveAlreadyPosted), errors.Is(err, ErrMoveAlreadyReversed),
		errors.Is(err, ErrMoveChanged),
		errors.Is(err, ErrCannotReverseUnpostedMove), errors.Is(err, ErrInvalidPeriodTransition),
		errors.Is(err, ErrMakerCheckerViolation), errors.Is(err, ErrMakerCheckerCreatorRequired),
		errors.Is(err, ErrAccountHasPostedMoves), errors.Is(err, ErrSubledgerMappingNotFound),
		errors.Is(err, ErrExchangeRateMissing), errors.Is(err, ErrAccountingConfigurationMissing),
		errors.Is(err, ErrAccountNotReconcilable), errors.Is(err, ErrFiscalYearNotClosable):
		err = platformerr.Conflict("accounting", "accounting resource state conflicts with this operation", err)
	case errors.Is(err, ErrReconciliationNotFound), errors.Is(err, ErrFiscalYearNotFound):
		err = platformerr.NotFound("accounting", "requested accounting resource was not found", err)
	case errors.Is(err, ErrMakerCheckerActorRequired):
		err = platformerr.Unauthorized("accounting.actor", "authenticated actor is required", err)
	case errors.Is(err, ErrAccountingActorRequired):
		err = platformerr.Unauthorized("accounting.actor", "authenticated actor is required", err)
	case errors.Is(err, ErrUnbalancedMove), errors.Is(err, ErrNegativeAmountsNotAllowed),
		errors.Is(err, ErrZeroAmountLine), errors.Is(err, ErrMoveRequiresAtLeastTwoLines):
		err = platformerr.Invalid("accounting", "accounting request violates domain rules", err)
		platformerr.WriteHTTPErrorWithStatus(w, r, err, logger, http.StatusUnprocessableEntity)
		return
	default:
		var appErr *platformerr.Error
		if !errors.As(err, &appErr) {
			err = platformerr.Internal("accounting", "an internal error occurred", err)
		}
	}
	platformerr.WriteHTTPError(w, r, err, logger)
}

func parseRouteUUID(r *http.Request, parameter string) (uuid.UUID, error) {
	value := chi.URLParam(r, parameter)
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", parameter)
	}
	return id, nil
}

func actorIDFromRequest(r *http.Request) (uuid.UUID, error) {
	userID, ok := platformctx.UserIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, nil
	}
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid authenticated user id")
	}
	return actorID, nil
}

func handlerLogger(logger *slog.Logger, name string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logger.With(slog.String("handler", name))
}
