package helpdeskusecase

import (
	"context"
	"testing"
	"time"

	helpdeskstorage "cashflow_backend/internal/adapters/storage/helpdesk"
	"cashflow_backend/internal/domain/helpdesk"
)

func TestHelpdeskUseCase_TicketSLAAndKnowledge(t *testing.T) {
	repo := helpdeskstorage.NewMemoryRepo()
	repo.SeedTeam(&helpdesk.Team{ID: 1, Name: "Support", CompanyID: 1, Active: true})
	repo.SeedStage(&helpdesk.Stage{ID: 2, TeamID: 1, Name: "New", Sequence: 1, CompanyID: 1})
	repo.SeedStage(&helpdesk.Stage{ID: 3, TeamID: 1, Name: "Solved", Sequence: 2, IsClosed: true, CompanyID: 1})
	repo.SeedSLAPolicy(&helpdesk.SLAPolicy{ID: 4, Name: "Standard", TeamID: 1, Priority: "1", MaxHoursFirstResp: 4, MaxHoursResolution: 24, Active: true, CompanyID: 1})
	uc := New(repo)
	ctx := context.Background()

	ticket, err := uc.CreateTicket(ctx, &helpdesk.Ticket{Name: "Cannot login", Description: "Customer cannot sign in", TeamID: 1, StageID: 2, Priority: "1", PartnerEmail: "customer@example.com"})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if ticket.ID == 0 || ticket.Number == "" || ticket.CompanyID != 1 {
		t.Fatalf("unexpected ticket: %#v", ticket)
	}
	if err := uc.ReplyToTicket(ctx, ticket.ID, "We are investigating"); err != nil {
		t.Fatalf("reply: %v", err)
	}
	status, err := uc.CalculateSLA(ctx, ticket)
	if err != nil || status.ReachedFirstResp != true || status.DeadlineResolution.Before(time.Now()) {
		t.Fatalf("unexpected SLA status: %v, %#v", err, status)
	}
	if err := uc.CloseTicket(ctx, ticket.ID); err != nil {
		t.Fatalf("close ticket: %v", err)
	}

	article, err := uc.CreateArticle(ctx, &helpdesk.KnowledgeArticle{CategoryID: 1, Title: "Login help", ContentHTML: "<p>Reset your password.</p>"})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}
	if article.Slug != "login-help" {
		t.Fatalf("unexpected article slug: %q", article.Slug)
	}
	if err := uc.TrackArticleView(ctx, article.ID); err != nil {
		t.Fatalf("track article view: %v", err)
	}
}
