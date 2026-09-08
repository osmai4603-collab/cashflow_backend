package delivery

import (
	"context"
)

// RateResult holds the output of a shipping rate calculation.
type RateResult struct {
	Success        bool    `json:"success"`
	Price          float64 `json:"price"`
	CarrierPrice   float64 `json:"carrier_price"`
	ErrorMessage   string  `json:"error_message,omitempty"`
	WarningMessage string  `json:"warning_message,omitempty"`
}

// Repository defines the persistence contract for delivery carriers.
type Repository interface {
	CreateCarrier(ctx context.Context, carrier *DeliveryCarrier) error
	GetCarrierByID(ctx context.Context, id int64) (*DeliveryCarrier, error)
	ListCarriers(ctx context.Context, filters map[string]interface{}) ([]DeliveryCarrier, error)
	UpdateCarrier(ctx context.Context, carrier *DeliveryCarrier) error
	DeleteCarrier(ctx context.Context, id int64) error

	CreatePriceRule(ctx context.Context, rule *DeliveryPriceRule) error
	DeletePriceRule(ctx context.Context, id int64) error

	GetZipPrefixes(ctx context.Context, ids []int64) ([]DeliveryZipPrefix, error)
	CreateZipPrefix(ctx context.Context, name string) (*DeliveryZipPrefix, error)
}
