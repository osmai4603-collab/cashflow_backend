package surveystorage

import (
	"context"
	"sort"
	"sync"

	"cashflow_backend/internal/domain/survey"
)

type MemoryRepo struct {
	mu       sync.RWMutex
	surveys  map[int64]*survey.Survey
	questions map[int64]*survey.SurveyQuestion
	inputs   map[int64]*survey.SurveyInput
	nextID   int64
}

func (r *MemoryRepo) allocateID() int64 {
	r.nextID++
	return r.nextID
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		surveys:  make(map[int64]*survey.Survey),
		questions: make(map[int64]*survey.SurveyQuestion),
		inputs:   make(map[int64]*survey.SurveyInput),
	}
}

func (r *MemoryRepo) CreateSurvey(ctx context.Context, s *survey.Survey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.ID == 0 {
		s.ID = r.allocateID()
	}
	r.surveys[s.ID] = s
	return nil
}

func (r *MemoryRepo) GetSurveyByID(ctx context.Context, id int64) (*survey.Survey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.surveys[id]
	if !ok {
		return nil, nil
	}
	return item, nil
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

func (r *MemoryRepo) ListSurveys(ctx context.Context, companyID int64) ([]survey.Survey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]survey.Survey, 0)
	for _, s := range r.surveys {
		if companyID == 0 || s.CompanyID == companyID {
			result = append(result, *s)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateQuestion(ctx context.Context, q *survey.SurveyQuestion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if q.ID == 0 {
		q.ID = r.allocateID()
	}
	r.questions[q.ID] = q
	return nil
}

func (r *MemoryRepo) GetQuestionsBySurveyID(ctx context.Context, surveyID int64) ([]survey.SurveyQuestion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]survey.SurveyQuestion, 0)
	for _, q := range r.questions {
		if q.SurveyID == surveyID {
			result = append(result, *q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result, nil
}

func (r *MemoryRepo) DeleteQuestion(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.questions, id)
	return nil
}

func (r *MemoryRepo) CreateSurveyInput(ctx context.Context, input *survey.SurveyInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if input.ID == 0 {
		input.ID = r.allocateID()
	}
	r.inputs[input.ID] = input
	return nil
}

func (r *MemoryRepo) GetSurveyInputByID(ctx context.Context, id int64) (*survey.SurveyInput, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	input, ok := r.inputs[id]
	if !ok {
		return nil, nil
	}
	return input, nil
}

func (r *MemoryRepo) ListInputsBySurveyID(ctx context.Context, surveyID int64) ([]survey.SurveyInput, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]survey.SurveyInput, 0)
	for _, input := range r.inputs {
		if input.SurveyID == surveyID {
			result = append(result, *input)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
