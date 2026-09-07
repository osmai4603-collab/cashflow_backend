package activity

import (
	"testing"
	"time"
)

func TestActivityValidateRequiresCompletePolymorphicReference(t *testing.T) {
	resourceID := int64(42)
	value := Activity{ActivityTypeID: 1, Summary: "Call customer", DateDeadline: time.Now(), AssignedUserID: 2, CompanyID: 1, ResModel: "partner"}
	if err := value.Validate(); err == nil {
		t.Fatal("expected incomplete polymorphic reference to fail")
	}
	value.ResID = &resourceID
	if err := value.Validate(); err != nil {
		t.Fatalf("expected valid reference, got %v", err)
	}
}

func TestActivityStateAtUsesUserTimezone(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)
	deadline := time.Date(2026, 9, 7, 0, 30, 0, 0, time.UTC)
	now := time.Date(2026, 9, 6, 22, 0, 0, 0, time.UTC)
	value := Activity{DateDeadline: deadline}
	if got := value.StateAt(now, location); got != StateToday {
		t.Fatalf("expected today in user timezone, got %s", got)
	}
}

func TestActivityCompleteArchivesAndPreservesFeedback(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	value := Activity{Active: true}
	if err := value.Complete(now, "customer confirmed"); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	if value.Active || value.DateDone == nil || value.Feedback != "customer confirmed" {
		t.Fatalf("expected archived completion with feedback: %+v", value)
	}
	if err := value.Complete(now, "again"); err == nil {
		t.Fatal("expected repeated completion to fail")
	}
}
