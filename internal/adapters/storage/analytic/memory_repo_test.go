package analyticstorage

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"cashflow_backend/internal/domain/analytic"
	"cashflow_backend/internal/platform/filter"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_Seeds(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	plans, err := repo.ListPlans(ctx, false)
	if err != nil {
		t.Fatalf("unexpected error listing plans: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 seeded plans, got %d", len(plans))
	}
	if planByName(plans, "Project") == nil || planByName(plans, "Departments") == nil {
		t.Fatalf("expected seeded plans Project and Departments")
	}

	accounts, err := repo.ListAccounts(ctx, nil, pagination.PageRequest{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing accounts: %v", err)
	}
	if accounts.TotalItems != 3 {
		t.Fatalf("expected 3 seeded accounts, got %d", accounts.TotalItems)
	}

	projectPlanID, err := repo.GetProjectPlanID(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting project plan id: %v", err)
	}
	if projectPlanID != 1 {
		t.Fatalf("expected project plan id 1, got %d", projectPlanID)
	}
}

func TestMemoryRepo_PlanCRUDAndHierarchy(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	region := &analytic.AnalyticPlan{
		Name:                "Region",
		Description:         "Regional branch plan",
		Sequence:            30,
		DefaultApplicability: analytic.AppOptional,
	}
	if err := repo.CreatePlan(ctx, region); err != nil {
		t.Fatalf("unexpected error creating root plan: %v", err)
	}
	if region.ID == 0 || region.RootID != region.ID {
		t.Fatalf("expected self-referencing root plan, got id=%d root=%d", region.ID, region.RootID)
	}

	sub := &analytic.AnalyticPlan{
		Name:                "North",
		ParentID:            intPtr(region.ID),
		DefaultApplicability: analytic.AppMandatory,
	}
	if err := repo.CreatePlan(ctx, sub); err != nil {
		t.Fatalf("unexpected error creating child plan: %v", err)
	}
	if sub.RootID != region.ID {
		t.Fatalf("expected child root = Region, got %d", sub.RootID)
	}
	if sub.ParentPath != "Region" {
		t.Fatalf("expected child parent_path Region, got %q", sub.ParentPath)
	}
	if sub.CompleteName != "Region / North" {
		t.Fatalf("expected complete_name 'Region / North', got %q", sub.CompleteName)
	}

	children, err := repo.GetChildrenPlans(ctx, region.ID)
	if err != nil {
		t.Fatalf("unexpected error listing children: %v", err)
	}
	if len(children) != 1 || children[0].ID != sub.ID {
		t.Fatalf("expected 1 child, got %d", len(children))
	}

	// UpdatePlan preserves audit and refreshes complete name.
	sub.Name = "South"
	sub.ParentPath = "Region"
	if err := repo.UpdatePlan(ctx, sub); err != nil {
		t.Fatalf("unexpected error updating plan: %v", err)
	}
	updated, err := repo.GetPlanByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("unexpected error re-fetching plan: %v", err)
	}
	if updated.CompleteName != "Region / South" {
		t.Fatalf("expected updated complete_name, got %q", updated.CompleteName)
	}

	// Root plan with attached accounts cannot be deleted.
	if err := repo.DeletePlan(ctx, 1); !isAppError(err, platformerrors.CodeConflict) {
		t.Fatalf("expected conflict deleting plan with accounts, got %v", err)
	}

	// Deleting a root plan cascades to its children.
	if err := repo.DeletePlan(ctx, region.ID); err != nil {
		t.Fatalf("unexpected error deleting childless root plan: %v", err)
	}
	if _, err := repo.GetPlanByID(ctx, region.ID); !isAppError(err, platformerrors.CodeNotFound) {
		t.Fatalf("expected NotFound for deleted plan, got %v", err)
	}
	if _, err := repo.GetPlanByID(ctx, sub.ID); !isAppError(err, platformerrors.CodeNotFound) {
		t.Fatalf("expected NotFound for cascade-deleted child, got %v", err)
	}
}

