package subscriptionusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/subscription"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo        subscription.Repository
	invoicePort subscription.InvoicePort
}

func (u *UseCase) ListSubscriptions(ctx context.Context) ([]subscription.SaleSubscription, error) {
	return u.repo.ListSubscriptionsByCompany(ctx, 1)
}

func (u *UseCase) CreateSubscription(ctx context.Context, s *subscription.SaleSubscription) (*subscription.SaleSubscription, error) {
	if s == nil {
		return nil, platformerrors.Validation("subscription is required", nil)
	}
	if s.CompanyID == 0 {
		s.CompanyID = 1
	}
	if s.Code == "" {
		s.Code = fmt.Sprintf("SUB-%d", time.Now().UnixNano())
	}
	if s.State == "" {
		s.State = subscription.StateDraft
	}
	if s.StartDate.IsZero() {
		s.StartDate = time.Now().UTC()
	}
	if s.NextBillingDate.IsZero() {
		s.NextBillingDate = s.StartDate
	}
	plan, err := u.repo.GetPlanByID(ctx, s.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("subscription plan %d not found", s.PlanID))
	}
	if s.RecurringAmount == 0 {
		s.RecurringAmount = plan.Price
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	s.CreatedAt = time.Now().UTC()
	s.UpdatedAt = s.CreatedAt
	if err := u.repo.CreateSubscription(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *UseCase) GetSubscription(ctx context.Context, id int64) (*subscription.SaleSubscription, error) {
	if id <= 0 {
		return nil, platformerrors.Validation("subscription_id is required", nil)
	}
	sub, err := u.repo.GetSubscriptionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("subscription %d not found", id))
	}
	return sub, nil
}

func (u *UseCase) ActivateSubscription(ctx context.Context, id int64) error {
	sub, err := u.GetSubscription(ctx, id)
	if err != nil {
		return err
	}
	if sub.State != subscription.StateDraft && sub.State != subscription.StatePastDue {
		return platformerrors.Conflict("subscription cannot be activated from its current state")
	}
	plan, err := u.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return err
	}
	if plan == nil || !plan.Active {
		return platformerrors.Conflict("subscription plan is inactive")
	}
	sub.State = subscription.StateActive
	sub.FailedChargeCount = 0
	sub.UpdatedAt = time.Now().UTC()
	return u.repo.UpdateSubscription(ctx, sub)
}

func (u *UseCase) CancelSubscription(ctx context.Context, id int64) error {
	sub, err := u.GetSubscription(ctx, id)
	if err != nil {
		return err
	}
	if sub.State == subscription.StateCanceled {
		return platformerrors.Conflict("subscription is already canceled")
	}
	now := time.Now().UTC()
	sub.State = subscription.StateCanceled
	sub.EndDate = &now
	sub.UpdatedAt = now
	return u.repo.UpdateSubscription(ctx, sub)
}

func (u *UseCase) ProcessBillingCycle(ctx context.Context) error {
	engine := subscription.NewRecurringEngine(u.repo, subscription.NewDunningProcess(u.repo), u.invoicePort)
	return engine.ProcessDueSubscriptions(ctx, time.Now().UTC(), 1)
}

func (u *UseCase) GetMRR(ctx context.Context) (float64, error) {
	metrics, err := subscription.NewMetricsEngine(u.repo).CalculateMetrics(ctx, 1)
	if err != nil {
		return 0, err
	}
	return metrics.MRR, nil
}

func New(repo subscription.Repository, invoicePorts ...subscription.InvoicePort) *UseCase {
	var invoicePort subscription.InvoicePort
	if len(invoicePorts) > 0 {
		invoicePort = invoicePorts[0]
	}
	if invoicePort == nil {
		invoicePort = noOpInvoicePort{}
	}
	return &UseCase{repo: repo, invoicePort: invoicePort}
}

type noOpInvoicePort struct{}

func (noOpInvoicePort) CreateSubscriptionInvoice(context.Context, *subscription.SaleSubscription, *subscription.SubscriptionPlan) error {
	return nil
}
