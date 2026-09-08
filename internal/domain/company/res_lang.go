package company

import (
	"time"
)

// Language represents a language available in the system (res.lang in Odoo).
type Language struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"` // e.g., "en_US", "ar_001"
	ISO          string    `json:"iso_code"`
	Direction    string    `json:"direction"` // "ltr" or "rtl"
	DateFormat   string    `json:"date_format"`
	TimeFormat   string    `json:"time_format"`
	WeekStart    int       `json:"week_start"` // 1 (Mon) to 7 (Sun)
	DecimalPoint string    `json:"decimal_point"`
	ThousandsSep string    `json:"thousands_sep"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GetDefaultLang returns a basic configuration for English.
func GetDefaultLang() *Language {
	return &Language{
		Name:         "English (US)",
		Code:         "en_US",
		ISO:          "en",
		Direction:    "ltr",
		DateFormat:   "%m/%d/%Y",
		TimeFormat:   "%H:%M:%S",
		WeekStart:    7,
		DecimalPoint: ".",
		ThousandsSep: ",",
		Active:       true,
	}
}

// GetArabicLang returns a basic configuration for Arabic.
func GetArabicLang() *Language {
	return &Language{
		Name:         "Arabic (Saudi Arabia)",
		Code:         "ar_001",
		ISO:          "ar",
		Direction:    "rtl",
		DateFormat:   "%d/%m/%Y",
		TimeFormat:   "%H:%M:%S",
		WeekStart:    7,
		DecimalPoint: ".",
		ThousandsSep: ",",
		Active:       true,
	}
}
