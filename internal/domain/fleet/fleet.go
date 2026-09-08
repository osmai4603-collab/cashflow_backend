package fleet

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
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

// Preset vehicle state names (Odoo fleet.vehicle.state records).
const (
	PresetStateNewRequest  = "New Request"
	PresetStateToOrder     = "To Order"
	PresetStateOrdered     = "Ordered"
	PresetStateRegistered  = "Registered"
	PresetStateDowngraded  = "Downgraded"
	PresetStateReserve     = "Reserve"
	PresetStateWaitingList = "Waiting List"
)

// Vehicle log service states.
const (
	ServiceStateNew       = "new"
	ServiceStateRunning   = "running"
	ServiceStateDone      = "done"
	ServiceStateCancelled = "cancelled"
)

// Vehicle log contract states.
const (
	ContractStateOpen     = "open"
	ContractStateExpired  = "expired"
	ContractStateClosed   = "closed"
)

// VehicleBrand represents a car brand (e.g., Toyota, Ford).
type VehicleBrand struct {
	ID        int64        `json:"id"`
	Name      i18n.TranslationString       `json:"name"`
	Image128  []byte       `json:"image_128,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// VehicleModelCategory represents a category for vehicle models (e.g., Sedan, SUV).
type VehicleModelCategory struct {
	ID        int64        `json:"id"`
	Name      i18n.TranslationString       `json:"name"`
	CreatedAt time.Time    `json:"created_at"`
}

// VehicleModel represents a specific car model (e.g., Camry, F-150).
type VehicleModel struct {
	ID         int64     `json:"id"`
	Name       i18n.TranslationString    `json:"name"`
	BrandID    int64     `json:"brand_id"`
	CategoryID *int64    `json:"category_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// VehicleTag represents a tag for vehicles.
type VehicleTag struct {
	ID        int64     `json:"id"`
	Name      i18n.TranslationString    `json:"name"`
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
	StateID           *int64       `json:"state_id,omitempty"`
	ManagerID         *int64       `json:"manager_id,omitempty"`
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
	Tags              []int64      `json:"tags,omitempty"` // tag IDs; nil leaves tags unchanged on update
	CompanyID         int64        `json:"company_id"`
	Audit             audit.Fields `json:"audit"`
}

// VehicleRenderName returns the display name used by Odoo: license plate or name.
func (v *Vehicle) VehicleRenderName() string {
	if v.LicensePlate != "" {
		return v.LicensePlate
	}
	return v.Name
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
	ID            int64        `json:"id"`
	VehicleID     int64        `json:"vehicle_id"`
	Description   string       `json:"description,omitempty"`
	Date          time.Time    `json:"date"`
	Amount        float64      `json:"amount"`
	VendorID      *int64       `json:"vendor_id,omitempty"`
	ServiceTypeID *int64       `json:"service_type_id,omitempty"`
	State         string       `json:"state"` // new, running, done, cancelled
	InvRef        string       `json:"inv_ref,omitempty"`
	Odometer      *float64     `json:"odometer,omitempty"`
	Notes         string       `json:"notes,omitempty"`
	CompanyID     int64        `json:"company_id"`
	Audit         audit.Fields `json:"audit"`
}

// VehicleLogContract represents an insurance or leasing contract for a vehicle.
type VehicleLogContract struct {
	ID             int64        `json:"id"`
	VehicleID      int64        `json:"vehicle_id"`
	Name           string       `json:"name,omitempty"`
	UserID         *int64       `json:"user_id,omitempty"`
	Date           *time.Time   `json:"date,omitempty"`
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

// Validate validates a service log entry.
func (s *VehicleLogService) Validate() error {
	s.Description = strings.TrimSpace(s.Description)
	if s.Description == "" {
		return platformerrors.Validation("service description is required", map[string]string{
			"description": "cannot be empty",
		})
	}
	if s.Date.IsZero() {
		return platformerrors.Validation("service date is required", nil)
	}
	if s.Amount < 0 {
		return platformerrors.Validation("service amount cannot be negative", nil)
	}
	if s.State == "" {
		s.State = ServiceStateNew
	}
	switch s.State {
	case ServiceStateNew, ServiceStateRunning, ServiceStateDone, ServiceStateCancelled:
	default:
		return platformerrors.Validation("invalid service state", map[string]string{"state": s.State})
	}
	return nil
}

// Validate validates a vehicle contract.
func (c *VehicleLogContract) Validate() error {
	if c.StartDate.IsZero() {
		return platformerrors.Validation("contract start date is required", nil)
	}
	if c.ExpirationDate != nil && c.ExpirationDate.Before(c.StartDate) {
		return platformerrors.Validation("expiration date cannot precede start date", nil)
	}
	if c.CostFrequency == "" {
		c.CostFrequency = "monthly"
	}
	switch c.CostFrequency {
	case "no", "daily", "weekly", "monthly", "yearly":
	default:
		return platformerrors.Validation("invalid cost frequency", map[string]string{"cost_frequency": c.CostFrequency})
	}
	if c.State == "" {
		c.State = ContractStateOpen
	}
	return nil
}

// DaysLeft returns whole days between today and expiration (negative means overdue).
// It returns nil when no expiration date is set.
func (c *VehicleLogContract) DaysLeft(now time.Time) *int {
	if c.ExpirationDate == nil {
		return nil
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	exp := time.Date(c.ExpirationDate.Year(), c.ExpirationDate.Month(), c.ExpirationDate.Day(), 0, 0, 0, 0, time.UTC)
	days := int(exp.Sub(today).Hours() / 24)
	return &days
}

// ExpiresToday reports whether the contract expiration falls on `now`'s calendar day.
func (c *VehicleLogContract) ExpiresToday(now time.Time) bool {
	if c.ExpirationDate == nil {
		return false
	}
	return c.ExpirationDate.Year() == now.Year() &&
		c.ExpirationDate.Month() == now.Month() &&
		c.ExpirationDate.Day() == now.Day()
}

// RefreshState transitions an open contract to expired when the expiration date passed.
func (c *VehicleLogContract) RefreshState(now time.Time) bool {
	if c.State != ContractStateOpen || c.ExpirationDate == nil {
		return false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	exp := time.Date(c.ExpirationDate.Year(), c.ExpirationDate.Month(), c.ExpirationDate.Day(), 0, 0, 0, 0, time.UTC)
	if exp.Before(today) {
		c.State = ContractStateExpired
		return true
	}
	return false
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
	switch v.OdometerUnit {
	case UnitKilometers, UnitMiles:
	default:
		return platformerrors.Validation("invalid odometer unit", map[string]string{"odometer_unit": v.OdometerUnit})
	}
	if v.State == "" {
		v.State = StateActive
	}
	switch v.State {
	case StateActive, StateInactive, StateRetired:
	default:
		return platformerrors.Validation("invalid vehicle state", map[string]string{"state": v.State})
	}
	return nil
}
