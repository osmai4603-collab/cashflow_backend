package calendar

type EventAlarm struct {
	ID              int64  `json:"id"`
	EventID         int64  `json:"event_id"`
	AlarmType       string `json:"alarm_type"`
	DurationMinutes int    `json:"duration_minutes"`
	Message         string `json:"message,omitempty"`
}

func (alarm *EventAlarm) Validate() error {
	if alarm.AlarmType == "" {
		alarm.AlarmType = "notification"
	}
	if alarm.AlarmType != "notification" && alarm.AlarmType != "email" && alarm.AlarmType != "sms" {
		return invalidCalendar("unsupported alarm type")
	}
	if alarm.DurationMinutes <= 0 {
		return invalidCalendar("alarm duration must be positive")
	}
	return nil
}
