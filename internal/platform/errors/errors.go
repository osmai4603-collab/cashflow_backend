package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard Error Codes
const (
	CodeNotFound     = "NOT_FOUND"
	CodeValidation   = "VALIDATION_ERROR"
	CodeConflict     = "CONFLICT"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeInternal     = "INTERNAL_ERROR"
	CodeBadRequest   = "BAD_REQUEST"
	CodeBadGateway   = "BAD_GATEWAY"
)

// AppError represents a structured, domain-level application error.
type AppError struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	TranslationKey string `json:"-"`
	Details        any    `json:"details,omitempty"`
	Err            error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Helper Constructors
func New(code, message string, err error) *AppError {
	return &AppError{
		Code:           code,
		Message:        message,
		TranslationKey: message,
		Err:            err,
	}
}

func NotFound(message string, err ...error) *AppError {
	var original error
	if len(err) > 0 {
		original = err[0]
	}
	return &AppError{
		Code:           CodeNotFound,
		Message:        message,
		TranslationKey: message,
		Err:            original,
	}
}

func Validation(message string, details any) *AppError {
	return &AppError{
		Code:           CodeValidation,
		Message:        message,
		TranslationKey: message,
		Details:        details,
	}
}

func Conflict(message string, err ...error) *AppError {
	var original error
	if len(err) > 0 {
		original = err[0]
	}
	return &AppError{
		Code:    CodeConflict,
		Message: message,
		Err:     original,
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{
		Code:    CodeUnauthorized,
		Message: message,
	}
}

func Forbidden(message string) *AppError {
	return &AppError{
		Code:    CodeForbidden,
		Message: message,
	}
}

func BadRequest(message string, err ...error) *AppError {
	var original error
	if len(err) > 0 {
		original = err[0]
	}
	return &AppError{
		Code:    CodeBadRequest,
		Message: message,
		Err:     original,
	}
}

func Internal(message string, err ...error) *AppError {
	var original error
	if len(err) > 0 {
		original = err[0]
	}
	return &AppError{
		Code:    CodeInternal,
		Message: message,
		Err:     original,
	}
}

func BadGateway(message string, err ...error) *AppError {
	var original error
	if len(err) > 0 {
		original = err[0]
	}
	return &AppError{
		Code:    CodeBadGateway,
		Message: message,
		Err:     original,
	}
}

// HTTPStatus maps an error to the corresponding HTTP status code.
func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case CodeNotFound:
			return http.StatusNotFound
		case CodeValidation, CodeBadRequest:
			return http.StatusBadRequest
		case CodeConflict:
			return http.StatusConflict
		case CodeUnauthorized:
			return http.StatusUnauthorized
		case CodeForbidden:
			return http.StatusForbidden
		case CodeBadGateway:
			return http.StatusBadGateway
		case CodeInternal:
			return http.StatusInternalServerError
		default:
			return http.StatusInternalServerError
		}
	}

	return http.StatusInternalServerError
}
