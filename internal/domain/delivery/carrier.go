package delivery

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// CarrierType defines the delivery pricing provider type.
type CarrierType string

const (
	CarrierTypeFixed      CarrierType = "fixed"
	CarrierTypeBaseOnRule CarrierType = "base_on_rule"
)

// IntegrationLevel defines the action while validating delivery orders.
type IntegrationLevel string

const (
	IntegrationLevelRate        IntegrationLevel = "rate"
	IntegrationLevelRateAndShip IntegrationLevel = "rate_and_ship"
)

// InvoicingPolicy defines how shipping is invoiced.
type InvoicingPolicy string

const (
	InvoicingPolicyEstimated InvoicingPolicy = "estimated"
	InvoicingPolicyReal      InvoicingPolicy = "real"
)

// DeliveryCarrier represents a shipping method (delivery.carrier in Odoo).
type DeliveryCarrier struct {
	ID               int64            `json:"id"`
	Name             string           `json:"name"`
	Active           bool             `json:"active"`
	Sequence         int              `json:"sequence"`
	DeliveryType     CarrierType      `json:"delivery_type"`
	IntegrationLevel IntegrationLevel `json:"integration_level"`
	InvoicePolicy    InvoicingPolicy  `json:"invoice_policy"`
	ProductID        int64            `json:"product_id"`
	FixedPrice       float64          `json:"fixed_price"`
	Margin           float64          `json:"margin"`
	FixedMargin      float64          `json:"fixed_margin"`
	FreeOver         bool             `json:"free_over"`
	Amount           float64          `json:"amount"`
	MaxWeight        float64          `json:"max_weight"`
	MaxVolume        float64          `json:"max_volume"`

	CountryIDs   []int64 `json:"country_ids,omitempty"`
	StateIDs     []int64 `json:"state_ids,omitempty"`
	ZipPrefixIDs []int64 `json:"zip_prefix_ids,omitempty"`

	PriceRules []DeliveryPriceRule `json:"price_rules,omitempty"`

	CompanyID *int64    `json:"company_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeliveryPriceRule represents a pricing rule (delivery.price.rule in Odoo).
type DeliveryPriceRule struct {
	ID             int64   `json:"id"`
	CarrierID      int64   `json:"carrier_id"`
	Sequence       int     `json:"sequence"`
	Variable       string  `json:"variable"` // weight, volume, wv, price, quantity
	Operator       string  `json:"operator"` // ==, <=, <, >=, >
	MaxValue       float64 `json:"max_value"`
	ListBasePrice  float64 `json:"list_base_price"`
	ListPrice      float64 `json:"list_price"`
	VariableFactor string  `json:"variable_factor"`
}

// DeliveryZipPrefix represents a zip code prefix (delivery.zip.prefix in Odoo).
type DeliveryZipPrefix struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Validate ensures the carrier has valid configuration.
func (c *DeliveryCarrier) Validate() error {
	if c.Name == "" {
		return platformerrors.Validation("carrier name is required", map[string]string{"name": "cannot be empty"})
	}
	if c.ProductID <= 0 {
		return platformerrors.Validation("product is required for carrier", map[string]string{"product_id": "must reference a valid product"})
	}
	return nil
}
