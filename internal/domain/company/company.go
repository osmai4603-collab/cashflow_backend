package company

import (
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Company represents an organizational entity (res.company in Odoo).
type Company struct {
	ID         int64        `json:"id"`
	Name       string       `json:"name"`
	PartnerID  *int64       `json:"partner_id,omitempty"`
	CurrencyID int64        `json:"currency_id"`
	Phone      string       `json:"phone,omitempty"`
	Email      string       `json:"email,omitempty"`
	Website    string       `json:"website,omitempty"`
	VAT        string       `json:"vat,omitempty"`
	Street     string       `json:"street,omitempty"`
	Street2    string       `json:"street2,omitempty"`
	City       string       `json:"city,omitempty"`
	State      string       `json:"state,omitempty"`
	Country    string       `json:"country,omitempty"`
	ZipCode    string       `json:"zip_code,omitempty"`
	Active     bool         `json:"active"`

	// Attendance Config
	AttendanceKioskMode      string  `json:"attendance_kiosk_mode"`
	AttendanceKioskDelay     int     `json:"attendance_kiosk_delay"`
	OvertimeCompanyThreshold int     `json:"overtime_company_threshold"`
	AutoCheckOutTolerance    float64 `json:"auto_check_out_tolerance"`

	// Landed cost default journal for stock landed-cost accounting.
	LandedCostJournalID *int64 `json:"landed_cost_journal_id,omitempty"`

	Audit audit.Fields `json:"audit"`
}

// Validate ensures the company entity satisfies all domain invariants.
func (c *Company) Validate() error {
	trimmedName := strings.TrimSpace(c.Name)
	if trimmedName == "" {
		return platformerrors.Validation("company name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if len(trimmedName) > 255 {
		return platformerrors.Validation("company name exceeds maximum length", map[string]string{
			"name": "must not exceed 255 characters",
		})
	}
	c.Name = trimmedName

	if c.CurrencyID <= 0 {
		return platformerrors.Validation("currency is required", map[string]string{
			"currency_id": "must be a valid currency",
		})
	}

	return nil
}
