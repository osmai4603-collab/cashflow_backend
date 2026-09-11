package recruitment

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type ApplicantInterview struct {
	ID             int64     `json:"id"`
	ApplicantID    int64     `json:"applicant_id"`
	InterviewerID  int64     `json:"interviewer_id"`
	EventID        *int64    `json:"event_id,omitempty"`
	InterviewDate  time.Time `json:"interview_date"`
	Score          int       `json:"score"`
	Feedback       string    `json:"feedback"`
	Recommendation string    `json:"recommendation"`
}

func (interview *ApplicantInterview) Validate() error {
	interview.Feedback = strings.TrimSpace(interview.Feedback)
	if interview.ApplicantID <= 0 || interview.InterviewerID <= 0 || interview.InterviewDate.IsZero() || interview.Score < 0 || interview.Score > 10 {
		return platformerrors.Validation("interview has invalid references, date, or score", nil)
	}
	if interview.Recommendation == "" {
		interview.Recommendation = "consider"
	}
	if interview.Recommendation != "hire" && interview.Recommendation != "consider" && interview.Recommendation != "reject" {
		return platformerrors.Validation("invalid interview recommendation", nil)
	}
	return nil
}
