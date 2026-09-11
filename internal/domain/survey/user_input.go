package survey

import (
	"time"
)

type SurveyInput struct {
	ID         int64       `json:"id"`
	SurveyID   int64       `json:"survey_id"`
	PartnerID  *int64      `json:"partner_id,omitempty"`
	TotalScore float64     `json:"total_score"`
	IsPassed   bool        `json:"is_passed"`
	CreatedAt  time.Time   `json:"created_at"`
	Lines      []InputLine `json:"lines,omitempty"`
}

type InputLine struct {
	ID         int64   `json:"id"`
	InputID    int64   `json:"input_id"`
	QuestionID int64   `json:"question_id"`
	ValueText  string  `json:"value_text,omitempty"`
	ValueScore float64 `json:"value_score,omitempty"`
}
