package ecommerce

import "time"

// AbandonedCartPolicy defines when a cart is considered abandoned and what to do.
type AbandonedCartPolicy struct {
	TimeoutHours      int  `json:"timeout_hours"`
	SendEmailReminder bool `json:"send_email_reminder"`
	ReminderDelay     int  `json:"reminder_delay_hours"`
}

func IsCartAbandoned(lastActivity time.Time, timeoutHours int) bool {
	return time.Since(lastActivity) > time.Duration(timeoutHours)*time.Hour
}
