package companyhttp

import (
	"time"

	"cashflow_backend/internal/domain/company"
	companyusecase "cashflow_backend/internal/usecase/company"
)

// CreateCompanyRequest represents the incoming JSON payload to register a new company.
type CreateCompanyRequest struct {
	Name       string `json:"name"`
	PartnerID  *int64 `json:"partner_id"`
	CurrencyID int64  `json:"currency_id"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Website    string `json:"website"`
	VAT        string `json:"vat"`
	Street     string `json:"street"`
	Street2    string `json:"street2"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	ZipCode    string `json:"zip_code"`
}

// ToInput maps the HTTP request DTO to the application usecase input.
func (r CreateCompanyRequest) ToInput() companyusecase.CreateCompanyInput {
	return companyusecase.CreateCompanyInput{
		Name:       r.Name,
		PartnerID:  r.PartnerID,
		CurrencyID: r.CurrencyID,
		Phone:      r.Phone,
		Email:      r.Email,
		Website:    r.Website,
		VAT:        r.VAT,
		Street:     r.Street,
		Street2:    r.Street2,
		City:       r.City,
		State:      r.State,
		Country:    r.Country,
		ZipCode:    r.ZipCode,
	}
}

// UpdateCompanyRequest represents partial or full fields to update an existing company.
type UpdateCompanyRequest struct {
	Name       *string `json:"name"`
	PartnerID  *int64  `json:"partner_id"`
	CurrencyID *int64  `json:"currency_id"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Website    *string `json:"website"`
	VAT        *string `json:"vat"`
	Street     *string `json:"street"`
	Street2    *string `json:"street2"`
	City       *string `json:"city"`
	State      *string `json:"state"`
	Country    *string `json:"country"`
	ZipCode    *string `json:"zip_code"`
}

// ToInput maps the update request DTO to the application usecase input.
func (r UpdateCompanyRequest) ToInput() companyusecase.UpdateCompanyInput {
	return companyusecase.UpdateCompanyInput{
		Name:       r.Name,
		PartnerID:  r.PartnerID,
		CurrencyID: r.CurrencyID,
		Phone:      r.Phone,
		Email:      r.Email,
		Website:    r.Website,
		VAT:        r.VAT,
		Street:     r.Street,
		Street2:    r.Street2,
		City:       r.City,
		State:      r.State,
		Country:    r.Country,
		ZipCode:    r.ZipCode,
	}
}

// CompanyResponse formats company data for HTTP JSON client output.
type CompanyResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PartnerID  *int64 `json:"partner_id,omitempty"`
	CurrencyID int64  `json:"currency_id"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
	Website    string `json:"website,omitempty"`
	VAT        string `json:"vat,omitempty"`
	Street     string `json:"street,omitempty"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	Country    string `json:"country,omitempty"`
	ZipCode    string `json:"zip_code,omitempty"`
	Active     bool   `json:"active"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ToCompanyResponse converts a domain Company entity into CompanyResponse DTO.
func ToCompanyResponse(c *company.Company) CompanyResponse {
	if c == nil {
		return CompanyResponse{}
	}
	return CompanyResponse{
		ID:         c.ID,
		Name:       c.Name,
		PartnerID:  c.PartnerID,
		CurrencyID: c.CurrencyID,
		Phone:      c.Phone,
		Email:      c.Email,
		Website:    c.Website,
		VAT:        c.VAT,
		Street:     c.Street,
		Street2:    c.Street2,
		City:       c.City,
		State:      c.State,
		Country:    c.Country,
		ZipCode:    c.ZipCode,
		Active:     c.Active,
		CreatedAt:  c.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  c.Audit.UpdatedAt.Format(time.RFC3339),
	}
}

// ToCompanyResponseList converts a slice of domain companies into response DTOs.
func ToCompanyResponseList(items []company.Company) []CompanyResponse {
	result := make([]CompanyResponse, len(items))
	for i, item := range items {
		result[i] = ToCompanyResponse(&item)
	}
	return result
}