func TestMemoryRepo_ApplicabilityAndRelevantPlans(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	// Global (all companies) mandatory rule for the Project plan in "general".
	if err := repo.SetApplicability(ctx, &analytic.AnalyticApplicability{
		PlanID:         1,
		BusinessDomain: analytic.DomainGeneral,
		Applicability:  analytic.AppMandatory,
		Sequence:       10,
	}); err != nil {
		t.Fatalf("unexpected error setting applicability: %v", err)
	}
	// Company-scoped optional rule (should outweigh global for that company).
	if err := repo.SetApplicability(ctx, &analytic.AnalyticApplicability{
		PlanID:         1,
		BusinessDomain: analytic.DomainGeneral,
		Applicability:  analytic.AppOptional,
		CompanyID:      intPtr(7),
		Sequence:       5,
	}); err != nil {
		t.Fatalf("unexpected error setting company applicability: %v", err)
	}

	// Upsert: the same (plan, domain, company) rule is updated, not duplicated.
	if err := repo.SetApplicability(ctx, &analytic.AnalyticApplicability{
		PlanID:         1,
		BusinessDomain: analytic.DomainGeneral,
		Applicability:  analytic.AppOptional,
		CompanyID:      intPtr(7),
		Sequence:       5,
	}); err != nil {
		t.Fatalf("unexpected error updating applicability: %v", err)
	}
	rules, err := repo.GetApplicabilities(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error listing applicabilities: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 applicability rules after upsert, got %d", len(rules))
	}

	// Company 7 resolves to its own optional rule.
	relevant, err := repo.GetRelevantPlans(ctx, 7, analytic.DomainGeneral)
	if err != nil {
		t.Fatalf("unexpected error getting relevant plans: %v", err)
	}
	if len(relevant) != 2 {
		t.Fatalf("expected 2 relevant plans, got %d", len(relevant))
	}
	project := relevantPlanByID(relevant, 1)
	if project == nil {
		t.Fatal("expected Project plan to be relevant")
	}
	if project.Applicability != analytic.AppOptional {
		t.Fatalf("expected optional for company 7, got %s", project.Applicability)
	}
	if project.ColumnName != "account_id" {
		t.Fatalf("expected project plan column account_id, got %s", project.ColumnName)
	}
	depts := relevantPlanByID(relevant, 2)
	if depts.ColumnName != "x_plan2_id" {
		t.Fatalf("expected departments column x_plan2_id, got %s", depts.ColumnName)
	}

	// Other companies fall back to the global mandatory rule.
	relevant, err = repo.GetRelevantPlans(ctx, 99, analytic.DomainGeneral)
	if err != nil {
		t.Fatalf("unexpected error getting relevant plans: %v", err)
	}
	if got := relevantPlanByID(relevant, 1); got == nil || got.Applicability != analytic.AppMandatory {
		t.Fatalf("expected global mandatory for company 99, got %+v", got)
	}

	// A plan marked unavailable for the domain is excluded entirely.
	if err := repo.SetApplicability(ctx, &analytic.AnalyticApplicability{
		PlanID:         2,
		BusinessDomain: analytic.DomainSaleOrder,
		Applicability:  analytic.AppUnavailable,
		Sequence:       1,
	}); err != nil {
		t.Fatalf("unexpected error setting unavailable rule: %v", err)
	}
	relevant, err = repo.GetRelevantPlans(ctx, 1, analytic.DomainSaleOrder)
	if err != nil {
		t.Fatalf("unexpected error getting relevant plans: %v", err)
	}
	for _, p := range relevant {
		if p.ID == 2 {
			t.Fatal("expected Departments plan to be excluded as unavailable")
		}
	}
}

