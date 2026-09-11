package recruitment

import (
	"testing"
	"time"
)

func TestApplicantAndInterviewValidation(t *testing.T) {
	applicant := &Applicant{PartnerName: "Candidate", Email: "candidate@example.com", JobID: 1, StageID: 1, CompanyID: 1}
	if err := applicant.Validate(); err != nil {
		t.Fatalf("validate applicant: %v", err)
	}
	interview := &ApplicantInterview{ApplicantID: 1, InterviewerID: 2, InterviewDate: time.Now(), Score: 8}
	if err := interview.Validate(); err != nil {
		t.Fatalf("validate interview: %v", err)
	}
}
