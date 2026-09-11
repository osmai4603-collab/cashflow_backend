package calendar

import (
	"fmt"
	"strings"
	"time"
)

type RecurrenceRule struct {
	Freq       string      `json:"freq"`
	Interval   int         `json:"interval"`
	Count      *int        `json:"count,omitempty"`
	Until      *time.Time  `json:"until,omitempty"`
	ByDay      []string    `json:"by_day,omitempty"`
	Exceptions []time.Time `json:"exceptions,omitempty"`
}

func (rule *RecurrenceRule) Validate() error {
	rule.Freq = strings.ToUpper(strings.TrimSpace(rule.Freq))
	if rule.Freq != "DAILY" && rule.Freq != "WEEKLY" && rule.Freq != "MONTHLY" && rule.Freq != "YEARLY" {
		return invalidCalendar("frequency must be DAILY, WEEKLY, MONTHLY, or YEARLY")
	}
	if rule.Interval <= 0 {
		rule.Interval = 1
	}
	if rule.Count != nil && *rule.Count <= 0 {
		return invalidCalendar("recurrence count must be positive")
	}
	for _, day := range rule.ByDay {
		switch strings.ToUpper(day) {
		case "MO", "TU", "WE", "TH", "FR", "SA", "SU":
		default:
			return invalidCalendar(fmt.Sprintf("invalid BYDAY value '%s'", day))
		}
	}
	return nil
}

func (rule *RecurrenceRule) Occurrences(start, from, to time.Time) ([]time.Time, error) {
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	if !to.After(from) {
		return nil, invalidCalendar("recurrence range must be increasing")
	}
	var occurrences []time.Time
	current := start
	stepDaily := rule.Freq == "WEEKLY" && len(rule.ByDay) > 0
	for occurrenceIndex := 0; occurrenceIndex < 10000 && current.Before(to); occurrenceIndex++ {
		if rule.Count != nil && !stepDaily && occurrenceIndex >= *rule.Count {
			break
		}
		if rule.Until != nil && current.After(*rule.Until) {
			break
		}
		if !containsDate(rule.Exceptions, current) && !current.Before(from) {
			if (len(rule.ByDay) == 0 || containsWeekday(rule.ByDay, current.Weekday())) && (!stepDaily || weeksSince(start, current)%rule.Interval == 0) {
				occurrences = append(occurrences, current)
			}
		}
		if stepDaily {
			current = current.AddDate(0, 0, 1)
		} else {
			current = nextOccurrence(current, rule)
		}
	}
	return occurrences, nil
}

func weeksSince(start, current time.Time) int {
	return int(current.Sub(start).Hours() / (24 * 7))
}

func nextOccurrence(current time.Time, rule *RecurrenceRule) time.Time {
	switch rule.Freq {
	case "DAILY":
		return current.AddDate(0, 0, rule.Interval)
	case "WEEKLY":
		return current.AddDate(0, 0, 7*rule.Interval)
	case "MONTHLY":
		return current.AddDate(0, rule.Interval, 0)
	default:
		return current.AddDate(rule.Interval, 0, 0)
	}
}

func containsDate(values []time.Time, candidate time.Time) bool {
	for _, value := range values {
		if value.Year() == candidate.Year() && value.YearDay() == candidate.YearDay() {
			return true
		}
	}
	return false
}

func containsWeekday(values []string, weekday time.Weekday) bool {
	wanted := []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}[weekday]
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}