func TestMemoryRepo_AccountTotalsAndDeletion(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	acct := &analytic.AnalyticAccount{Name: "Marketing Campaign", PlanID: 2}
	if err := repo.CreateAccount(ctx, acct); err != nil {
		t.Fatalf("unexpected error creating account: %v", err)
	}
	if acct.RootPlanID != 2 {
		t.Fatalf("expected root plan 2, got %d", acct.RootPlanID)
	}

	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	lines := []analytic.AnalyticLine{
		{Name: "Invoice revenue", Date: date, Amount: 1000, UserID: 1, CompanyID: 1, AccountID: acct.ID, Source: analytic.SourceInvoice},
		{Name: "Refund", Date: date, Amount: -250, UserID: 1, CompanyID: 1, AccountID: acct.ID},
		{Name: "Older entry", Date: date.AddDate(0, -1, 0), Amount: 750, UserID: 1, CompanyID: 1, AccountID: acct.ID},
	}
	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("unexpected error creating lines: %v", err)
	}

	// Totals across all dates: credit 1750, debit 250, balance 1500.
	totals, err := repo.GetAccountTotals(ctx, acct.ID, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error getting totals: %v", err)
	}
	if round4(totals.Credit) != 1750 || round4(totals.Debit) != 250 || round4(totals.Balance) != 1500 {
		t.Fatalf("expected (1750, 250, 1500), got (%v, %v, %v)", totals.Credit, totals.Debit, totals.Balance)
	}

	// Totals within June 2026 exclude the older entry: balance 750.
	from, to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	totals, err = repo.GetAccountTotals(ctx, acct.ID, &from, &to)
	if err != nil {
		t.Fatalf("unexpected error getting dated totals: %v", err)
	}
	if round4(totals.Balance) != 750 {
		t.Fatalf("expected June balance 750, got %v", totals.Balance)
	}

	// List filter by account_id with pagination.
	page, err := repo.ListAccounts(ctx,
		filter.NewFilter(filter.Criterion{Field: "plan_id", Operator: "eq", Value: 2}),
		pagination.PageRequest{Limit: 1, Page: 1},
	)
	if err != nil {
		t.Fatalf("unexpected error filtering accounts: %v", err)
	}
	if page.TotalItems != 3 { // 2 seeded + 1 new
		t.Fatalf("expected 3 accounts in plan 2, got %d", page.TotalItems)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected page of 1, got %d", len(page.Items))
	}

	// Account referenced by lines cannot be deleted.
	if err := repo.DeleteAccount(ctx, acct.ID); !isAppError(err, platformerrors.CodeConflict) {
		t.Fatalf("expected conflict deleting account with lines, got %v", err)
	}
}

