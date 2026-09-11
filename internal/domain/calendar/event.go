package calendar

import (
	"fmt"
	"strings"
	"time"
)

type EventPrivacy string

const (
	PrivacyPublic       EventPrivacy = "public"
	PrivacyPrivate      EventPrivacy = "private"
	PrivacyConfidential EventPrivacy = "confidential"
)

type CalendarEvent struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	Description    string       `json:"description,omitempty"`
	Start          time.Time    `json:"start"`
	Stop           time.Time    `json:"stop"`
	Duration       float64      `json:"duration"`
	Allday         bool         `json:"allday"`
	Location       string       `json:"location,omitempty"`
	VideoURL       string       `json:"video_url,omitempty"`
	Privacy        EventPrivacy `json:"privacy"`
	ShowAs         string       `json:"show_as"`
	UserID         int64        `json:"user_id"`
	ResModel       string       `json:"res_model,omitempty"`
	ResID          *int64       `json:"res_id,omitempty"`
	RecurrenceID   *int64       `json:"recurrence_id,omitempty"`
	RecurrenceRule string       `json:"recurrence_rule,omitempty"`
	Attendees      []Attendee   `json:"attendees,omitempty"`
	Alarms         []EventAlarm `json:"alarms,omitempty"`
	CompanyID      int64        `json:"company_id"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

func (event *CalendarEvent) Validate() error {
	event.Name = strings.TrimSpace(event.Name)
	if event.Name == "" || event.UserID <= 0 || event.CompanyID <= 0 {
		return invalidCalendar("event requires name, user, and company")
	}
	if event.Start.IsZero() || event.Stop.IsZero() || !event.Stop.After(event.Start) {
		return invalidCalendar("event stop must be after event start")
	}
	if event.Privacy == "" {
		event.Privacy = PrivacyPublic
	}
	if event.Privacy != PrivacyPublic && event.Privacy != PrivacyPrivate && event.Privacy != PrivacyConfidential {
		return invalidCalendar(fmt.Sprintf("unsupported event privacy '%s'", event.Privacy))
	}
	if event.ShowAs == "" {
		event.ShowAs = "busy"
	}
	event.Duration = event.Stop.Sub(event.Start).Hours()
	return nil
}

func (event *CalendarEvent) Overlaps(start, stop time.Time) bool {
	return event.ShowAs != "free" && event.Start.Before(stop) && start.Before(event.Stop)
}
