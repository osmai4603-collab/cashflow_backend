package fleet

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Fuel types
const (
	FuelGasoline = "gasoline"
	FuelDiesel   = "diesel"
	FuelLPG      = "lpg"
	FuelElectric = "electric"
	FuelHybrid   = "hybrid"
)

// Odometer units
const (
	UnitKilometers = "kilometers"
	UnitMiles      = "miles"
)

// Vehicle states
const (
	StateActive    = "active"
	StateInactive  = "inactive"
	StateRetired   = "retired"
)

// VehicleBrand represents a car brand (e.g., Toyota, Ford).
type VehicleBrand struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	Image128  []byte       `json:"image_128,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// VehicleModelCategory represents a category for vehicle models (e.g., Sedan, SUV).
type VehicleModelCategory struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	CreatedAt time.Time    `json:"created_at"`
}

// VehicleModel represents a specific car model (e.g., Camry, F-150).
type VehicleModel struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	BrandID    int64     `json:"brand_id"`
	CategoryID *int64    `json:"category_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// VehicleTag represents a tag for vehicles.
type VehicleTag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     int       `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// Vehicle represents a vehicle in the fleet.
type Vehicle struct {
	ID                int64        `json:"id"`
	Name              string       `json:"name"`
	LicensePlate      string       `json:"license_plate"`
	ModelID           int64        `json:"model_id"`
	DriverID          *int64       `json:"driver_id,omitempty"`
	FutureDriverID    *int64       `json:"future_driver_id,omitempty"`
	VIN               string       `json:"vin_sn,omitempty"`
	AcquisitionDate   *time.Time   `json:"acquisition_date,omitempty"`
	FirstContractDate *time.Time   `json:"first_contract_date,omitempty"`
	Odometer          float64      `json:"odometer"`
	OdometerUnit      string       `json:"odometer_unit"`
	FuelType          string       `json:"fuel_type,omitempty"`
	Horsepower        int          `json:"horsepower,omitempty"`
	HorsepowerTax     float64      `json:"horsepower_tax,omitempty"`
	Seats             int          `json:"seats,omitempty"`
	Doors             int          `json:"doors,omitempty"`
	Color             string       `json:"color,omitempty"`
	Location          string       `json:"location,omitempty"`
	State             string       `json:"state"`
	Active            bool         `json:"active"`
	CompanyID         int64        `json:"company_id"`
	Audit             audit.Fields `json:"audit"`
}

// VehicleAssignationLog tracks the history of drivers for a vehicle.
type VehicleAssignationLog struct {
	ID        int64      `json:"id"`
	VehicleID int64      `json:"vehicle_id"`
	DriverID  int64      `json:"driver_id"`
	DateStart *time.Time `json:"date_start,omitempty"`
	DateEnd   *time.Time `json:"date_end,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// VehicleOdometer represents an odometer reading for a vehicle.
type VehicleOdometer struct {
	ID        int64     `json:"id"`
	VehicleID int64     `json:"vehicle_id"`
	Date      time.Time `json:"date"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"created_at"`
}

// VehicleLogService represents a maintenance service performed on a vehicle.
type VehicleLogService struct {
	ID          int64        `json:"id"`
	VehicleID   int64        `json:"vehicle_id"`
	Description string       `json:"description,omitempty"`
	Date        time.Time    `json:"date"`
	Amount      float64      `json:"amount"`
	VendorID    *int64       `json:"vendor_id,omitempty"`
	Odometer    *float64     `json:"odometer,omitempty"`
	Notes       string       `json:"notes,omitempty"`
	CompanyID   int64        `json:"company_id"`
	Audit       audit.Fields `json:"audit"`
}

// VehicleLogContract represents an insurance or leasing contract for a vehicle.
type VehicleLogContract struct {
	ID             int64        `json:"id"`
	VehicleID      int64        `json:"vehicle_id"`
	StartDate      time.Time    `json:"start_date"`
	ExpirationDate *time.Time   `json:"expiration_date,omitempty"`
	CostGenerated  float64      `json:"cost_generated"`
	CostFrequency  string       `json:"cost_frequency"` // no, daily, weekly, monthly, yearly
	InsRef         string       `json:"ins_ref,omitempty"`
	InsurerID      *int64       `json:"insurer_id,omitempty"`
	State          string       `json:"state"` // open, expired, closed
	Notes          string       `json:"notes,omitempty"`
	CompanyID      int64        `json:"company_id"`
	Audit          audit.Fields `json:"audit"`
}

func (v *Vehicle) Validate() error {
	v.LicensePlate = strings.ToUpper(strings.TrimSpace(v.LicensePlate))
	if v.LicensePlate == "" {
		return platformerrors.Validation("license plate is required", map[string]string{
			"license_plate": "cannot be empty",
		})
	}
	if v.OdometerUnit == "" {
		v.OdometerUnit = UnitKilometers
	}
	if v.State == "" {
		v.State = StateActive
	}
	return nil
}
