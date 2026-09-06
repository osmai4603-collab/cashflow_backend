package crmstorage

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_StagesAndReasons(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	// Initial seeds
	stages, err := repo.ListStages(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing stages: %v", err)
	}
	if len(stages) != 4 {
		t.Fatalf("expected 4 seeded stages, got %d", len(stages))
	}

	initialStage, err := repo.GetInitialStage(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting initial stage: %v", err)
	}
	if initialStage.Name != "New" {
		t.Fatalf("expected initial stage New, got %s", initialStage.Name)
	}

	wonStage, err := repo.GetWonStage(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting won stage: %v", err)
	}
	if wonStage.Name != "Won" || !wonStage.IsWon {
		t.Fatalf("expected won stage Won, got %s", wonStage.Name)
	}

	reasons, err := repo.ListLostReasons(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing lost reasons: %v", err)
	}
	if len(reasons) != 4 {
		t.Fatalf("expected 4 seeded lost reasons, got %d", len(reasons))
	}
}

func TestMemoryRepo_LeadCRUDAndPipeline(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	lead := &crm.Lead{
		Name:            "Enterprise Cloud Deal",
		Type:            crm.LeadTypeOpportunity,
		StageID:         1,
		ExpectedRevenue: 40000.0,
		Probability:     25.0,
		Priority:        crm.PriorityHigh,
	}

	err := repo.CreateLead(ctx, lead)
	if err != nil {
		t.Fatalf("unexpected error creating lead: %v", err)
	}
	if lead.ID == 0 {
		t.Fatal("expected assigned lead ID")
	}
	if lead.ProratedRevenue != 10000.0 {
		t.Fatalf("expected prorated revenue 10000, got %f", lead.ProratedRevenue)
	}

	// Fetch
	fetched, err := repo.GetLeadByID(ctx, lead.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching lead: %v", err)
	}
	if fetched.Name != lead.Name {
		t.Fatalf("expected name %s, got %s", lead.Name, fetched.Name)
	}

	// Pipeline View
	pipeline, err := repo.GetPipeline(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error getting pipeline: %v", err)
	}
	if len(pipeline) != 4 {
		t.Fatalf("expected 4 pipeline stage cards, got %d", len(pipeline))
	}
	// First stage (New) should have 1 opportunity
	if pipeline[0].TotalOpportunities != 1 {
		t.Fatalf("expected 1 opportunity in New stage, got %d", pipeline[0].TotalOpportunities)
	}
	if pipeline[0].TotalExpectedRevenue != 40000.0 {
		t.Fatalf("expected 40000 expected revenue, got %f", pipeline[0].TotalExpectedRevenue)
	}
	if pipeline[0].TotalProratedRevenue != 10000.0 {
		t.Fatalf("expected 10000 prorated revenue, got %f", pipeline[0].TotalProratedRevenue)
	}

	// Stats View
	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting stats: %v", err)
	}
	if stats.TotalOpportunities != 1 {
		t.Fatalf("expected 1 opportunity in stats, got %d", stats.TotalOpportunities)
	}
	if stats.TotalExpectedRevenue != 40000.0 {
		t.Fatalf("expected 40000 total expected revenue, got %f", stats.TotalExpectedRevenue)
	}

	// Filtering & Pagination
	f := filter.NewFilter().Add("type", filter.OpEqual, "opportunity")
	pageRes, err := repo.ListLeads(ctx, f, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error listing leads: %v", err)
	}
	if pageRes.TotalItems != 1 {
		t.Fatalf("expected 1 item, got %d", pageRes.TotalItems)
	}
}
