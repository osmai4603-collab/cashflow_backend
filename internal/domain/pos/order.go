package pos

import (
	"fmt"
	"math"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type OrderState string

const (
	OrderStateDraft     OrderState = "draft"
	OrderStatePaid      OrderState = "paid"
	OrderStateDone      OrderState = "done"
	OrderStateInvoiced  OrderState = "invoiced"
	OrderStateCancelled OrderState = "cancelled"
)

type PosOrderLine struct {
	ID                int64     `json:"id"`
	OrderID           int64     `json:"order_id"`
	ProductID         int64     `json:"product_id"`
	Qty               float64   `json:"qty"`
	PriceUnit         float64   `json:"price_unit"`
	Discount          float64   `json:"discount"`
	PriceSubtotal     float64   `json:"price_subtotal"`
	PriceSubtotalIncl float64   `json:"price_subtotal_incl"`
	TaxRate           float64   `json:"tax_rate"`
	CustomerNote      string    `json:"customer_note,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

func (line *PosOrderLine) ComputeAmount() {
	discounted := line.Qty * line.PriceUnit * (1 - line.Discount/100)
	line.PriceSubtotal = roundMoney(discounted)
	line.PriceSubtotalIncl = roundMoney(discounted * (1 + line.TaxRate/100))
}

type PosOrder struct {
	ID            int64          `json:"id"`
	Name          string         `json:"name"`
	ClientUUID    string         `json:"client_uuid"`
	SessionID     int64          `json:"session_id"`
	PartnerID     *int64         `json:"partner_id,omitempty"`
	UserID        int64          `json:"user_id"`
	TableID       *int64         `json:"table_id,omitempty"`
	CustomerCount int            `json:"customer_count"`
	State         OrderState     `json:"state"`
	AmountUntaxed float64        `json:"amount_untaxed"`
	AmountTax     float64        `json:"amount_tax"`
	AmountTotal   float64        `json:"amount_total"`
	AmountPaid    float64        `json:"amount_paid"`
	AmountReturn  float64        `json:"amount_return"`
	TipAmount     float64        `json:"tip_amount"`
	Lines         []PosOrderLine `json:"lines"`
	Payments      []PosPayment   `json:"payments"`
	CompanyID     int64          `json:"company_id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (order *PosOrder) RecomputeTotals() {
	var untaxed, total float64
	for index := range order.Lines {
		order.Lines[index].ComputeAmount()
		untaxed += order.Lines[index].PriceSubtotal
		total += order.Lines[index].PriceSubtotalIncl
	}
	order.AmountUntaxed = roundMoney(untaxed)
	order.AmountTotal = roundMoney(total + order.TipAmount)
	order.AmountTax = roundMoney(order.AmountTotal - order.AmountUntaxed - order.TipAmount)
}

func (order *PosOrder) Validate() error {
	order.ClientUUID = strings.TrimSpace(order.ClientUUID)
	if order.ClientUUID == "" || order.SessionID <= 0 || order.UserID <= 0 || order.CompanyID <= 0 {
		return platformerrors.Validation("order references and client UUID are required", nil)
	}
	if len(order.Lines) == 0 {
		return platformerrors.Validation("order must contain at least one line", nil)
	}
	for index := range order.Lines {
		line := &order.Lines[index]
		if line.ProductID <= 0 || line.Qty <= 0 || line.PriceUnit < 0 || line.Discount < 0 || line.Discount > 100 || line.TaxRate < 0 {
			return platformerrors.Validation(fmt.Sprintf("invalid order line at index %d", index), nil)
		}
	}
	if order.TipAmount < 0 || math.IsNaN(order.TipAmount) {
		return platformerrors.Validation("tip amount must be non-negative", nil)
	}
	if order.State == "" {
		order.State = OrderStateDraft
	}
	return nil
}

func (order *PosOrder) AddPayment(payment PosPayment) error {
	if order.State != OrderStateDraft {
		return invalidState("order", order.State)
	}
	if err := payment.Validate(); err != nil {
		return err
	}
	if payment.OrderID == 0 {
		payment.OrderID = order.ID
	}
	order.Payments = append(order.Payments, payment)
	order.AmountPaid = roundMoney(order.AmountPaid + payment.Amount)
	if order.AmountPaid >= order.AmountTotal {
		order.AmountReturn = roundMoney(order.AmountPaid - order.AmountTotal)
		order.State = OrderStatePaid
	}
	return nil
}
