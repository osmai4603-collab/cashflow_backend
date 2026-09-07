package analyticusecase

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/analytic"
	analyticstorage "cashflow_backend/internal/adapters/storage/analytic"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func newTestUsecase(t *testing.T) *UseCase {
	t.Helper()
	return New(analyticstorage.NewMemoryRepo(), nil)
}

func TestUsecase_PlanCRUDAndStructure(t *testing.T) {
	uc := newTestUsecase(t)
	ctx := context.Background()

	region, err := uc.CreatePlan(ctx, CreatePlanInput{
		Name: "Branches", Description: "Branch plans", Sequence: 30,
	})
	if err != nil {
		t.Fatalf("unexpected error creating plan: %v", err)
	}
	if region.RootID != region.ID {
		t.Fatalf("expected self-rooted plan, got root=%d", region.RootID)
	}

	north, err := uc.CreatePlan(ctx, CreatePlanInput{
		Name: "North", ParentID: &region.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error creating child plan: %v", err)
	}
	if north.RootID != region.ID || north.CompleteName != "Branches / North" {
		t.Fatalf("unexpected child plan hierarchy: root=%d name=%q", north.RootID, north.CompleteName)
	}

	children, err := uc.GetChildrenPlans(ctx, region.ID)
	if err != nil {
		t.Fatalf("unexpected error listing child plans: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child plan, got %d", len(children))
	}

	// Update plan applicability.
	opt := "mandatory"
	updated, err := uc.UpdatePlan(ctx, region.ID, UpdatePlanInput{DefaultApplicability: &opt})
	if err != nil {
		t.Fatalf("unexpected error updating plan: %v", err)
	}
	if updated.DefaultApplicability != analytic.AppMandatory {
		t.Fatalf("expected mandatory applicability, got %s", updated.DefaultApplicability)
	}

	// Account created under child plan resolves to the root.
	acc, err := uc.CreateAccount(ctx, CreateAccountInput{Name: "North Ops", PlanID: north.ID})
	if err != nil {
		t.Fatalf("unexpected error creating account: %v", err)
	}
	if acc.RootPlanID != region.ID {
		t.Fatalf("expected account root plan %d, got %d", region.ID, acc.RootPlanID)
	}

	// Structure includes the account via root_plan_id.
	structure, err := uc.GetPlanStructure(ctx, region.ID)
	if err != nil {
		t.Fatalf("unexpected error getting plan structure: %v", err)
	}
	if len(structure.Accounts) != 1 || structure.Accounts[0].ID != acc.ID {
		t.Fatalf("expected structure to include the new account, got %+v", structure.Accounts)
	}

	if err := uc.DeletePlan(ctx, region.ID); err != nil {
		t.Fatalf("unexpected error deleting plan tree: %v", err)
	}
	if _, err := uc.GetPlan(ctx, north.ID); !isAppErr(err, platformerrors.CodeNotFound) {
		t.Fatalf("expected cascade-deleted child plan, got %v", err)
	}
}

func TestUsecase_RelevantPlansAndForcedFilter(t *testing.T) {
	uc := newTestUsecase(t)
	ctx := context.Background()

	if _, err := uc.SetApplicability(ctx, SetApplicabilityInput{
		PlanID:         1,
		BusinessDomain: "general",
		Applicability:  "mandatory",
	}); err != nil {
		t.Fatalf("unexpected error setting applicability: %v", err)
	}

	relevant, err := uc.GetRelevantPlans(ctx, 1, analytic.DomainGeneral, nil)
	if err != nil {
		t.Fatalf("unexpected error getting relevant plans: %v", err)
	}
	if len(relevant) != 2 {
		t.Fatalf("expected 2 relevant plans, got %d", len(relevant))
	}

	// Forced plans are dropped from the proposal (Odoo forced_plans behavior).
	relevant, err = uc.GetRelevantPlans(ctx, 1, analytic.DomainGeneral, []int64{1})
	if err != nil {
		t.Fatalf("unexpected error getting relevant plans: %v", err)
	}
	if len(relevant) != 1 || relevant[0].ID != 2 {
		t.Fatalf("expected only plan 2 after forcing plan 1, got %+v", relevant)
	}
}

func TestUsecase_ReconcileLineDistribution(t *testing.T) {
	uc := newTestUsecase(t)

	moveLineID := int64(77)
	date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	base := &analytic.AnalyticLine{
		Name:         "Split me",
		Date:         date,
		Amount:       1000,
		UnitAmount:   2,
		UserID:       1,
		CompanyID:    1,
		Currency:     "USD",
		MoveLineID:   &moveLineID,
		Source:       analytic.SourceInvoice,
	}

	lines, err := uc.ReconcileLineDistribution(base, analytic.AnalyticDistribution{"1": 50, "2": 50})
	if err != nil {
		t.Fatalf("unexpected error splitting distribution: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 split lines, got %d", len(lines))
	}
	if lines[0].Amount != 500 || lines[1].Amount != 500 {
		t.Fatalf("expected split amounts 500/500, got %v/%v", lines[0].Amount, lines[1].Amount)
	}
	if lines[0].MoveLineID == nil || *lines[0].MoveLineID != 77 {
		t.Fatalf("expected move line reference preserved on split lines")
	}
	if lines[0].Distribution["1"] != 50 || len(lines[0].Distribution) != 1 {
		t.Fatalf("expected single-key distribution snapshot, got %+v", lines[0].Distribution)
	}

	// Zero-amount and empty distributions keep the base line untouched.
	empty, err := uc.ReconcileLineDistribution(base, nil)
	if err != nil {
		t.Fatalf("unexpected error for empty distribution: %v", err)
	}
	if len(empty) != 1 || empty[0].Amount != 1000 {
		t.Fatalf("expected single unsplit line, got %+v", empty)
	}

	_, err = uc.ReconcileLineDistribution(base, analytic.AnalyticDistribution{"garbage": 100})
	if !isAppErr(err, platformerrors.CodeValidation) {
		t.Fatalf("expected validation error for invalid key, got %v", err)
	}
}

func TestUsecase_ValidateDistribution(t *testing.T) {
	uc := newTestUsecase(t)
	ctx := context.Background()

	// Plan 1 (Project) is mandatory in "general" for company 1.
	if _, err := uc.SetApplicability(ctx, SetApplicabilityInput{
		PlanID:         1,
		BusinessDomain: "general",
		Applicability:  "mandatory",
	}); err != nil {
		t.Fatalf("unexpected error setting applicability: %v", err)
	}

	// Account 1 belongs to plan 1 -> fully covered.
	err := uc.ValidateDistribution(ctx, 1, analytic.DomainGeneral,
		[]analytic.AnalyticLine{{Distribution: analytic.AnalyticDistribution{"1": 100}}})
	if err != nil {
		t.Fatalf("expected valid distribution, got %v", err)
	}

	// Account 2 belongs to plan 2 -> plan 1 uncovered -> error.
	err = uc.ValidateDistribution(ctx, 1, analytic.DomainGeneral,
		[]analytic.AnalyticLine{{Distribution: analytic.AnalyticDistribution{"2": 100}}})
	if !isAppErr(err, platformerrors.CodeValidation) {
		t.Fatalf("expected validation error for uncovered mandatory plan, got %v", err)
	}

	// A split across both plans covers plan 1 with 50 only -> error.
	err = uc.ValidateDistribution(ctx, 1, analytic.DomainGeneral,
		[]analytic.AnalyticLine{{Distribution: analytic.AnalyticDistribution{"1": 50, "2": 50}}})
	if !isAppErr(err, platformerrors.CodeValidation) {
		t.Fatalf("expected validation error for partial mandatory coverage, got %v", err)
	}
}

func TestUsecase_AutoCompleteDistribution(t *testing.T) {
	uc := newTestUsecase(t)
	ctx := context.Background()

	if _, err := uc.CreateDistributionModel(ctx, CreateDistributionModelInput{
		Sequence:     5,
		Distribution: analytic.AnalyticDistribution{"2": 100},
	}); err != nil {
		t.Fatalf("unexpected error creating distribution model: %v", err)
	}

	merged, changed, err := uc.AutoCompleteDistribution(ctx, nil, nil, 1, analytic.AnalyticDistribution{"3": 100})
	if err != nil {
		t.Fatalf("unexpected error auto-completing distribution: %v", err)
	}
	if !changed {
		t.Fatal("expected distribution to be changed")
	}
	if merged["2"] != 100 || merged["3"] != 100 {
		t.Fatalf("expected merged distribution, got %+v", merged)
	}

	// Unchanged when the matched rule is already applied.
	merged, changed, err = uc.AutoCompleteDistribution(ctx, nil, nil, 1, analytic.AnalyticDistribution{"2": 100})
	if err != nil {
		t.Fatalf("unexpected error auto-completing distribution: %v", err)
	}
	if changed || merged["2"] != 100 {
		t.Fatalf("expected no-op merge, got changed=%v merged=%+v", changed, merged)
	}
}

func TestUsecase_CreateLinesFromMoveLine(t *testing.T) {
	uc := newTestUsecase(t)
	ctx := context.Background()

	date := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	lines, err := uc.CreateLinesFromMoveLine(ctx, MoveLineAnalyticInput{
		MoveLineID:       900,
		GeneralAccountID: 1010,
		Name:             "Invoice line analytic",
		Date:             date,
		UserID:           1,
		CompanyID:        1,
		Amount:           2000,
		UnitAmount:       4,
		Source:           string(analytic.SourceInvoice),
		Distribution:     analytic.AnalyticDistribution{"1": 25, "2": 75},
	})
	if err != nil {
		t.Fatalf("unexpected error creating lines from move line: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 analytic lines, got %d", len(lines))
	}

	byAccount := map[int64]float64{}
	for _, l := range lines {
		byAccount[l.AccountID] = l.Amount
	}
	if byAccount[1] != 500 || byAccount[2] != 1500 {
		t.Fatalf("expected amounts 500/1500, got %+v", byAccount)
	}
	for _, l := range lines {
		if l.MoveLineID == nil || *l.MoveLineID != 900 {
			t.Fatalf("expected move line reference on analytic line")
		}
	}

	// The split is idempotent: totals derive from the distribution snapshot.
	stored, err := uc.ListLinesByMoveLine(ctx, 900)
	if err != nil {
		t.Fatalf("unexpected error listing lines by move line: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected stored 2 lines, got %d", len(stored))
	}
}

func TestUsecase_RegisterManualLineAndBalance(t *testing.T) {
	uc := newTestUsecase(t)
	ctx := context.Background()

	line, err := uc.RegisterManualLine(ctx, CreateLineInput{
		Name:      "Manual entry",
		Date:      time.Now().UTC(),
		Amount:    -120,
		UserID:    2,
		CompanyID: 1,
		AccountID: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error registering manual line: %v", err)
	}
	if line.Source != analytic.SourceManual {
		t.Fatalf("expected manual source, got %s", line.Source)
	}

	balance, err := uc.GetAccountBalance(ctx, 1, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error computing balance: %v", err)
	}
	if balance.Debit != 120 || balance.Balance != -120 {
		t.Fatalf("expected debit 120 balance -120, got %+v", balance)
	}

	page, err := uc.ListLines(ctx, nil, pagination.PageRequest{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing lines: %v", err)
	}
	if page.TotalItems != 1 {
		t.Fatalf("expected 1 line, got %d", page.TotalItems)
	}

	linesByFilter, err := uc.repoFiltered(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error filtering line: %v", err)
	}
	if len(linesByFilter) != 1 {
		t.Fatalf("expected line filtered by account_id, got %d", len(linesByFilter))
	}
}

func (uc *UseCase) repoFiltered(ctx context.Context, accountID int64) ([]analytic.AnalyticLine, error) {
	page, err := uc.ListLines(ctx,
		filter.NewFilter(filter.Criterion{Field: "account_id", Operator: "eq", Value: accountID}),
		pagination.PageRequest{Limit: 10},
	)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func TestUsecase_MoveLineIntegrationHelper(t *testing.T) {
	partnerID := int64(9)
	move := accounting.AccountMoveLine{
		ID:        42,
		AccountID: 1001,
		PartnerID: &partnerID,
		Name:      "Integration",
		Quantity:  3,
		Balance:   750,
	}
	in := MoveLineToAnalyticInput(move, 1, 1, analytic.AnalyticDistribution{"1": 100})
	if in.MoveLineID != 42 || in.GeneralAccountID != 1001 || in.Amount != 750 {
		t.Fatalf("unexpected move line mapping: %+v", in)
	}
	if in.PartnerID == nil || *in.PartnerID != 9 {
		t.Fatalf("expected partner reference preserved, got %+v", in.PartnerID)
	}
}

func TestUsecase_PlanAndAccountValidation(t *testing.T) {
	uc := newTestUsecase(t)
	ctx := context.Background()

	if _, err := uc.CreatePlan(ctx, CreatePlanInput{Name: "  "}); !isAppErr(err, platformerrors.CodeValidation) {
		t.Fatalf("expected validation error for blank plan name, got %v", err)
	}

	if _, err := uc.CreateAccount(ctx, CreateAccountInput{Name: "X", PlanID: 9999}); !isAppErr(err, platformerrors.CodeValidation) {
		t.Fatalf("expected validation error for unknown plan, got %v", err)
	}

	if _, err := uc.CreateDistributionModel(ctx, CreateDistributionModelInput{
		Distribution: analytic.AnalyticDistribution{"1": 50},
	}); !isAppErr(err, platformerrors.CodeValidation) {
		t.Fatalf("expected validation error for 50%% distribution model, got %v", err)
	}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func isAppErr(err error, code string) bool {
	var appErr *platformerrors.AppError
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code == code
}