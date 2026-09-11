package surveyusecase

import (
	"context"
	"cashflow_backend/internal/domain/survey"
)

type UseCase struct {
	repo survey.Repository
}

func (u *UseCase) ListSurveys(ctx context.Context) ([]survey.Survey, error) {
	return nil, nil
}

func (u *UseCase) CreateSurvey(ctx context.Context, s *survey.Survey) (*survey.Survey, error) {
	return nil, nil
}

func (u *UseCase) GetSurvey(ctx context.Context, id int64) (*survey.Survey, error) {
	return nil, nil
}

func (u *UseCase) SubmitSurvey(ctx context.Context, input *survey.UserInput) (interface{}, error) {
	return nil, nil
}

func New(repo survey.Repository) *UseCase {
	return &UseCase{repo: repo}
}
