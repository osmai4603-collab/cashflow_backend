package fleet

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// ServiceTypeCategory discriminates contracts (insurance/leasing) from one-off services.
type ServiceTypeCategory string

const (
	ServiceTypeCategoryContract ServiceTypeCategory = "contract"
	ServiceTypeCategoryService  ServiceTypeCategory = "service"
)

// VehicleState is a kanban stage record for fleet vehicles (Odoo fleet.vehicle.state).
type VehicleState struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
	Fold     bool   `json:"fold"`
}

// Validate ensures the state has a name.
func (s *VehicleState) Validate() error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return platformerrors.Validation("state name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	return nil
}

// ServiceType categorizes contracts vs services on the vehicle's log.
type ServiceType struct {
	ID       int64               `json:"id"`
	Name     string              `json:"name"`
	Category ServiceTypeCategory `json:"category"`
}

// Validate ensures the service type has a name and category.
func (s *ServiceType) Validate() error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return platformerrors.Validation("service type name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if s.Category == "" {
		s.Category = ServiceTypeCategoryService
	}
	switch s.Category {
	case ServiceTypeCategoryService, ServiceTypeCategoryContract:
	default:
		return platformerrors.Validation("invalid service type category", map[string]string{"category": string(s.Category)})
	}
	return nil
}

// VehicleCost aggregates service spend for the cost-by-vehicle report.
type VehicleCost struct {
	VehicleID       int64   `json:"vehicle_id"`
	VehicleName     string  `json:"vehicle_name"`
	LicensePlate    string  `json:"license_plate"`
	TotalAmount     float64 `json:"total_amount"`
	ServiceCount    int     `json:"service_count"`
	LastServiceDate *string `json:"last_service_date,omitempty"`
}