package subscription

import (
	"time"
)

type SubscriptionState string

const (
	StateDraft    SubscriptionState = "draft"
	StateActive   SubscriptionState = "active"
	StatePastDue  SubscriptionState = "past_due"
	StateCanceled SubscriptionState = "canceled"
)

type SaleSubscription struct {
	ID                int64             `json:"id"`
	Code              string            `json:"code"`
	PartnerID         int64             `json:"partner_id"`
	PlanID            int64             `json:"plan_id"`
	State             SubscriptionState `json:"state"`
	StartDate         time.Time         `json:"start_date"`
	NextBillingDate   time.Time         `json:"next_billing_date"`
	EndDate           *time.Time        `json:"end_date,omitempty"`
	RecurringAmount   float64           `json:"recurring_amount"`
	PaymentTokenID    *int64            `json:"payment_token_id,omitempty"`
	FailedChargeCount int               `json:"failed_charge_count"`
	CompanyID         int64             `json:"company_id"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func (s *SaleSubscription) Validate() error {
	if s.Code == "" {
		return errInvalid("subscription code cannot be empty")
	}
	if s.PartnerID <= 0 {
		return errInvalid("partner ID is required")
	}
	if s.PlanID <= 0 {
		return errInvalid("plan ID is required")
	}
	if s.NextBillingDate.IsZero() {
		return errInvalid("next billing date is required")
	}
	if s.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	return nil
}

func (s *SaleSubscription) CalculateNextBillingDate(plan *SubscriptionPlan) time.Time {
	base := s.NextBillingDate
	if base.IsZero() {
		base = s.StartDate
	}
	interval := plan.PeriodInterval
	if interval <= 0 {
		interval = 1
	}

	switch plan.Period {
	case "daily":
		return base.AddDate(0, 0, interval)
	case "yearly":
		return base.AddDate(interval, 0, 0)
	case "monthly":
	default:
		return base.AddDate(0, interval, 0)
	}
	return base.AddDate(0, interval, 0)
}
