package surveyusecase

import (
	"context"
	"testing"

	surveystorage "cashflow_backend/internal/adapters/storage/survey"
	"cashflow_backend/internal/domain/survey"
)

func TestSurveyUseCase_CreateAndSubmitSurvey(t *testing.T) {
	repo := surveystorage.NewMemoryRepo()
	uc := New(repo)
	ctx := context.Background()

	passingScore := 7.0
	s, err := uc.CreateSurvey(ctx, &survey.Survey{
		Title:        "Customer satisfaction",
		Description:  "Rate the experience",
		IsScoring:    true,
		PassingScore: &passingScore,
		CompanyID:    1,
	})
	if err != nil {
		t.Fatalf("create survey: %v", err)
	}
	if s.ID == 0 || s.Title != "Customer satisfaction" {
		t.Fatalf("unexpected survey created: %#v", s)
	}

	questions := []*survey.SurveyQuestion{
		{SurveyID: s.ID, Title: "How satisfied are you?", Type: survey.TypeRating, Sequence: 1},
		{SurveyID: s.ID, Title: "Would you recommend us?", Type: survey.TypeRating, Sequence: 2},
	}
	for _, q := range questions {
		if err := repo.CreateQuestion(ctx, q); err != nil {
			t.Fatalf("create question: %v", err)
		}
	}

	result, err := uc.SubmitSurvey(ctx, &survey.SurveyInput{
		SurveyID: s.ID,
		Lines: []survey.InputLine{
			{QuestionID: questions[0].ID, ValueScore: 4},
			{QuestionID: questions[1].ID, ValueScore: 5},
		},
	})
	if err != nil {
		t.Fatalf("submit survey: %v", err)
	}
	input, ok := result.(*survey.SurveyInput)
	if !ok {
		t.Fatalf("submit survey returned unexpected type: %T", result)
	}
	if input.TotalScore != 9 || !input.IsPassed {
		t.Fatalf("unexpected submitted survey result: %#v", input)
	}

	items, err := uc.ListSurveys(ctx)
	if err != nil {
		t.Fatalf("list surveys: %v", err)
	}
	if len(items) != 1 || items[0].ID != s.ID {
		t.Fatalf("unexpected survey list: %#v", items)
	}
}
