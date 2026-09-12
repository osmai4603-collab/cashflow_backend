package surveyusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/survey"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo survey.Repository
}

func (u *UseCase) ListSurveys(ctx context.Context) ([]survey.Survey, error) {
	return u.repo.ListSurveys(ctx, 0)
}

func (u *UseCase) CreateSurvey(ctx context.Context, s *survey.Survey) (*survey.Survey, error) {
	if s == nil {
		return nil, platformerrors.Validation("survey is required", nil)
	}
	if s.CompanyID == 0 {
		s.CompanyID = 1
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.CreateSurvey(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *UseCase) GetSurvey(ctx context.Context, id int64) (*survey.Survey, error) {
	if id <= 0 {
		return nil, platformerrors.Validation("survey_id is required", nil)
	}
	s, err := u.repo.GetSurveyByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("survey %d not found", id))
	}
	return s, nil
}

func (u *UseCase) SubmitSurvey(ctx context.Context, input *survey.SurveyInput) (interface{}, error) {
	if input == nil {
		return nil, platformerrors.Validation("survey input is required", nil)
	}
	if input.SurveyID <= 0 {
		return nil, platformerrors.Validation("survey_id is required", nil)
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	if _, err := u.GetSurvey(ctx, input.SurveyID); err != nil {
		return nil, err
	}
	if len(input.Lines) == 0 {
		return nil, platformerrors.Validation("at least one answer line is required", nil)
	}
	var total float64
	for i := range input.Lines {
		line := &input.Lines[i]
		if line.QuestionID <= 0 {
			return nil, platformerrors.Validation("question_id is required", nil)
		}
		total += line.ValueScore
	}
	input.TotalScore = total
	if surveyObj, err := u.GetSurvey(ctx, input.SurveyID); err == nil && surveyObj != nil && surveyObj.IsScoring && surveyObj.PassingScore != nil {
		input.IsPassed = total >= *surveyObj.PassingScore
	} else {
		input.IsPassed = true
	}
	if err := u.repo.CreateSurveyInput(ctx, input); err != nil {
		return nil, err
	}
	return input, nil
}

func New(repo survey.Repository) *UseCase {
	return &UseCase{repo: repo}
}
