package survey

type QuestionType string

const (
	TypeSingleChoice   QuestionType = "single_choice"
	TypeMultipleChoice QuestionType = "multiple_choice"
	TypeText           QuestionType = "text"
	TypeRating         QuestionType = "rating"
)

type SurveyQuestion struct {
	ID       int64        `json:"id"`
	SurveyID int64        `json:"survey_id"`
	Title    string       `json:"title"`
	Type     QuestionType `json:"type"`
	Sequence int          `json:"sequence"`
}

func (q *SurveyQuestion) Validate() error {
	if q.Title == "" {
		return errInvalid("question title cannot be empty")
	}
	if q.SurveyID <= 0 {
		return errInvalid("survey ID is required")
	}
	switch q.Type {
	case TypeSingleChoice, TypeMultipleChoice, TypeText, TypeRating:
	default:
		return errInvalid("invalid question type")
	}
	return nil
}
