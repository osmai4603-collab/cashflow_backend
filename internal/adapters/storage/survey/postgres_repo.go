package surveystorage

import (
	"context"
	"cashflow_backend/internal/domain/survey"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CreateSurvey(ctx context.Context, s *survey.Survey) error { return nil }
func (r *PostgresRepo) GetSurveyByID(ctx context.Context, id int64) (*survey.Survey, error) { return nil, nil }
func (r *PostgresRepo) UpdateSurvey(ctx context.Context, s *survey.Survey) error { return nil }
func (r *PostgresRepo) DeleteSurvey(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) ListSurveys(ctx context.Context, companyID int64) ([]survey.Survey, error) { return nil, nil }
func (r *PostgresRepo) CreateQuestion(ctx context.Context, q *survey.SurveyQuestion) error { return nil }
func (r *PostgresRepo) GetQuestionsBySurveyID(ctx context.Context, surveyID int64) ([]survey.SurveyQuestion, error) { return nil, nil }
func (r *PostgresRepo) DeleteQuestion(ctx context.Context, id int64) error { return nil }
func (r *PostgresRepo) CreateSurveyInput(ctx context.Context, input *survey.SurveyInput) error { return nil }
func (r *PostgresRepo) GetSurveyInputByID(ctx context.Context, id int64) (*survey.SurveyInput, error) { return nil, nil }
func (r *PostgresRepo) ListInputsBySurveyID(ctx context.Context, surveyID int64) ([]survey.SurveyInput, error) { return nil, nil }
