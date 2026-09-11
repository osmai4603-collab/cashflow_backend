package subscription

import (
	"context"
)

type SubscriptionMetrics struct {
	MRR            float64 `json:"mrr"`
	ARR            float64 `json:"arr"`
	ActiveCount    int     `json:"active_count"`
	CanceledCount  int     `json:"canceled_count"`
	ChurnRate      float64 `json:"churn_rate"` // Percentage e.g. 5.5 for 5.5%
}

type MetricsEngine struct {
	repo Repository
}

func NewMetricsEngine(repo Repository) *MetricsEngine {
	return &MetricsEngine{
		repo: repo,
	}
}

func (m *MetricsEngine) CalculateMetrics(ctx context.Context, companyID int64) (*SubscriptionMetrics, error) {
	subs, err := m.repo.ListSubscriptionsByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}

	metrics := &SubscriptionMetrics{}
	var activeTotal float64

	for _, sub := range subs {
		plan, err := m.repo.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			continue
		}

		// Normalize to monthly recurring revenue (MRR)
		var monthlyAmount float64
		interval := plan.PeriodInterval
		if interval <= 0 {
			interval = 1
		}

		switch plan.Period {
		case "daily":
			monthlyAmount = (sub.RecurringAmount / float64(interval)) * 30.0
		case "yearly":
			monthlyAmount = sub.RecurringAmount / (float64(interval) * 12.0)
		case "monthly":
			monthlyAmount = sub.RecurringAmount / float64(interval)
		default:
			monthlyAmount = sub.RecurringAmount / float64(interval)
		}

		if sub.State == StateActive {
			metrics.ActiveCount++
			activeTotal += monthlyAmount
		} else if sub.State == StateCanceled {
			metrics.CanceledCount++
		}
	}

	metrics.MRR = activeTotal
	metrics.ARR = activeTotal * 12.0

	totalEndedOrActive := metrics.ActiveCount + metrics.CanceledCount
	if totalEndedOrActive > 0 {
		metrics.ChurnRate = (float64(metrics.CanceledCount) / float64(totalEndedOrActive)) * 100.0
	}

	return metrics, nil
}
