package crm

import (
	"testing"
)

func TestLead_ComputeProratedRevenue(t *testing.T) {
	lead := &Lead{
		Name:            "Acme ERP Deal",
		ExpectedRevenue: 50000.0,
		Probability:     70.0,
	}

	lead.ComputeProratedRevenue()
	expected := 35000.0
	if lead.ProratedRevenue != expected {
		t.Fatalf("expected prorated revenue %f, got %f", expected, lead.ProratedRevenue)
	}

	// Boundary condition: 0 revenue
	lead.ExpectedRevenue = 0
	lead.ComputeProratedRevenue()
	if lead.ProratedRevenue != 0 {
		t.Fatalf("expected 0, got %f", lead.ProratedRevenue)
	}

	// Boundary condition: 100% probability
	lead.ExpectedRevenue = 10000
	lead.Probability = 100
	lead.ComputeProratedRevenue()
	if lead.ProratedRevenue != 10000 {
		t.Fatalf("expected 10000, got %f", lead.ProratedRevenue)
	}
}

func TestLead_ActionConvert(t *testing.T) {
	lead := &Lead{
		ID:      1,
		Name:    "Website Inquiry",
		Type:    LeadTypeLead,
		StageID: 1,
	}

	partnerID := int64(10)
	err := lead.ActionConvert(2, &partnerID)
	if err != nil {
		t.Fatalf("unexpected error on conversion: %v", err)
	}

	if lead.Type != LeadTypeOpportunity {
		t.Fatalf("expected type opportunity, got %s", lead.Type)
	}
	if lead.StageID != 2 {
		t.Fatalf("expected stage 2, got %d", lead.StageID)
	}
	if lead.PartnerID == nil || *lead.PartnerID != 10 {
		t.Fatalf("expected partner ID 10")
	}
	if lead.DateConversion == nil {
		t.Fatalf("expected DateConversion to be set")
	}

	// Attempting to convert again must return error
	err = lead.ActionConvert(3, nil)
	if err == nil {
		t.Fatalf("expected conflict error when converting already converted opportunity")
	}
}

func TestLead_ActionMarkWon(t *testing.T) {
	lead := &Lead{
		ID:              1,
		Name:            "ERP Contract",
		Type:            LeadTypeOpportunity,
		StageID:         2,
		ExpectedRevenue: 25000.0,
		Probability:     60.0,
	}

	err := lead.ActionMarkWon(4)
	if err != nil {
		t.Fatalf("unexpected error marking won: %v", err)
	}

	if lead.StageID != 4 {
		t.Fatalf("expected stage 4, got %d", lead.StageID)
	}
	if lead.Probability != 100.0 {
		t.Fatalf("expected probability 100, got %f", lead.Probability)
	}
	if lead.ProratedRevenue != 25000.0 {
		t.Fatalf("expected prorated revenue 25000, got %f", lead.ProratedRevenue)
	}
	if lead.DateClosed == nil {
		t.Fatalf("expected DateClosed to be set")
	}
}

func TestLead_ActionMarkLost(t *testing.T) {
	lead := &Lead{
		ID:              1,
		Name:            "Competitor Lost Deal",
		Type:            LeadTypeOpportunity,
		StageID:         2,
		ExpectedRevenue: 15000.0,
		Probability:     40.0,
		Active:          true,
	}

	err := lead.ActionMarkLost(1, "Client chose alternative solution")
	if err != nil {
		t.Fatalf("unexpected error marking lost: %v", err)
	}

	if lead.LostReasonID == nil || *lead.LostReasonID != 1 {
		t.Fatalf("expected lost reason 1")
	}
	if lead.LostFeedback != "Client chose alternative solution" {
		t.Fatalf("unexpected feedback: %s", lead.LostFeedback)
	}
	if lead.Probability != 0.0 {
		t.Fatalf("expected probability 0, got %f", lead.Probability)
	}
	if lead.ProratedRevenue != 0.0 {
		t.Fatalf("expected prorated revenue 0, got %f", lead.ProratedRevenue)
	}
	if lead.Active != false {
		t.Fatalf("expected active to be false after being lost")
	}
}

func TestLead_Validate(t *testing.T) {
	// Missing name
	l := &Lead{StageID: 1, Type: LeadTypeLead}
	if err := l.Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}

	// Invalid type
	l = &Lead{Name: "Test", StageID: 1, Type: "invalid"}
	if err := l.Validate(); err == nil {
		t.Fatal("expected error for invalid type")
	}

	// Missing stage
	l = &Lead{Name: "Test", Type: LeadTypeLead, StageID: 0}
	if err := l.Validate(); err == nil {
		t.Fatal("expected error for missing stage")
	}

	// Negative expected revenue
	l = &Lead{Name: "Test", Type: LeadTypeLead, StageID: 1, ExpectedRevenue: -100}
	if err := l.Validate(); err == nil {
		t.Fatal("expected error for negative expected revenue")
	}

	// Valid lead
	l = &Lead{
		Name:            "Valid Deal",
		Type:            LeadTypeLead,
		StageID:         1,
		ExpectedRevenue: 5000,
		Probability:     20,
	}
	if err := l.Validate(); err != nil {
		t.Fatalf("expected valid lead, got: %v", err)
	}
	if l.ProratedRevenue != 1000 {
		t.Fatalf("expected prorated revenue 1000, got %f", l.ProratedRevenue)
	}
}

func TestStage_Validate(t *testing.T) {
	s := &Stage{Name: ""}
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for empty stage name")
	}

	s = &Stage{Name: "Qualified"}
	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestLostReason_Validate(t *testing.T) {
	r := &LostReason{Name: ""}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty reason name")
	}

	r = &LostReason{Name: "Too Expensive"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