func TestMemoryRepo_LineCRUD(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	dist := analytic.AnalyticDistribution{"1": 100}
	line := &analytic.AnalyticLine{
		Name:         "Consulting",
		Date:         time.Now().UTC(),
		Amount:       500,
		UserID:       3,
		CompanyID:    1,
		AccountID:    1,
		MoveLineID:   intPtr(9001),
		Distribution: dist,
	}
	if err := repo.CreateLine(ctx, line); err != nil {
		t.Fatalf("unexpected error creating line: %v", err)
	}
	if line.ID == 0 {
		t.Fatal("expected line id to be assigned")
	}

	got, err := repo.GetLineByID(ctx, line.ID)
	if err != nil {
		t.Fatalf("unexpected error reading line: %v", err)
	}
	if got.Distribution["1"] != 100 {
		t.Fatalf("expected distribution snapshot to persist, got %+v", got.Distribution)
	}

	line.Amount = 600
	if err := repo.UpdateLine(ctx, line); err != nil {
		t.Fatalf("unexpected error updating line: %v", err)
	}
	if got, err = repo.GetLineByID(ctx, line.ID); err != nil {
		t.Fatalf("unexpected error re-reading line: %v", err)
	}
	if got.Amount != 600 {
		t.Fatalf("expected updated amount 600, got %v", got.Amount)
	}

	moveLines, err := repo.ListLinesByMoveLine(ctx, 9001)
	if err != nil {
		t.Fatalf("unexpected error listing move lines: %v", err)
	}
	if len(moveLines) != 1 || moveLines[0].ID != line.ID {
		t.Fatalf("expected 1 line linked to move line 9001, got %d", len(moveLines))
	}

	page, err := repo.ListLines(ctx, nil, pagination.PageRequest{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing lines: %v", err)
	}
	if page.TotalItems != 1 || page.Items[0].ID != line.ID {
		t.Fatalf("expected 1 line, got %d", page.TotalItems)
	}

	if err := repo.DeleteAccount(ctx, 1); !isAppError(err, platformerrors.CodeConflict) {
		t.Fatalf("expected conflict deleting seeded account with lines, got %v", err)
	}
}

func TestMemoryRepo_DistributionModels(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	general := &analytic.DistributionModel{
		Sequence:     30,
		Distribution: analytic.AnalyticDistribution{"2": 100},
	}
	category := &analytic.DistributionModel{
		Sequence:          20,
		PartnerCategoryID: intPtr(5),
		Distribution:      analytic.AnalyticDistribution{"2": 50, "3": 50},
	}
	partner := &analytic.DistributionModel{
		Sequence:     10,
		PartnerID:    intPtr(42),
		Distribution: analytic.AnalyticDistribution{"1": 100},
	}
	for _, m := range []*analytic.DistributionModel{general, category, partner} {
		if err := repo.CreateDistributionModel(ctx, m); err != nil {
			t.Fatalf("unexpected error creating distribution model: %v", err)
		}
	}

	// Partner-specific rule (specificity 2) beats the generic rules.
	matched, err := repo.MatchDistribution(ctx, intPtr(42), nil, 1)
	if err != nil {
		t.Fatalf("unexpected error matching distribution: %v", err)
	}
	if matched["1"] != 100 || len(matched) != 1 {
		t.Fatalf("expected partner-specific rule to win, got %+v", matched)
	}

	// Same partner but with a category: partner specificity (2) > category (1).
	matched, err = repo.MatchDistribution(ctx, intPtr(42), intPtr(5), 1)
	if err != nil {
		t.Fatalf("unexpected error matching distribution: %v", err)
	}
	if matched["1"] != 100 || len(matched) != 1 {
		t.Fatalf("expected partner rule to outrank the category rule, got %+v", matched)
	}

	// With no partner/category in context, only the generic rule applies.
	matched, err = repo.MatchDistribution(ctx, nil, nil, 1)
	if err != nil {
		t.Fatalf("unexpected error matching distribution: %v", err)
	}
	if matched["2"] != 100 || len(matched) != 1 {
		t.Fatalf("expected global rule to win, got %+v", matched)
	}

	if err := repo.DeleteDistributionModel(ctx, partner.ID); err != nil {
		t.Fatalf("unexpected error deleting distribution model: %v", err)
	}
	if _, err := repo.GetDistributionModelByID(ctx, partner.ID); !isAppError(err, platformerrors.CodeNotFound) {
		t.Fatalf("expected NotFound for deleted model, got %v", err)
	}

	// The returned distribution must be a copy, not shared state.
	partnerAgain := &analytic.DistributionModel{
		Sequence:     1,
		PartnerID:    intPtr(77),
		Distribution: analytic.AnalyticDistribution{"2": 100},
	}
	if err := repo.CreateDistributionModel(ctx, partnerAgain); err != nil {
		t.Fatalf("unexpected error creating distribution model: %v", err)
	}
	matched, err = repo.MatchDistribution(ctx, intPtr(77), nil, 1)
	if err != nil {
		t.Fatalf("unexpected error matching distribution: %v", err)
	}
	matched["1"] = 90 // mutation must not corrupt the stored model
	stored, err := repo.GetDistributionModelByID(ctx, partnerAgain.ID)
	if err != nil {
		t.Fatalf("unexpected error reading stored model: %v", err)
	}
	if stored.Distribution["2"] != 100 {
		t.Fatalf("expected stored distribution untouched, got %+v", stored.Distribution)
	}
}

func TestMemoryRepo_ProjectPlan(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	if err := repo.SetProjectPlanID(ctx, 2); err != nil {
		t.Fatalf("unexpected error setting project plan: %v", err)
	}
	planID, err := repo.GetProjectPlanID(ctx)
	if err != nil {
		t.Fatalf("unexpected error reading project plan: %v", err)
	}
	if planID != 2 {
		t.Fatalf("expected project plan 2, got %d", planID)
	}

	if err := repo.SetProjectPlanID(ctx, 999); !isAppError(err, platformerrors.CodeNotFound) {
		t.Fatalf("expected NotFound setting unknown project plan, got %v", err)
	}
}

func TestMemoryRepo_ValidateDistribution(t *testing.T) {
	if err := (analytic.AnalyticDistribution{"1": 50, "2": 50}).Validate100(); err != nil {
		t.Fatalf("expected valid distribution, got %v", err)
	}
	if err := (analytic.AnalyticDistribution{"1": 60, "2": 40.02}).Validate100(); err == nil {
		t.Fatal("expected out-of-tolerance distribution to fail")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func planByName(plans []analytic.AnalyticPlan, name string) *analytic.AnalyticPlan {
	for i := range plans {
		if plans[i].Name == name {
			return &plans[i]
		}
	}
	return nil
}

func relevantPlanByID(plans []analytic.RelevantPlan, id int64) *analytic.RelevantPlan {
	for i := range plans {
		if plans[i].ID == id {
			return &plans[i]
		}
	}
	return nil
}

func intPtr(v int64) *int64 { return &v }

func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func isAppError(err error, code string) bool {
	var appErr *platformerrors.AppError
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code == code
}