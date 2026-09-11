package subscription

import (
	"time"
)

type SubscriptionPlan struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Period         string    `json:"period"` // e.g., "monthly", "yearly", "daily"
	PeriodInterval int       `json:"period_interval"`
	Price          float64   `json:"price"`
	Currency       string    `json:"currency"`
	ProductID      int64     `json:"product_id"`
	CompanyID      int64     `json:"company_id"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}

func (p *SubscriptionPlan) Validate() error {
	if p.Name == "" {
		return errInvalid("plan name cannot be empty")
	}
	if p.Period == "" {
		return errInvalid("plan period cannot be empty")
	}
	if p.PeriodInterval <= 0 {
		return errInvalid("period interval must be greater than zero")
	}
	if p.Price < 0 {
		return errInvalid("price cannot be negative")
	}
	if p.ProductID <= 0 {
		return errInvalid("product ID is required")
	}
	if p.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	return nil
}
