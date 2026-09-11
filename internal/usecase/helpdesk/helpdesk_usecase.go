package helpdeskusecase

import (
	"context"
	"cashflow_backend/internal/domain/helpdesk"
)

type UseCase struct {
	repo helpdesk.Repository
}

func New(repo helpdesk.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) CreateTicket(ctx context.Context, ticket *helpdesk.Ticket) (*helpdesk.Ticket, error) {
	return nil, nil
}

func (u *UseCase) AssignTicket(ctx context.Context, ticketID int64, userID int64) error {
	return nil
}

func (u *UseCase) MoveToStage(ctx context.Context, ticketID int64, stageID int64) error {
	return nil
}

func (u *UseCase) CloseTicket(ctx context.Context, ticketID int64) error {
	return nil
}

func (u *UseCase) ListTickets(ctx context.Context) ([]helpdesk.Ticket, error) {
	return nil, nil
}

func (u *UseCase) GetTicket(ctx context.Context, id int64) (*helpdesk.Ticket, error) {
	return nil, nil
}

func (u *UseCase) ReplyToTicket(ctx context.Context, ticketID int64, body string) error {
	return nil
}

func (u *UseCase) ListArticles(ctx context.Context) ([]helpdesk.KnowledgeArticle, error) {
	return nil, nil
}

func (u *UseCase) GetArticleBySlug(ctx context.Context, slug string) (*helpdesk.KnowledgeArticle, error) {
	return nil, nil
}

func (u *UseCase) CreateArticle(ctx context.Context, a *helpdesk.KnowledgeArticle) (*helpdesk.KnowledgeArticle, error) {
	return nil, nil
}

func (u *UseCase) UpdateArticle(ctx context.Context, id int64, a *helpdesk.KnowledgeArticle) (*helpdesk.KnowledgeArticle, error) {
	return nil, nil
}

func (u *UseCase) CheckSLABreach(ctx context.Context, ticketID int64) error {
	return nil
}

func (u *UseCase) CalculateSLA(ctx context.Context, ticket *helpdesk.Ticket) (*helpdesk.SLAStatus, error) {
	return nil, nil
}

func (u *UseCase) TrackArticleView(ctx context.Context, articleID int64) error {
	return nil
}

func (u *UseCase) VoteArticleHelpful(ctx context.Context, articleID int64) error {
	return nil
}
