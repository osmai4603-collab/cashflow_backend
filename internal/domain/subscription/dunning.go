package subscription

import (
	"context"
	"fmt"
)

type DunningProcess struct {
	repo Repository
}

func NewDunningProcess(repo Repository) *DunningProcess {
	return &DunningProcess{
		repo: repo,
	}
}

func (d *DunningProcess) HandleFailedPayment(ctx context.Context, sub *SaleSubscription) error {
	// Increment failed charge logic is handled by recurring engine or here
	// Determine action based on failed charge count
	switch sub.FailedChargeCount {
	case 0:
		// First failure: send reminder, keep active
		fmt.Printf("Dunning: First payment failure for subscription %s. Sending reminder.\n", sub.Code)
	case 1:
		// Second failure: retry warning
		fmt.Printf("Dunning: Second payment failure for subscription %s. Warning sent.\n", sub.Code)
	case 2:
		// Third failure: Mark past due / suspend services
		sub.State = StatePastDue
		fmt.Printf("Dunning: Third payment failure for subscription %s. Status changed to past_due.\n", sub.Code)
	default:
		// Critical: cancel subscription
		sub.State = StateCanceled
		fmt.Printf("Dunning: Multiple failures for subscription %s. Canceling subscription.\n", sub.Code)
	}
	return nil
}
