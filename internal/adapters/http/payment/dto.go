package paymenthttp

import (
	"time"

	"cashflow_backend/internal/domain/payment"
	paymentusecase "cashflow_backend/internal/usecase/payment"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreatePaymentRequest struct {
	PartnerID     int64                 `json:"partner_id"`
	Amount        float64               `json:"amount"`
	PaymentType   payment.PaymentType   `json:"payment_type"`
	PartnerType   payment.PartnerType   `json:"partner_type"`
	PaymentMethod payment.PaymentMethod `json:"payment_method"`
	JournalID     int64                 `json:"journal_id"`
	Date          *time.Time            `json:"date"`
	Ref           string                `json:"ref"`
	Currency      string                `json:"currency"`
	CompanyID     *int64                `json:"company_id"`
	InvoiceIDs    []int64               `json:"invoice_ids"`
	AutoReconcile bool                  `json:"auto_reconcile"`
}

func (r CreatePaymentRequest) ToInput() paymentusecase.CreatePaymentInput {
	var d time.Time
	if r.Date != nil {
		d = *r.Date
	}
	return paymentusecase.CreatePaymentInput{
		PartnerID:     r.PartnerID,
		Amount:        r.Amount,
		PaymentType:   r.PaymentType,
		PartnerType:   r.PartnerType,
		PaymentMethod: r.PaymentMethod,
		JournalID:     r.JournalID,
		Date:          d,
		Ref:           r.Ref,
		Currency:      r.Currency,
		CompanyID:     r.CompanyID,
		InvoiceIDs:    r.InvoiceIDs,
		AutoReconcile: r.AutoReconcile,
	}
}

type UpdatePaymentRequest struct {
	PartnerID     *int64                 `json:"partner_id"`
	Amount        *float64               `json:"amount"`
	PaymentMethod *payment.PaymentMethod `json:"payment_method"`
	JournalID     *int64                 `json:"journal_id"`
	Date          *time.Time             `json:"date"`
	Ref           *string                `json:"ref"`
	InvoiceIDs    []int64                `json:"invoice_ids"`
}

func (r UpdatePaymentRequest) ToInput() paymentusecase.UpdatePaymentInput {
	return paymentusecase.UpdatePaymentInput{
		PartnerID:     r.PartnerID,
		Amount:        r.Amount,
		PaymentMethod: r.PaymentMethod,
		JournalID:     r.JournalID,
		Date:          r.Date,
		Ref:           r.Ref,
		InvoiceIDs:    r.InvoiceIDs,
	}
}

type PostPaymentRequest struct {
	AutoReconcile bool `json:"auto_reconcile"`
}

type ReconcilePaymentRequest struct {
	InvoiceIDs []int64 `json:"invoice_ids"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Response DTOs
// ─────────────────────────────────────────────────────────────────────────────

type PaymentResponse struct {
	ID               int64                 `json:"id"`
	Name             string                `json:"name"`
	PaymentType      payment.PaymentType   `json:"payment_type"`
	PartnerType      payment.PartnerType   `json:"partner_type"`
	PartnerID        int64                 `json:"partner_id"`
	Amount           float64               `json:"amount"`
	Currency         string                `json:"currency"`
	PaymentMethod    payment.PaymentMethod `json:"payment_method"`
	JournalID        int64                 `json:"journal_id"`
	Date             string                `json:"date"`
	State            payment.PaymentState  `json:"state"`
	Ref              string                `json:"ref,omitempty"`
	MoveID           *int64                `json:"move_id,omitempty"`
	InvoiceIDs       []int64               `json:"invoice_ids"`
	ReconciledAmount float64               `json:"reconciled_amount"`
	ResidualAmount   float64               `json:"residual_amount"`
	CompanyID        *int64                `json:"company_id,omitempty"`
	Active           bool                  `json:"active"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

func ToPaymentResponse(p *payment.Payment) PaymentResponse {
	invIDs := p.InvoiceIDs
	if invIDs == nil {
		invIDs = make([]int64, 0)
	}
	return PaymentResponse{
		ID:               p.ID,
		Name:             p.Name,
		PaymentType:      p.PaymentType,
		PartnerType:      p.PartnerType,
		PartnerID:        p.PartnerID,
		Amount:           p.Amount,
		Currency:         p.Currency,
		PaymentMethod:    p.PaymentMethod,
		JournalID:        p.JournalID,
		Date:             p.Date.Format("2006-01-02"),
		State:            p.State,
		Ref:              p.Ref,
		MoveID:           p.MoveID,
		InvoiceIDs:       invIDs,
		ReconciledAmount: p.ReconciledAmount,
		ResidualAmount:   p.ResidualAmount,
		CompanyID:        p.CompanyID,
		Active:           p.Active,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.Format(time.RFC3339),
	}
}

type AgingReportResponse struct {
	ReportType string                     `json:"report_type"`
	AsOfDate   string                     `json:"as_of_date"`
	Total      payment.AgingBucket        `json:"total"`
	Partners   []payment.PartnerAgingItem `json:"partners"`
}

func ToAgingReportResponse(r *payment.AgingReport) AgingReportResponse {
	partners := r.Partners
	if partners == nil {
		partners = make([]payment.PartnerAgingItem, 0)
	}
	return AgingReportResponse{
		ReportType: r.ReportType,
		AsOfDate:   r.AsOfDate.Format("2006-01-02"),
		Total:      r.Total,
		Partners:   partners,
	}
}
