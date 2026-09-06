package partnerhttp

import (
	"time"

	"cashflow_backend/internal/domain/partner"
	partnerusecase "cashflow_backend/internal/usecase/partner"
)

// CreatePartnerRequest represents the incoming JSON payload to register a new partner.
type CreatePartnerRequest struct {
	Name       string              `json:"name"`
	Email      string              `json:"email"`
	Phone      string              `json:"phone"`
	Mobile     string              `json:"mobile"`
	Type       partner.PartnerType `json:"type"`
	IsCustomer *bool               `json:"is_customer"`
	IsSupplier *bool               `json:"is_supplier"`
	VATNumber  string              `json:"vat_number"`
	Website    string              `json:"website"`
	CompanyID  *int64              `json:"company_id"`
	ParentID   *int64              `json:"parent_id"`
	Street     string              `json:"street"`
	Street2    string              `json:"street2"`
	City       string              `json:"city"`
	State      string              `json:"state"`
	Country    string              `json:"country"`
	ZipCode    string              `json:"zip_code"`
}

// ToInput maps the HTTP request DTO to the application usecase input.
func (r CreatePartnerRequest) ToInput() partnerusecase.CreatePartnerInput {
	return partnerusecase.CreatePartnerInput{
		Name:       r.Name,
		Email:      r.Email,
		Phone:      r.Phone,
		Mobile:     r.Mobile,
		Type:       r.Type,
		IsCustomer: r.IsCustomer,
		IsSupplier: r.IsSupplier,
		VATNumber:  r.VATNumber,
		Website:    r.Website,
		CompanyID:  r.CompanyID,
		ParentID:   r.ParentID,
		Street:     r.Street,
		Street2:    r.Street2,
		City:       r.City,
		State:      r.State,
		Country:    r.Country,
		ZipCode:    r.ZipCode,
	}
}

// UpdatePartnerRequest represents partial or full fields to update an existing partner.
type UpdatePartnerRequest struct {
	Name       *string              `json:"name"`
	Email      *string              `json:"email"`
	Phone      *string              `json:"phone"`
	Mobile     *string              `json:"mobile"`
	Type       *partner.PartnerType `json:"type"`
	IsCustomer *bool                `json:"is_customer"`
	IsSupplier *bool                `json:"is_supplier"`
	VATNumber  *string              `json:"vat_number"`
	Website    *string              `json:"website"`
	CompanyID  *int64               `json:"company_id"`
	ParentID   *int64               `json:"parent_id"`
	Street     *string              `json:"street"`
	Street2    *string              `json:"street2"`
	City       *string              `json:"city"`
	State      *string              `json:"state"`
	Country    *string              `json:"country"`
	ZipCode    *string              `json:"zip_code"`
}

// ToInput maps the update request DTO to the application usecase input.
func (r UpdatePartnerRequest) ToInput() partnerusecase.UpdatePartnerInput {
	return partnerusecase.UpdatePartnerInput{
		Name:       r.Name,
		Email:      r.Email,
		Phone:      r.Phone,
		Mobile:     r.Mobile,
		Type:       r.Type,
		IsCustomer: r.IsCustomer,
		IsSupplier: r.IsSupplier,
		VATNumber:  r.VATNumber,
		Website:    r.Website,
		CompanyID:  r.CompanyID,
		ParentID:   r.ParentID,
		Street:     r.Street,
		Street2:    r.Street2,
		City:       r.City,
		State:      r.State,
		Country:    r.Country,
		ZipCode:    r.ZipCode,
	}
}

// PartnerResponse formats partner data for HTTP JSON client output.
type PartnerResponse struct {
	ID         int64               `json:"id"`
	Name       string              `json:"name"`
	Email      string              `json:"email,omitempty"`
	Phone      string              `json:"phone,omitempty"`
	Mobile     string              `json:"mobile,omitempty"`
	Type       partner.PartnerType `json:"type"`
	IsCustomer bool                `json:"is_customer"`
	IsSupplier bool                `json:"is_supplier"`
	VATNumber  string              `json:"vat_number,omitempty"`
	Website    string              `json:"website,omitempty"`
	CompanyID  *int64              `json:"company_id,omitempty"`
	ParentID   *int64              `json:"parent_id,omitempty"`
	Street     string              `json:"street,omitempty"`
	Street2    string              `json:"street2,omitempty"`
	City       string              `json:"city,omitempty"`
	State      string              `json:"state,omitempty"`
	Country    string              `json:"country,omitempty"`
	ZipCode    string              `json:"zip_code,omitempty"`
	Active     bool                `json:"active"`
	CreatedAt  string              `json:"created_at"`
	UpdatedAt  string              `json:"updated_at"`
}

// ToPartnerResponse converts a domain Partner entity into PartnerResponse DTO.
func ToPartnerResponse(p *partner.Partner) PartnerResponse {
	if p == nil {
		return PartnerResponse{}
	}
	return PartnerResponse{
		ID:         p.ID,
		Name:       p.Name,
		Email:      p.Email,
		Phone:      p.Phone,
		Mobile:     p.Mobile,
		Type:       p.Type,
		IsCustomer: p.IsCustomer,
		IsSupplier: p.IsSupplier,
		VATNumber:  p.VATNumber,
		Website:    p.Website,
		CompanyID:  p.CompanyID,
		ParentID:   p.ParentID,
		Street:     p.Street,
		Street2:    p.Street2,
		City:       p.City,
		State:      p.State,
		Country:    p.Country,
		ZipCode:    p.ZipCode,
		Active:     p.Active,
		CreatedAt:  p.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  p.Audit.UpdatedAt.Format(time.RFC3339),
	}
}

// ToPartnerResponseList converts a slice of domain partners into response DTOs.
func ToPartnerResponseList(items []partner.Partner) []PartnerResponse {
	result := make([]PartnerResponse, len(items))
	for i, item := range items {
		result[i] = ToPartnerResponse(&item)
	}
	return result
}
