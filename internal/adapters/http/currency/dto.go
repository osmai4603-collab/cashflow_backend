package currencyhttp

import (
	"time"

	"cashflow_backend/internal/domain/currency"
	currencyusecase "cashflow_backend/internal/usecase/currency"
)

// CreateCurrencyRequest represents the incoming JSON payload to register a new currency.
type CreateCurrencyRequest struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Symbol        string `json:"symbol"`
	DecimalPlaces int16  `json:"decimal_places"`
}

// ToInput maps the HTTP request DTO to the application usecase input.
func (r CreateCurrencyRequest) ToInput() currencyusecase.CreateCurrencyInput {
	return currencyusecase.CreateCurrencyInput{
		Name:          r.Name,
		FullName:      r.FullName,
		Symbol:        r.Symbol,
		DecimalPlaces: r.DecimalPlaces,
	}
}

// UpdateCurrencyRequest represents partial or full fields to update an existing currency.
type UpdateCurrencyRequest struct {
	Name          *string `json:"name"`
	FullName      *string `json:"full_name"`
	Symbol        *string `json:"symbol"`
	DecimalPlaces *int16  `json:"decimal_places"`
}

// ToInput maps the update request DTO to the application usecase input.
func (r UpdateCurrencyRequest) ToInput() currencyusecase.UpdateCurrencyInput {
	return currencyusecase.UpdateCurrencyInput{
		Name:          r.Name,
		FullName:      r.FullName,
		Symbol:        r.Symbol,
		DecimalPlaces: r.DecimalPlaces,
	}
}

// CreateRateRequest represents the incoming JSON payload to register an exchange rate.
type CreateRateRequest struct {
	CurrencyID int64   `json:"currency_id"`
	Rate       float64 `json:"rate"`
	Date       string  `json:"date"`
	CompanyID  *int64  `json:"company_id"`
}

// ToRateInput maps the HTTP request DTO to the application usecase input.
func (r CreateRateRequest) ToRateInput() currencyusecase.CreateRateInput {
	return currencyusecase.CreateRateInput{
		CurrencyID: r.CurrencyID,
		Rate:       r.Rate,
		Date:       r.Date,
		CompanyID:  r.CompanyID,
	}
}

// ConvertRequest represents the incoming JSON payload for a currency conversion.
type ConvertRequest struct {
	Amount       float64 `json:"amount"`
	FromCurrency int64   `json:"from_currency_id"`
	ToCurrency   int64   `json:"to_currency_id"`
	Date         string  `json:"date"`
}

// ToConvertInput maps the HTTP request DTO to the application conversion input.
func (r ConvertRequest) ToConvertInput() currencyusecase.ConvertInput {
	return currencyusecase.ConvertInput{
		Amount:       r.Amount,
		FromCurrency: r.FromCurrency,
		ToCurrency:   r.ToCurrency,
		Date:         r.Date,
	}
}

// CurrencyResponse formats currency data for HTTP JSON client output.
type CurrencyResponse struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Symbol        string    `json:"symbol"`
	DecimalPlaces int16     `json:"decimal_places"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToCurrencyResponse converts a domain Currency entity into CurrencyResponse DTO.
func ToCurrencyResponse(c *currency.Currency) CurrencyResponse {
	if c == nil {
		return CurrencyResponse{}
	}
	return CurrencyResponse{
		ID:            c.ID,
		Name:          c.Name,
		FullName:      c.FullName,
		Symbol:        c.Symbol,
		DecimalPlaces: c.DecimalPlaces,
		Active:        c.Active,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// ToCurrencyResponseList converts a slice of domain currencies into response DTOs.
func ToCurrencyResponseList(items []currency.Currency) []CurrencyResponse {
	result := make([]CurrencyResponse, len(items))
	for i, item := range items {
		result[i] = ToCurrencyResponse(&item)
	}
	return result
}

// CurrencyRateResponse formats currency rate data for HTTP JSON client output.
type CurrencyRateResponse struct {
	ID         int64   `json:"id"`
	CurrencyID int64   `json:"currency_id"`
	Rate       float64 `json:"rate"`
	Date       string  `json:"date"`
	CompanyID  *int64  `json:"company_id,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// ToCurrencyRateResponse converts a domain CurrencyRate entity into CurrencyRateResponse DTO.
func ToCurrencyRateResponse(r *currency.CurrencyRate) CurrencyRateResponse {
	if r == nil {
		return CurrencyRateResponse{}
	}
	return CurrencyRateResponse{
		ID:         r.ID,
		CurrencyID: r.CurrencyID,
		Rate:       r.Rate,
		Date:       r.Date,
		CompanyID:  r.CompanyID,
		CreatedAt:  r.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  r.Audit.UpdatedAt.Format(time.RFC3339),
	}
}

// ToCurrencyRateResponseList converts a slice of domain currency rates into response DTOs.
func ToCurrencyRateResponseList(items []currency.CurrencyRate) []CurrencyRateResponse {
	result := make([]CurrencyRateResponse, len(items))
	for i, item := range items {
		result[i] = ToCurrencyRateResponse(&item)
	}
	return result
}
