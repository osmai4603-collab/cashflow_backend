package partner

import (
	"fmt"
	"net/mail"
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// PartnerType represents the entity nature: individual person or company/organization.
type PartnerType string

const (
	PartnerTypeIndividual PartnerType = "individual"
	PartnerTypeCompany    PartnerType = "company"
)

// Partner represents a contact, customer, vendor, or organizational entity (res.partner).
type Partner struct {
	ID         int64        `json:"id"`
	Name       string       `json:"name"`
	Email      string       `json:"email,omitempty"`
	Phone      string       `json:"phone,omitempty"`
	Mobile     string       `json:"mobile,omitempty"`
	Type       PartnerType  `json:"type"`
	IsCustomer bool         `json:"is_customer"`
	IsSupplier bool         `json:"is_supplier"`
	VATNumber  string       `json:"vat_number,omitempty"`
	Website    string       `json:"website,omitempty"`
	CompanyID  *int64       `json:"company_id,omitempty"`
	ParentID   *int64       `json:"parent_id,omitempty"`
	Street     string       `json:"street,omitempty"`
	Street2    string       `json:"street2,omitempty"`
	City       string       `json:"city,omitempty"`
	State      string       `json:"state,omitempty"`
	Country    string       `json:"country,omitempty"`
	ZipCode    string       `json:"zip_code,omitempty"`
	Active     bool         `json:"active"`
	Audit      audit.Fields `json:"audit"`
}

// Validate ensures domain invariants and business rules are strictly upheld.
func (p *Partner) Validate() error {
	trimmedName := strings.TrimSpace(p.Name)
	if trimmedName == "" {
		return platformerrors.Validation("partner name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if len(trimmedName) > 255 {
		return platformerrors.Validation("partner name exceeds maximum length", map[string]string{
			"name": "must not exceed 255 characters",
		})
	}
	p.Name = trimmedName

	if p.Type != PartnerTypeIndividual && p.Type != PartnerTypeCompany {
		return platformerrors.Validation("invalid partner type", map[string]string{
			"type": fmt.Sprintf("must be either '%s' or '%s'", PartnerTypeIndividual, PartnerTypeCompany),
		})
	}

	if p.Email != "" {
		trimmedEmail := strings.TrimSpace(p.Email)
		if _, err := mail.ParseAddress(trimmedEmail); err != nil {
			return platformerrors.Validation("invalid email address format", map[string]string{
				"email": "invalid email address",
			})
		}
		p.Email = strings.ToLower(trimmedEmail)
	}

	if p.ParentID != nil && p.ID > 0 && *p.ParentID == p.ID {
		return platformerrors.Validation("circular reference in hierarchy", map[string]string{
			"parent_id": "partner cannot be its own parent",
		})
	}

	return nil
}
