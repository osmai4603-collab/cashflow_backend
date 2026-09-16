package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5/middleware"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// Standard Response Envelopes
type Envelope struct {
	Success bool              `json:"success"`
	Data    any               `json:"data,omitempty"`
	Meta    any               `json:"meta,omitempty"`
	Error   *ErrorPayload     `json:"error,omitempty"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// logger is used to emit structured records for server-side errors so the root
// cause (message + details) never disappears from the logs. It is overridable
// for tests via SetLogger.
var logger = slog.Default()

// SetLogger replaces the internal logger used by Error for 5xx diagnostics.
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}

// maxTrackedErrors bounds the in-flight request error capture so an
// unconventional handler that never goes through the outer access-log
// middleware cannot grow memory without limit.
const maxTrackedErrors = 1024

// requestErrors correlates the error handled by response.Error with the outer
// access-log middleware so both records share the same request_id.
var requestErrors = struct {
	mu sync.Mutex
	m  map[*http.Request]error
}{m: make(map[*http.Request]error)}

func trackError(r *http.Request, err error) {
	if r == nil || err == nil {
		return
	}
	requestErrors.mu.Lock()
	defer requestErrors.mu.Unlock()
	if len(requestErrors.m) >= maxTrackedErrors {
		return
	}
	requestErrors.m[r] = err
}

// TakeError fetches and removes the error captured for an in-flight request.
// It is used by the access-log middleware to enrich 5xx records with the root
// cause. It returns nil when no error was tracked for the request.
func TakeError(r *http.Request) error {
	if r == nil {
		return nil
	}
	requestErrors.mu.Lock()
	defer requestErrors.mu.Unlock()
	err, ok := requestErrors.m[r]
	if ok {
		delete(requestErrors.m, r)
	}
	return err
}

// JSON sends a JSON response with the provided status code and data wrapped in Envelope.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: true,
		Data:    data,
	})
}

// Paginated sends a JSON response with data and pagination metadata.
func Paginated(w http.ResponseWriter, status int, data any, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created sends a 201 Created response with the resource data.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// NoContent sends a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error responds with a standardized error envelope mapping AppError to appropriate HTTP status.
// The variadic form keeps compatibility with handlers that pass either (writer, err)
// or (writer, request, err) while the response translation migration is completed.
func Error(w http.ResponseWriter, args ...any) {
	var req *http.Request
	var err error
	for _, arg := range args {
		switch v := arg.(type) {
		case *http.Request:
			req = v
		case error:
			err = v
		}
	}
	if err == nil {
		err = platformerrors.Internal("internal server error", nil)
	}
	status := platformerrors.HTTPStatus(err)
	translate := func(msg string) string {
		if req == nil {
			return msg
		}
		return i18n.T(req.Context(), msg)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var appErr *platformerrors.AppError
	if ok := isAppError(err, &appErr); ok {
		if status >= 500 && req != nil {
			// Keep the outer access-log middleware correlated with the root
			// cause by the same request_id.
			trackError(req, err)
		}
		logServerError(req, status, appErr, err)
		_ = json.NewEncoder(w).Encode(Envelope{
			Success: false,
			Error: &ErrorPayload{
				Code:    appErr.Code,
				Message: translate(appErr.Message),
				Details: appErr.Details,
			},
		})
		return
	}

	logServerError(req, status, nil, err)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: false,
		Error: &ErrorPayload{
			Code:    platformerrors.CodeInternal,
			Message: translate("internal server error"),
		},
	})
}

// logServerError records server-side error responses (>= 500) with the root
// cause, including the wrapped database error and attached details, so the
// reason is always correlated with the request_id in the access log.
func logServerError(req *http.Request, status int, appErr *platformerrors.AppError, err error) {
	if status < 500 {
		return
	}

	attrs := []any{"status", status}
	if req != nil {
		attrs = append(attrs,
			"method", req.Method,
			"path", req.URL.Path,
			"request_id", middleware.GetReqID(req.Context()),
		)
	}
	if appErr != nil {
		attrs = append(attrs, "code", appErr.Code, "message", appErr.Message)
		if appErr.Err != nil {
			attrs = append(attrs, "error", appErr.Err.Error())
		} else if err != nil {
			attrs = append(attrs, "error", err.Error())
		}
		if appErr.Details != nil {
			attrs = append(attrs, "details", appErr.Details)
		}
	} else if err != nil {
		attrs = append(attrs, "code", platformerrors.CodeInternal, "error", err.Error())
	}
	logger.Error("internal server error response", attrs...)
}

func isAppError(err error, target **platformerrors.AppError) bool {
	var appErr *platformerrors.AppError
	if errorsAs(err, &appErr) {
		*target = appErr
		return true
	}
	return false
}

// errorsAs wrapper for error unwrapping
var errorsAs = func(err error, target any) bool {
	return platformerrorsAs(err, target)
}

func platformerrorsAs(err error, target any) bool {
	type unwrapper interface {
		Unwrap() error
	}

	for err != nil {
		if t, ok := target.(**platformerrors.AppError); ok {
			if e, ok := err.(*platformerrors.AppError); ok {
				*t = e
				return true
			}
		}
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return false
}
