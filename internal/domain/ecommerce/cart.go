package ecommerce

import (
	"time"
)

type CartState string

const (
	CartStateActive    CartState = "active"
	CartStateAbandoned CartState = "abandoned"
	CartStateConverted CartState = "converted" // Turned into a Sale Order
)

type Cart struct {
	ID                int64      `json:"id"`
	WebsiteID         int64      `json:"website_id"`
	SessionUUID       string     `json:"session_uuid"`
	PartnerID         *int64     `json:"partner_id,omitempty"`
	PricelistID       int64      `json:"pricelist_id"`
	Currency          string     `json:"currency"`
	State             CartState  `json:"state"`
	ShippingAddressID *int64     `json:"shipping_address_id,omitempty"`
	InvoiceAddressID  *int64     `json:"invoice_address_id,omitempty"`
	DeliveryMethodID  *int64     `json:"delivery_method_id,omitempty"`
	ShippingAmount    float64    `json:"shipping_amount"`
	CouponCode        *string    `json:"coupon_code,omitempty"`
	DiscountAmount    float64    `json:"discount_amount"`
	AmountUntaxed     float64    `json:"amount_untaxed"`
	AmountTax         float64    `json:"amount_tax"`
	AmountTotal       float64    `json:"amount_total"`
	ConvertedOrderID  *int64     `json:"converted_order_id,omitempty"`
	LastActivityAt    time.Time  `json:"last_activity_at"`
	Lines             []CartLine `json:"lines,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type CartLine struct {
	ID         int64   `json:"id"`
	CartID     int64   `json:"cart_id"`
	ProductID  int64   `json:"product_id"`
	Quantity   float64 `json:"quantity"`
	PriceUnit  float64 `json:"price_unit"`
	Discount   float64 `json:"discount"`
	PriceTotal float64 `json:"price_total"`
	TaxIDs     []int64 `json:"tax_ids,omitempty"`
	Notes      string  `json:"notes,omitempty"`
}

func (c *Cart) Recalculate() {
	var untaxed float64
	for _, line := range c.Lines {
		untaxed += line.PriceTotal
	}
	c.AmountUntaxed = untaxed
	c.AmountTotal = untaxed + c.AmountTax + c.ShippingAmount - c.DiscountAmount
}
