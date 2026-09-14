package apidesign

import (
	"encoding/json"
	"net/http"
	"time"
)

// =============================================================================
// Unified JSON Response Envelope Builder
// =============================================================================

// SuccessResponse defines the standard success envelope.
type SuccessResponse struct {
	Data interface{} `json:"data"`
	Meta *MetaInfo   `json:"meta,omitempty"`
}

type MetaInfo struct {
	RequestID    string `json:"request_id,omitempty"`
	Timestamp    string `json:"timestamp"`
	Page         int    `json:"page,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	TotalRecords int64  `json:"total_records,omitempty"`
	TotalPages   int    `json:"total_pages,omitempty"`
}

// RespondJSON writes a standardized 200/201 JSON envelope.
func RespondJSON(w http.ResponseWriter, status int, data interface{}, meta *MetaInfo) {
	if meta == nil {
		meta = &MetaInfo{}
	}
	if meta.Timestamp == "" {
		meta.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	payload := SuccessResponse{
		Data: data,
		Meta: meta,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// NewPaginationMeta calculates total pages and formats pagination metadata safely.
func NewPaginationMeta(page, limit int, totalRecords int64, requestID string) *MetaInfo {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Enforce max bounds
	}

	totalPages := int((totalRecords + int64(limit) - 1) / int64(limit))

	return &MetaInfo{
		RequestID:    requestID,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Page:         page,
		Limit:        limit,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
	}
}
