package sequencehttp

import (
	"time"

	"cashflow_backend/internal/domain/sequence"
	sequenceusecase "cashflow_backend/internal/usecase/sequence"
)

// CreateSequenceRequest represents the incoming JSON payload to register a new sequence.
type CreateSequenceRequest struct {
	Name         string                        `json:"name"`
	Code         string                        `json:"code"`
	Prefix       string                        `json:"prefix"`
	Suffix       string                        `json:"suffix"`
	Padding      int16                         `json:"padding"`
	IncrementBy  int                           `json:"increment_by"`
	StartNumber  int                           `json:"start_number"`
	SequenceType sequence.SequenceType         `json:"sequence_type"`
	DateRange    sequence.DateRangeGranularity `json:"date_range"`
	CompanyID    *int64                        `json:"company_id"`
}

// ToInput maps the HTTP request DTO to the application usecase input.
func (r CreateSequenceRequest) ToInput() sequenceusecase.CreateSequenceInput {
	return sequenceusecase.CreateSequenceInput{
		Name:         r.Name,
		Code:         r.Code,
		Prefix:       r.Prefix,
		Suffix:       r.Suffix,
		Padding:      r.Padding,
		IncrementBy:  r.IncrementBy,
		StartNumber:  r.StartNumber,
		SequenceType: r.SequenceType,
		DateRange:    r.DateRange,
		CompanyID:    r.CompanyID,
	}
}

// UpdateSequenceRequest represents partial or full fields to update an existing sequence.
type UpdateSequenceRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Prefix      *string `json:"prefix"`
	Suffix      *string `json:"suffix"`
	Padding     *int16  `json:"padding"`
	IncrementBy *int    `json:"increment_by"`
	StartNumber *int    `json:"start_number"`
	Active      *bool   `json:"active"`
}

// ToInput maps the update request DTO to the application usecase input.
func (r UpdateSequenceRequest) ToInput() sequenceusecase.UpdateSequenceInput {
	return sequenceusecase.UpdateSequenceInput{
		Name:        r.Name,
		Code:        r.Code,
		Prefix:      r.Prefix,
		Suffix:      r.Suffix,
		Padding:     r.Padding,
		IncrementBy: r.IncrementBy,
		StartNumber: r.StartNumber,
		Active:      r.Active,
	}
}

// NextNumberRequest is empty for a POST next-number call; date is optional.
type NextNumberRequest struct {
	Date string `json:"date"`
}

// SequenceResponse formats sequence data for HTTP JSON client output.
type SequenceResponse struct {
	ID            int64                         `json:"id"`
	Name          string                        `json:"name"`
	Code          string                        `json:"code"`
	Prefix        string                        `json:"prefix,omitempty"`
	Suffix        string                        `json:"suffix,omitempty"`
	Padding       int16                         `json:"padding"`
	IncrementBy   int                           `json:"increment_by"`
	StartNumber   int                           `json:"start_number"`
	CurrentNumber int                           `json:"current_number"`
	SequenceType  sequence.SequenceType         `json:"sequence_type"`
	DateRange     sequence.DateRangeGranularity `json:"date_range,omitempty"`
	CompanyID     *int64                        `json:"company_id,omitempty"`
	Active        bool                          `json:"active"`
	CreatedAt     time.Time                     `json:"created_at"`
	UpdatedAt     time.Time                     `json:"updated_at"`
}

// NextNumberResponse formats an atomically generated document reference.
type NextNumberResponse struct {
	SequenceID int64  `json:"sequence_id"`
	Code       string `json:"code"`
	Number     int    `json:"number"`
	Reference  string `json:"reference"`
}

// ToSequenceResponse converts a domain Sequence entity into SequenceResponse DTO.
func ToSequenceResponse(s *sequence.Sequence) SequenceResponse {
	if s == nil {
		return SequenceResponse{}
	}
	return SequenceResponse{
		ID:            s.ID,
		Name:          s.Name,
		Code:          s.Code,
		Prefix:        s.Prefix,
		Suffix:        s.Suffix,
		Padding:       s.Padding,
		IncrementBy:   s.IncrementBy,
		StartNumber:   s.StartNumber,
		CurrentNumber: s.CurrentNumber,
		SequenceType:  s.SequenceType,
		DateRange:     s.DateRange,
		CompanyID:     s.CompanyID,
		Active:        s.Active,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

// ToSequenceResponseList converts a slice of domain sequences into response DTOs.
func ToSequenceResponseList(items []sequence.Sequence) []SequenceResponse {
	result := make([]SequenceResponse, len(items))
	for i, item := range items {
		result[i] = ToSequenceResponse(&item)
	}
	return result
}
