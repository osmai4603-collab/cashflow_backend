package subscriptionusecase

import (
	"context"
	"testing"
	"time"

	subscriptionstorage "cashflow_backend/internal/adapters/storage/subscription"
	"cashflow_backend/internal/domain/subscription"
)

func TestSubscriptionUseCase_LifecycleAndRecurringBilling(t *testing.T) {
	repo := subscriptionstorage.NewMemoryRepo()
	ctx := context.Background()
	plan := &subscription.SubscriptionPlan{ID: 1, Name: "Monthly Pro", Period: "monthly", PeriodInterval: 1, Price: 125, Currency: "USD", ProductID: 1, CompanyID: 1, Active: true}
	if err := repo.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	uc := New(repo)
	start := time.Now().UTC().Add(-2 * time.Hour)
	sub, err := uc.CreateSubscription(ctx, &subscription.SaleSubscription{PartnerID: 10, PlanID: 1, StartDate: start, NextBillingDate: start})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	if sub.RecurringAmount != 125 || sub.State != subscription.StateDraft || sub.Code == "" {
		t.Fatalf("unexpected subscription defaults: %#v", sub)
	}
	if err := uc.ActivateSubscription(ctx, sub.ID); err != nil {
		t.Fatalf("activate subscription: %v", err)
	}
	if err := uc.ProcessBillingCycle(ctx); err != nil {
		t.Fatalf("process billing cycle: %v", err)
	}
	updated, err := uc.GetSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("get billed subscription: %v", err)
	}
	if !updated.NextBillingDate.After(start) || updated.State != subscription.StateActive || updated.FailedChargeCount != 0 {
		t.Fatalf("billing cycle did not advance subscription: %#v", updated)
	}
	mrr, err := uc.GetMRR(ctx)
	if err != nil || mrr != 125 {
		t.Fatalf("unexpected MRR: %v, %v", mrr, err)
	}
	if err := uc.CancelSubscription(ctx, sub.ID); err != nil {
		t.Fatalf("cancel subscription: %v", err)
	}
}
