package subscription

import (
	"context"
	"fmt"
	"time"
)

type RecurringEngine struct {
	repo         Repository
	dunning      *DunningProcess
	invoicePorts InvoicePort
}

func NewRecurringEngine(repo Repository, dunning *DunningProcess, invoicePorts InvoicePort) *RecurringEngine {
	return &RecurringEngine{
		repo:         repo,
		dunning:      dunning,
		invoicePorts: invoicePorts,
	}
}

func (e *RecurringEngine) ProcessDueSubscriptions(ctx context.Context, now time.Time, companyID int64) error {
	subs, err := e.repo.ListDueSubscriptions(ctx, now, companyID)
	if err != nil {
		return fmt.Errorf("failed to list due subscriptions: %w", err)
	}

	for _, sub := range subs {
		plan, err := e.repo.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			continue
		}

		// Attempt to bill via InvoicePort or payment token
		err = e.invoicePorts.CreateSubscriptionInvoice(ctx, &sub, plan)
		if err != nil {
			// Billing failed, trigger dunning process
			errDunning := e.dunning.HandleFailedPayment(ctx, &sub)
			if errDunning != nil {
				// Log or handle error if needed
			}
			sub.FailedChargeCount++
			if sub.FailedChargeCount >= 3 {
				sub.State = StatePastDue
			}
			sub.UpdatedAt = now
			_ = e.repo.UpdateSubscription(ctx, &sub)
		} else {
			// Billing succeeded, advance next billing date
			sub.State = StateActive
			sub.FailedChargeCount = 0
			sub.NextBillingDate = sub.CalculateNextBillingDate(plan)
			sub.UpdatedAt = now
			_ = e.repo.UpdateSubscription(ctx, &sub)
		}
	}

	return nil
}
