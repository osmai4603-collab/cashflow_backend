package payment

import (
	"time"
)

// AgingBucket captures financial balances distributed across aging periods.
type AgingBucket struct {
	Current    float64 `json:"current"`     // Not yet overdue (due_date >= as_of_date)
	Days1_30   float64 `json:"days_1_30"`   // 1 to 30 days overdue
	Days31_60  float64 `json:"days_31_60"`  // 31 to 60 days overdue
	Days61_90  float64 `json:"days_61_90"`  // 61 to 90 days overdue
	Days91_120 float64 `json:"days_91_120"` // 91 to 120 days overdue
	Older      float64 `json:"older"`       // > 120 days overdue
	Total      float64 `json:"total"`       // Sum of all buckets
}

// Add adds another AgingBucket to the receiver.
func (b *AgingBucket) Add(other AgingBucket) {
	b.Current = roundTo4(b.Current + other.Current)
	b.Days1_30 = roundTo4(b.Days1_30 + other.Days1_30)
	b.Days31_60 = roundTo4(b.Days31_60 + other.Days31_60)
	b.Days61_90 = roundTo4(b.Days61_90 + other.Days61_90)
	b.Days91_120 = roundTo4(b.Days91_120 + other.Days91_120)
	b.Older = roundTo4(b.Older + other.Older)
	b.Total = roundTo4(b.Total + other.Total)
}

// PartnerAgingItem summarizes outstanding balances for a specific partner.
type PartnerAgingItem struct {
	PartnerID    int64       `json:"partner_id"`
	PartnerName  string      `json:"partner_name"`
	Buckets      AgingBucket `json:"buckets"`
	InvoiceCount int         `json:"invoice_count"`
}

// AgingReport represents an aged receivable or payable financial analysis.
type AgingReport struct {
	ReportType string             `json:"report_type"` // "receivable" or "payable"
	AsOfDate   time.Time          `json:"as_of_date"`
	Total      AgingBucket        `json:"total"`
	Partners   []PartnerAgingItem `json:"partners"`
}

// CategorizeAging categorizes an open amount into the appropriate aging bucket based on days overdue.
func CategorizeAging(asOfDate, dueDate time.Time, amount float64) AgingBucket {
	amount = roundTo4(amount)
	b := AgingBucket{Total: amount}

	// Normalize to midnight UTC for date-only comparison
	asOfMidnight := time.Date(asOfDate.Year(), asOfDate.Month(), asOfDate.Day(), 0, 0, 0, 0, time.UTC)
	dueMidnight := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, time.UTC)

	daysOverdue := int(asOfMidnight.Sub(dueMidnight).Hours() / 24)

	switch {
	case daysOverdue <= 0:
		b.Current = amount
	case daysOverdue <= 30:
		b.Days1_30 = amount
	case daysOverdue <= 60:
		b.Days31_60 = amount
	case daysOverdue <= 90:
		b.Days61_90 = amount
	case daysOverdue <= 120:
		b.Days91_120 = amount
	default:
		b.Older = amount
	}

	return b
}
