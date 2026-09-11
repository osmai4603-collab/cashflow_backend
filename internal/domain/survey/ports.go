package survey

import (
	"context"
)

type Repository interface {
	CreateSurvey(ctx context.Context, s *Survey) error
	GetSurveyByID(ctx context.Context, id int64) (*Survey, error)
	UpdateSurvey(ctx context.Context, s *Survey) error
	DeleteSurvey(ctx context.Context, id int64) error
	ListSurveys(ctx context.Context, companyID int64) ([]Survey, error)

	CreateQuestion(ctx context.Context, q *SurveyQuestion) error
	GetQuestionsBySurveyID(ctx context.Context, surveyID int64) ([]SurveyQuestion, error)
	DeleteQuestion(ctx context.Context, id int64) error

	CreateSurveyInput(ctx context.Context, input *SurveyInput) error
	GetSurveyInputByID(ctx context.Context, id int64) (*SurveyInput, error)
	ListInputsBySurveyID(ctx context.Context, surveyID int64) ([]SurveyInput, error)
}
