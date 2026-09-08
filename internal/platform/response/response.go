package response

import (
	"encoding/json"
	"net/http"

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

	_ = json.NewEncoder(w).Encode(Envelope{
		Success: false,
		Error: &ErrorPayload{
			Code:    platformerrors.CodeInternal,
			Message: translate("internal server error"),
		},
	})
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
