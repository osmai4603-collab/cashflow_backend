package payment

import (
	platformerrors "cashflow_backend/internal/platform/errors"
	"time"
)

type PaymentToken struct {
	ID             int64     `json:"id"`
	ProviderID     int64     `json:"provider_id"`
	PartnerID      int64     `json:"partner_id"`
	ProviderRef    string    `json:"provider_ref"`
	DisplayName    string    `json:"display_name"`
	PaymentDetails string    `json:"payment_details"`
	Active         bool      `json:"active"`
	Verified       bool      `json:"verified"`
	CompanyID      int64     `json:"company_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func (t *PaymentToken) Validate() error {
	if t.ProviderID <= 0 || t.PartnerID <= 0 || t.ProviderRef == "" || t.CompanyID <= 0 { return platformerrors.Validation("payment token provider, partner, reference and company are required", nil) }
	return nil
}