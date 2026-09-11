package helpdeskstorage

import (
	"context"
	"cashflow_backend/internal/domain/helpdesk"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) GetTeam(ctx context.Context, id int64) (*helpdesk.Team, error) { return nil, nil }
func (r *PostgresRepo) ListTeams(ctx context.Context, companyID int64) ([]helpdesk.Team, error) { return nil, nil }
func (r *PostgresRepo) GetStage(ctx context.Context, id int64) (*helpdesk.Stage, error) { return nil, nil }
func (r *PostgresRepo) ListStages(ctx context.Context, teamID int64) ([]helpdesk.Stage, error) { return nil, nil }
func (r *PostgresRepo) GetTicket(ctx context.Context, id int64) (*helpdesk.Ticket, error) { return nil, nil }
func (r *PostgresRepo) GetTicketByNumber(ctx context.Context, number string) (*helpdesk.Ticket, error) { return nil, nil }
func (r *PostgresRepo) CreateTicket(ctx context.Context, ticket *helpdesk.Ticket) error { return nil }
func (r *PostgresRepo) UpdateTicket(ctx context.Context, ticket *helpdesk.Ticket) error { return nil }
func (r *PostgresRepo) ListTickets(ctx context.Context, teamID int64, stageID int64) ([]helpdesk.Ticket, error) { return nil, nil }
func (r *PostgresRepo) GetSLAPolicy(ctx context.Context, id int64) (*helpdesk.SLAPolicy, error) { return nil, nil }
func (r *PostgresRepo) ListSLAPolicies(ctx context.Context, teamID int64) ([]helpdesk.SLAPolicy, error) { return nil, nil }
func (r *PostgresRepo) GetArticle(ctx context.Context, id int64) (*helpdesk.KnowledgeArticle, error) { return nil, nil }
func (r *PostgresRepo) GetArticleBySlug(ctx context.Context, slug string) (*helpdesk.KnowledgeArticle, error) { return nil, nil }
func (r *PostgresRepo) ListArticles(ctx context.Context, categoryID int64, includeInternal bool) ([]helpdesk.KnowledgeArticle, error) { return nil, nil }
func (r *PostgresRepo) ListCategories(ctx context.Context, companyID int64) ([]helpdesk.KnowledgeCategory, error) { return nil, nil }
