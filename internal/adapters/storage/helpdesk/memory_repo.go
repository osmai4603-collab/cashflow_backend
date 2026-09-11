package helpdeskstorage

import (
	"context"
	"cashflow_backend/internal/domain/helpdesk"
	"sync"
)

type MemoryRepo struct {
	mu sync.RWMutex
	teams    map[int64]*helpdesk.Team
	stages   map[int64]*helpdesk.Stage
	tickets  map[int64]*helpdesk.Ticket
	policies map[int64]*helpdesk.SLAPolicy
	articles map[int64]*helpdesk.KnowledgeArticle
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		teams:    make(map[int64]*helpdesk.Team),
		stages:   make(map[int64]*helpdesk.Stage),
		tickets:  make(map[int64]*helpdesk.Ticket),
		policies: make(map[int64]*helpdesk.SLAPolicy),
		articles: make(map[int64]*helpdesk.KnowledgeArticle),
	}
}

func (r *MemoryRepo) GetTeam(ctx context.Context, id int64) (*helpdesk.Team, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.teams[id], nil
}

func (r *MemoryRepo) ListTeams(ctx context.Context, companyID int64) ([]helpdesk.Team, error) { return nil, nil }
func (r *MemoryRepo) GetStage(ctx context.Context, id int64) (*helpdesk.Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stages[id], nil
}
func (r *MemoryRepo) ListStages(ctx context.Context, teamID int64) ([]helpdesk.Stage, error) { return nil, nil }
func (r *MemoryRepo) GetTicket(ctx context.Context, id int64) (*helpdesk.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tickets[id], nil
}
func (r *MemoryRepo) GetTicketByNumber(ctx context.Context, number string) (*helpdesk.Ticket, error) { return nil, nil }
func (r *MemoryRepo) CreateTicket(ctx context.Context, ticket *helpdesk.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tickets[ticket.ID] = ticket
	return nil
}
func (r *MemoryRepo) UpdateTicket(ctx context.Context, ticket *helpdesk.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tickets[ticket.ID] = ticket
	return nil
}
func (r *MemoryRepo) ListTickets(ctx context.Context, teamID int64, stageID int64) ([]helpdesk.Ticket, error) { return nil, nil }
func (r *MemoryRepo) GetSLAPolicy(ctx context.Context, id int64) (*helpdesk.SLAPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.policies[id], nil
}
func (r *MemoryRepo) ListSLAPolicies(ctx context.Context, teamID int64) ([]helpdesk.SLAPolicy, error) { return nil, nil }
func (r *MemoryRepo) GetArticle(ctx context.Context, id int64) (*helpdesk.KnowledgeArticle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.articles[id], nil
}
func (r *MemoryRepo) GetArticleBySlug(ctx context.Context, slug string) (*helpdesk.KnowledgeArticle, error) { return nil, nil }
func (r *MemoryRepo) ListArticles(ctx context.Context, categoryID int64, includeInternal bool) ([]helpdesk.KnowledgeArticle, error) { return nil, nil }
func (r *MemoryRepo) ListCategories(ctx context.Context, companyID int64) ([]helpdesk.KnowledgeCategory, error) { return nil, nil }
