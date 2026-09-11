package helpdesk

import (
	"context"
)

type Repository interface {
	// Team & Stage
	GetTeam(ctx context.Context, id int64) (*Team, error)
	ListTeams(ctx context.Context, companyID int64) ([]Team, error)
	GetStage(ctx context.Context, id int64) (*Stage, error)
	ListStages(ctx context.Context, teamID int64) ([]Stage, error)

	// Ticket
	GetTicket(ctx context.Context, id int64) (*Ticket, error)
	GetTicketByNumber(ctx context.Context, number string) (*Ticket, error)
	CreateTicket(ctx context.Context, ticket *Ticket) error
	UpdateTicket(ctx context.Context, ticket *Ticket) error
	ListTickets(ctx context.Context, teamID int64, stageID int64) ([]Ticket, error)

	// SLA
	GetSLAPolicy(ctx context.Context, id int64) (*SLAPolicy, error)
	ListSLAPolicies(ctx context.Context, teamID int64) ([]SLAPolicy, error)

	// Knowledge
	GetArticle(ctx context.Context, id int64) (*KnowledgeArticle, error)
	GetArticleBySlug(ctx context.Context, slug string) (*KnowledgeArticle, error)
	ListArticles(ctx context.Context, categoryID int64, includeInternal bool) ([]KnowledgeArticle, error)
	ListCategories(ctx context.Context, companyID int64) ([]KnowledgeCategory, error)
}

type Service interface {
	// Ticket Management
	CreateTicket(ctx context.Context, ticket *Ticket) (*Ticket, error)
	AssignTicket(ctx context.Context, ticketID int64, userID int64) error
	MoveToStage(ctx context.Context, ticketID int64, stageID int64) error
	CloseTicket(ctx context.Context, ticketID int64) error

	// SLA Engine
	CheckSLABreach(ctx context.Context, ticketID int64) error
	CalculateSLA(ctx context.Context, ticket *Ticket) (*SLAStatus, error)

	// Knowledge Base
	TrackArticleView(ctx context.Context, articleID int64) error
	VoteArticleHelpful(ctx context.Context, articleID int64) error
}
