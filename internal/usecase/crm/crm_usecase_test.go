package crmusecase

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"cashflow_backend/internal/adapters/storage/crm"
	domaincrm "cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/sale"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	saleusecase "cashflow_backend/internal/usecase/sale"
)

// Mock Partner Service
type mockPartnerService struct {
	createdPartners []*partner.Partner
	lastID          int64
}

func (m *mockPartnerService) CreatePartner(ctx context.Context, in partnerusecase.CreatePartnerInput) (*partner.Partner, error) {
	m.lastID++
	p := &partner.Partner{
		ID:         m.lastID,
		Name:       in.Name,
		Email:      in.Email,
		Phone:      in.Phone,
		IsCustomer: true,
	}
	m.createdPartners = append(m.createdPartners, p)
	return p, nil
}

func (m *mockPartnerService) GetPartner(ctx context.Context, id int64) (*partner.Partner, error) {
	for _, p := range m.createdPartners {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

// Mock Sale Service
type mockSaleService struct {
	createdOrders []*sale.SaleOrder
	lastID        int64
}

func (m *mockSaleService) CreateOrder(ctx context.Context, in saleusecase.CreateSaleOrderInput) (*sale.SaleOrder, error) {
	m.lastID++
	order := &sale.SaleOrder{
		ID:        m.lastID,
		Name:      "SO/2026/00001",
		PartnerID: in.PartnerID,
		Note:      in.Note,
		Currency:  in.Currency,
		State:     sale.OrderStateDraft,
	}
	m.createdOrders = append(m.createdOrders, order)
	return order, nil
}

func TestUseCase_FullLifecycle(t *testing.T) {
	repo := crmstorage.NewMemoryRepo()
	mockPartner := &mockPartnerService{}
	mockSale := &mockSaleService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	uc := New(repo, mockPartner, mockSale, logger)
	ctx := context.Background()

	// 1. Create Inbound Lead
	lead, err := uc.CreateLead(ctx, CreateLeadInput{
		Name:            "Inbound ERP Request",
		Type:            "lead",
		ContactName:     "Alice Smith",
		PartnerName:     "Acme Corp",
		EmailFrom:       "alice@acmecorp.com",
		Phone:           "+123456789",
		ExpectedRevenue: 60000.0,
	})
	if err != nil {
		t.Fatalf("unexpected error creating lead: %v", err)
	}
	if lead.Type != domaincrm.LeadTypeLead {
		t.Fatalf("expected type lead, got %s", lead.Type)
	}
	if lead.ProratedRevenue != 6000.0 { // 10% default for New stage
		t.Fatalf("expected prorated revenue 6000, got %f", lead.ProratedRevenue)
	}

	// 2. Convert to Opportunity with auto-partner creation
	converted, err := uc.ConvertLead(ctx, lead.ID, ConvertLeadInput{
		CreatePartner: true,
	})
	if err != nil {
		t.Fatalf("unexpected error converting lead: %v", err)
	}
	if converted.Type != domaincrm.LeadTypeOpportunity {
		t.Fatalf("expected type opportunity, got %s", converted.Type)
	}
	if converted.PartnerID == nil {
		t.Fatal("expected auto-created partner ID to be linked")
	}
	if len(mockPartner.createdPartners) != 1 {
		t.Fatalf("expected 1 partner to be created, got %d", len(mockPartner.createdPartners))
	}
	if mockPartner.createdPartners[0].Name != "Acme Corp" {
		t.Fatalf("expected partner name Acme Corp, got %s", mockPartner.createdPartners[0].Name)
	}

	// 3. Update Opportunity (Moving to Proposition stage with 70% probability)
	prob := 70.0
	stageProposition := int64(3)
	updated, err := uc.UpdateLead(ctx, converted.ID, UpdateLeadInput{
		StageID:     &stageProposition,
		Probability: &prob,
	})
	if err != nil {
		t.Fatalf("unexpected error updating lead: %v", err)
	}
	if updated.ProratedRevenue != 42000.0 {
		t.Fatalf("expected prorated revenue 42000, got %f", updated.ProratedRevenue)
	}

	// 4. Mark as Won and spawn Sale Order
	wonLead, so, err := uc.MarkLeadWon(ctx, updated.ID, MarkWonInput{
		CreateSaleOrder: true,
	})
	if err != nil {
		t.Fatalf("unexpected error marking lead won: %v", err)
	}
	if wonLead.Probability != 100.0 {
		t.Fatalf("expected 100 probability, got %f", wonLead.Probability)
	}
	if wonLead.ProratedRevenue != 60000.0 {
		t.Fatalf("expected 60000 prorated revenue, got %f", wonLead.ProratedRevenue)
	}
	if so == nil {
		t.Fatal("expected draft sale order to be generated")
	}
	if len(mockSale.createdOrders) != 1 {
		t.Fatalf("expected 1 sale order created, got %d", len(mockSale.createdOrders))
	}

	// 5. Check Pipeline View
	pipeline, err := uc.GetPipelineView(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error fetching pipeline: %v", err)
	}
	if len(pipeline) != 4 {
		t.Fatalf("expected 4 pipeline stages, got %d", len(pipeline))
	}
	// Won stage is 4th
	wonStageData := pipeline[3]
	if wonStageData.TotalOpportunities != 1 {
		t.Fatalf("expected 1 opportunity in won stage, got %d", wonStageData.TotalOpportunities)
	}

	// 6. Check Stats
	stats, err := uc.GetCRMStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error fetching stats: %v", err)
	}
	if stats.WonCount != 1 {
		t.Fatalf("expected won count 1, got %d", stats.WonCount)
	}
	if stats.WinRate != 100.0 {
		t.Fatalf("expected 100 win rate, got %f", stats.WinRate)
	}
}

func TestUseCase_MarkLost(t *testing.T) {
	repo := crmstorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := New(repo, nil, nil, logger)
	ctx := context.Background()

	lead, err := uc.CreateLead(ctx, CreateLeadInput{
		Name:            "Budget inquiry",
		Type:            "opportunity",
		ExpectedRevenue: 10000.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lostLead, err := uc.MarkLeadLost(ctx, lead.ID, MarkLostInput{
		LostReasonID: 1, // Too expensive
		LostFeedback: "Price is over their budget",
	})
	if err != nil {
		t.Fatalf("unexpected error marking lost: %v", err)
	}
	if lostLead.Active != false {
		t.Fatalf("expected active to be false")
	}
	if lostLead.LostReasonID == nil || *lostLead.LostReasonID != 1 {
		t.Fatalf("expected lost reason 1")
	}

	stats, err := uc.GetCRMStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting stats: %v", err)
	}
	if stats.LostCount != 1 {
		t.Fatalf("expected lost count 1, got %d", stats.LostCount)
	}
	if len(stats.TopLostReasons) == 0 || stats.TopLostReasons[0].ReasonID != 1 {
		t.Fatalf("expected top lost reason to be 1")
	}
}
