package helpdeskusecase

import (
	"context"
	"strings"
	"time"

	"cashflow_backend/internal/domain/helpdesk"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo helpdesk.Repository
}

func New(repo helpdesk.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) CreateTicket(ctx context.Context, ticket *helpdesk.Ticket) (*helpdesk.Ticket, error) {
	if ticket == nil {
		return nil, platformerrors.Validation("ticket is required", nil)
	}
	if ticket.CompanyID == 0 {
		ticket.CompanyID = 1
	}
	if ticket.Priority == "" {
		ticket.Priority = "1"
	}
	if err := ticket.Validate(); err != nil {
		return nil, err
	}
	if _, err := u.repo.GetTeam(ctx, ticket.TeamID); err != nil {
		return nil, err
	}
	if _, err := u.repo.GetStage(ctx, ticket.StageID); err != nil {
		return nil, err
	}
	ticket.CreatedAt = time.Now().UTC()
	ticket.UpdatedAt = ticket.CreatedAt
	if err := u.repo.CreateTicket(ctx, ticket); err != nil {
		return nil, err
	}
	if ticket.Number == "" {
		ticket.GenerateNumber(ticket.ID)
		if err := u.repo.UpdateTicket(ctx, ticket); err != nil {
			return nil, err
		}
	}
	return ticket, nil
}

func (u *UseCase) AssignTicket(ctx context.Context, ticketID int64, userID int64) error {
	ticket, err := u.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return helpdesk.ErrTicketNotFound
	}
	if userID <= 0 {
		return platformerrors.Validation("user_id is required", nil)
	}
	ticket.ActionAssign(userID)
	return u.repo.UpdateTicket(ctx, ticket)
}

func (u *UseCase) MoveToStage(ctx context.Context, ticketID int64, stageID int64) error {
	ticket, err := u.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return helpdesk.ErrTicketNotFound
	}
	if _, err := u.repo.GetStage(ctx, stageID); err != nil {
		return err
	}
	ticket.StageID = stageID
	ticket.UpdatedAt = time.Now().UTC()
	return u.repo.UpdateTicket(ctx, ticket)
}

func (u *UseCase) CloseTicket(ctx context.Context, ticketID int64) error {
	ticket, err := u.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return helpdesk.ErrTicketNotFound
	}
	stages, err := u.repo.ListStages(ctx, ticket.TeamID)
	if err != nil {
		return err
	}
	closedStage := int64(0)
	for _, stage := range stages {
		if stage.IsClosed {
			closedStage = stage.ID
			break
		}
	}
	if closedStage == 0 {
		return platformerrors.Validation("closed stage is not configured", nil)
	}
	ticket.ActionClose(closedStage)
	return u.repo.UpdateTicket(ctx, ticket)
}

func (u *UseCase) ListTickets(ctx context.Context) ([]helpdesk.Ticket, error) {
	tickets, err := u.repo.ListTickets(ctx, 0, 0)
	return tickets, err
}

func (u *UseCase) GetTicket(ctx context.Context, id int64) (*helpdesk.Ticket, error) {
	if id <= 0 {
		return nil, platformerrors.Validation("ticket_id is required", nil)
	}
	ticket, err := u.repo.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, helpdesk.ErrTicketNotFound
	}
	return ticket, nil
}

func (u *UseCase) ReplyToTicket(ctx context.Context, ticketID int64, body string) error {
	if strings.TrimSpace(body) == "" {
		return platformerrors.Validation("reply body is required", nil)
	}
	ticket, err := u.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return helpdesk.ErrTicketNotFound
	}
	ticket.ActionRespond()
	return u.repo.UpdateTicket(ctx, ticket)
}

func (u *UseCase) ListArticles(ctx context.Context) ([]helpdesk.KnowledgeArticle, error) {
	articles, err := u.repo.ListArticles(ctx, 0, false)
	return articles, err
}

func (u *UseCase) GetArticleBySlug(ctx context.Context, slug string) (*helpdesk.KnowledgeArticle, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, platformerrors.Validation("article slug is required", nil)
	}
	article, err := u.repo.GetArticleBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, helpdesk.ErrArticleNotFound
	}
	return article, nil
}

func (u *UseCase) CreateArticle(ctx context.Context, a *helpdesk.KnowledgeArticle) (*helpdesk.KnowledgeArticle, error) {
	if a == nil {
		return nil, platformerrors.Validation("article is required", nil)
	}
	if a.CompanyID == 0 {
		a.CompanyID = 1
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	if err := u.repo.CreateArticle(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *UseCase) UpdateArticle(ctx context.Context, id int64, a *helpdesk.KnowledgeArticle) (*helpdesk.KnowledgeArticle, error) {
	if a == nil || id <= 0 {
		return nil, platformerrors.Validation("article and id are required", nil)
	}
	existing, err := u.repo.GetArticle(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, helpdesk.ErrArticleNotFound
	}
	a.ID = id
	a.CompanyID = existing.CompanyID
	a.CreatedAt = existing.CreatedAt
	a.UpdatedAt = time.Now().UTC()
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.UpdateArticle(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *UseCase) CheckSLABreach(ctx context.Context, ticketID int64) error {
	ticket, err := u.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return helpdesk.ErrTicketNotFound
	}
	status, err := u.CalculateSLA(ctx, ticket)
	if err != nil {
		return err
	}
	ticket.SLABreach = status.IsBreached
	return u.repo.UpdateTicket(ctx, ticket)
}

func (u *UseCase) CalculateSLA(ctx context.Context, ticket *helpdesk.Ticket) (*helpdesk.SLAStatus, error) {
	if ticket == nil {
		return nil, platformerrors.Validation("ticket is required", nil)
	}
	policies, err := u.repo.ListSLAPolicies(ctx, ticket.TeamID)
	if err != nil {
		return nil, err
	}
	var selected *helpdesk.SLAPolicy
	for i := range policies {
		if policies[i].Active && policies[i].Priority == ticket.Priority {
			selected = &policies[i]
			break
		}
	}
	if selected == nil {
		return nil, helpdesk.ErrSLAPolicyNotFound
	}
	status := &helpdesk.SLAStatus{TicketID: ticket.ID, PolicyID: selected.ID, ReachedFirstResp: ticket.FirstResponseAt != nil, ReachedResolution: ticket.ClosedAt != nil}
	first := ticket.CreatedAt.Add(time.Duration(selected.MaxHoursFirstResp * float64(time.Hour)))
	resolution := ticket.CreatedAt.Add(time.Duration(selected.MaxHoursResolution * float64(time.Hour)))
	status.DeadlineFirstResp = &first
	status.DeadlineResolution = &resolution
	status.CalculateBreach()
	return status, nil
}

func (u *UseCase) TrackArticleView(ctx context.Context, articleID int64) error {
	article, err := u.repo.GetArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if article == nil {
		return helpdesk.ErrArticleNotFound
	}
	article.ViewCount++
	article.UpdatedAt = time.Now().UTC()
	return u.repo.UpdateArticle(ctx, article)
}

func (u *UseCase) VoteArticleHelpful(ctx context.Context, articleID int64) error {
	article, err := u.repo.GetArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if article == nil {
		return helpdesk.ErrArticleNotFound
	}
	article.HelpfulCount++
	article.UpdatedAt = time.Now().UTC()
	return u.repo.UpdateArticle(ctx, article)
}
