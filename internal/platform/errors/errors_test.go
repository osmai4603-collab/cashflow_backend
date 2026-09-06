package errors_test

import (
	"errors"
	"net/http"
	"testing"

	platformerrors "cashflow_backend/internal/platform/errors"
)

func TestAppError_Formatting(t *testing.T) {
	origErr := errors.New("underlying db error")
	appErr := platformerrors.New(platformerrors.CodeInternal, "failed to query database", origErr)

	expected := "[INTERNAL_ERROR] failed to query database: underlying db error"
	if appErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, appErr.Error())
	}

	if !errors.Is(appErr, origErr) {
		t.Errorf("expected errors.Is to match original error")
	}
}

func TestAppError_Constructors(t *testing.T) {
	tests := []struct {
		name       string
		err        *platformerrors.AppError
		wantCode   string
		wantStatus int
	}{
		{
			name:       "not found",
			err:        platformerrors.NotFound("partner not found"),
			wantCode:   platformerrors.CodeNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "validation",
			err:        platformerrors.Validation("invalid input", map[string]string{"name": "required"}),
			wantCode:   platformerrors.CodeValidation,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "conflict",
			err:        platformerrors.Conflict("email already exists"),
			wantCode:   platformerrors.CodeConflict,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "unauthorized",
			err:        platformerrors.Unauthorized("token expired"),
			wantCode:   platformerrors.CodeUnauthorized,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "forbidden",
			err:        platformerrors.Forbidden("permission denied"),
			wantCode:   platformerrors.CodeForbidden,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "internal",
			err:        platformerrors.Internal("unexpected system failure"),
			wantCode:   platformerrors.CodeInternal,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "bad request",
			err:        platformerrors.BadRequest("invalid json payload"),
			wantCode:   platformerrors.CodeBadRequest,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("got code %q, want %q", tt.err.Code, tt.wantCode)
			}
			status := platformerrors.HTTPStatus(tt.err)
			if status != tt.wantStatus {
				t.Errorf("got http status %d, want %d", status, tt.wantStatus)
			}
		})
	}
}

func TestHTTPStatus_Fallback(t *testing.T) {
	if status := platformerrors.HTTPStatus(nil); status != http.StatusOK {
		t.Errorf("expected 200 for nil error, got %d", status)
	}

	stdErr := errors.New("standard error")
	if status := platformerrors.HTTPStatus(stdErr); status != http.StatusInternalServerError {
		t.Errorf("expected 500 for standard error, got %d", status)
	}
}
