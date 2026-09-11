package surveystorage

import (
	"context"
	"cashflow_backend/internal/domain/survey"
	"sync"
)

type MemoryRepo struct {
	mu sync.RWMutex
	surveys   map[int64]*survey.Survey
	questions map[int64]*survey.SurveyQuestion
	inputs    map[int64]*survey.SurveyInput
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		surveys:   make(map[int64]*survey.Survey),
		questions: make(map[int64]*survey.SurveyQuestion),
		inputs:    make(map[int64]*survey.SurveyInput),
	}
}

func (r *MemoryRepo) CreateSurvey(ctx context.Context, s *survey.Survey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.surveys[s.ID] = s
	return nil
}
func (r *MemoryRepo) GetSurveyByID(ctx context.Context, id int64) (*survey.Survey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.surveys[id], nil
}
func (r *MemoryRepo) UpdateSurvey(ctx context.Context, s *survey.Survey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.surveys[s.ID] = s
	return nil
}
func (r *MemoryRepo) DeleteSurvey(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.surveys, id)
	return nil
}
func (r *MemoryRepo) ListSurveys(ctx context.Context, companyID int64) ([]survey.Survey, error) { return nil, nil }
func (r *MemoryRepo) CreateQuestion(ctx context.Context, q *survey.SurveyQuestion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.questions[q.ID] = q
	return nil
}
func (r *MemoryRepo) GetQuestionsBySurveyID(ctx context.Context, surveyID int64) ([]survey.SurveyQuestion, error) { return nil, nil }
func (r *MemoryRepo) DeleteQuestion(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.questions, id)
	return nil
}
func (r *MemoryRepo) CreateSurveyInput(ctx context.Context, input *survey.SurveyInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputs[input.ID] = input
	return nil
}
func (r *MemoryRepo) GetSurveyInputByID(ctx context.Context, id int64) (*survey.SurveyInput, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.inputs[id], nil
}
func (r *MemoryRepo) ListInputsBySurveyID(ctx context.Context, surveyID int64) ([]survey.SurveyInput, error) { return nil, nil }
