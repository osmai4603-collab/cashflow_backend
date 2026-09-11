package helpdeskstorage

import (
	"cashflow_backend/internal/domain/helpdesk"
	"context"
	"sort"
	"sync"
)

type MemoryRepo struct {
	mu       sync.RWMutex
	teams    map[int64]*helpdesk.Team
	stages   map[int64]*helpdesk.Stage
	tickets  map[int64]*helpdesk.Ticket
	policies map[int64]*helpdesk.SLAPolicy
	articles map[int64]*helpdesk.KnowledgeArticle
	nextID   int64
}

func (r *MemoryRepo) allocateID() int64 { r.nextID++; return r.nextID }
func (r *MemoryRepo) SeedTeam(team *helpdesk.Team) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if team.ID == 0 {
		team.ID = r.allocateID()
	}
	r.teams[team.ID] = team
}
func (r *MemoryRepo) SeedStage(stage *helpdesk.Stage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if stage.ID == 0 {
		stage.ID = r.allocateID()
	}
	r.stages[stage.ID] = stage
}
func (r *MemoryRepo) SeedSLAPolicy(policy *helpdesk.SLAPolicy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if policy.ID == 0 {
		policy.ID = r.allocateID()
	}
	r.policies[policy.ID] = policy
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

func (r *MemoryRepo) ListTeams(ctx context.Context, companyID int64) ([]helpdesk.Team, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]helpdesk.Team, 0)
	for _, team := range r.teams {
		if companyID == 0 || team.CompanyID == companyID {
			result = append(result, *team)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) GetStage(ctx context.Context, id int64) (*helpdesk.Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stages[id], nil
}
func (r *MemoryRepo) ListStages(ctx context.Context, teamID int64) ([]helpdesk.Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]helpdesk.Stage, 0)
	for _, stage := range r.stages {
		if teamID == 0 || stage.TeamID == teamID {
			result = append(result, *stage)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result, nil
}
func (r *MemoryRepo) GetTicket(ctx context.Context, id int64) (*helpdesk.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tickets[id], nil
}
func (r *MemoryRepo) GetTicketByNumber(ctx context.Context, number string) (*helpdesk.Ticket, error) {
	return nil, nil
}
func (r *MemoryRepo) CreateTicket(ctx context.Context, ticket *helpdesk.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ticket.ID == 0 {
		ticket.ID = r.allocateID()
	}
	r.tickets[ticket.ID] = ticket
	return nil
}
func (r *MemoryRepo) UpdateTicket(ctx context.Context, ticket *helpdesk.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tickets[ticket.ID] = ticket
	return nil
}
func (r *MemoryRepo) ListTickets(ctx context.Context, teamID int64, stageID int64) ([]helpdesk.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]helpdesk.Ticket, 0)
	for _, ticket := range r.tickets {
		if (teamID == 0 || ticket.TeamID == teamID) && (stageID == 0 || ticket.StageID == stageID) {
			result = append(result, *ticket)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) GetSLAPolicy(ctx context.Context, id int64) (*helpdesk.SLAPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.policies[id], nil
}
func (r *MemoryRepo) ListSLAPolicies(ctx context.Context, teamID int64) ([]helpdesk.SLAPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]helpdesk.SLAPolicy, 0)
	for _, policy := range r.policies {
		if teamID == 0 || policy.TeamID == teamID {
			result = append(result, *policy)
		}
	}
	return result, nil
}
func (r *MemoryRepo) GetArticle(ctx context.Context, id int64) (*helpdesk.KnowledgeArticle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.articles[id], nil
}
func (r *MemoryRepo) CreateArticle(ctx context.Context, article *helpdesk.KnowledgeArticle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if article.ID == 0 {
		article.ID = r.allocateID()
	}
	r.articles[article.ID] = article
	return nil
}
func (r *MemoryRepo) UpdateArticle(ctx context.Context, article *helpdesk.KnowledgeArticle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.articles[article.ID]; !ok {
		return helpdesk.ErrArticleNotFound
	}
	r.articles[article.ID] = article
	return nil
}
func (r *MemoryRepo) GetArticleBySlug(ctx context.Context, slug string) (*helpdesk.KnowledgeArticle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, article := range r.articles {
		if article.Slug == slug {
			copy := *article
			return &copy, nil
		}
	}
	return nil, nil
}
func (r *MemoryRepo) ListArticles(ctx context.Context, categoryID int64, includeInternal bool) ([]helpdesk.KnowledgeArticle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]helpdesk.KnowledgeArticle, 0)
	for _, article := range r.articles {
		if (categoryID == 0 || article.CategoryID == categoryID) && (includeInternal || !article.IsInternal) {
			result = append(result, *article)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) ListCategories(ctx context.Context, companyID int64) ([]helpdesk.KnowledgeCategory, error) {
	return []helpdesk.KnowledgeCategory{}, nil
}
