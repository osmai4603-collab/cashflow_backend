package apidesign

import (
	"encoding/json"
	"net/http"
)

// =============================================================================
// RFC 7807 Problem Details Error Handler (application/problem+json)
// =============================================================================

type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type ProblemDetails struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	Code          string         `json:"code,omitempty"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
}

// WriteProblem writes an RFC 7807 JSON error response.
func WriteProblem(w http.ResponseWriter, p ProblemDetails) {
	if p.Type == "" {
		p.Type = "about:blank"
	}
	if p.Status == 0 {
		p.Status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// NotFoundProblem creates a 404 Problem Details response.
func NotFoundProblem(detail, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "urn:problem:not-found",
		Title:    "Resource Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: instance,
		Code:     "RESOURCE_NOT_FOUND",
	}
}

// ValidationProblem creates a 422 Unprocessable Entity Problem Details response.
func ValidationProblem(detail, instance string, params []InvalidParam) ProblemDetails {
	return ProblemDetails{
		Type:          "urn:problem:validation-error",
		Title:         "Validation Failed",
		Status:        http.StatusUnprocessableEntity,
		Detail:        detail,
		Instance:      instance,
		Code:          "VALIDATION_ERROR",
		InvalidParams: params,
	}
}
