package fleethttp

import (
	"time"

	"cashflow_backend/internal/domain/fleet"
	"cashflow_backend/internal/platform/i18n"
)

// CreateBrandRequest is the request body for creating a vehicle brand.
type CreateBrandRequest struct {
	Name string `json:"name"`
}

func (r CreateBrandRequest) ToDomain() *fleet.VehicleBrand {
	return &fleet.VehicleBrand{Name: i18n.NewTranslation(r.Name)}
}

// UpdateBrandRequest is the request body for updating a vehicle brand.
type UpdateBrandRequest struct {
	Name *string `json:"name"`
}

// CreateModelCategoryRequest is the request body for creating a model category.
type CreateModelCategoryRequest struct {
	Name string `json:"name"`
}

func (r CreateModelCategoryRequest) ToDomain() *fleet.VehicleModelCategory {
	return &fleet.VehicleModelCategory{Name: i18n.NewTranslation(r.Name)}
}

// UpdateModelCategoryRequest is the request body for updating a model category.
type UpdateModelCategoryRequest struct {
	Name *string `json:"name"`
}

// CreateModelRequest is the request body for creating a vehicle model.
type CreateModelRequest struct {
	Name       string `json:"name"`
	BrandID    int64  `json:"brand_id"`
	CategoryID *int64 `json:"category_id"`
}

func (r CreateModelRequest) ToDomain() *fleet.VehicleModel {
	return &fleet.VehicleModel{
		Name:       i18n.NewTranslation(r.Name),
		BrandID:    r.BrandID,
		CategoryID: r.CategoryID,
	}
}

// UpdateModelRequest is the request body for updating a vehicle model.
type UpdateModelRequest struct {
	Name       *string `json:"name"`
	BrandID    *int64  `json:"brand_id"`
	CategoryID *int64  `json:"category_id"`
}

// CreateTagRequest is the request body for creating a vehicle tag.
type CreateTagRequest struct {
	Name  string `json:"name"`
	Color int    `json:"color"`
}

func (r CreateTagRequest) ToDomain() *fleet.VehicleTag {
	return &fleet.VehicleTag{Name: i18n.NewTranslation(r.Name), Color: r.Color}
}

// UpdateTagRequest is the request body for updating a vehicle tag.
type UpdateTagRequest struct {
	Name  *string `json:"name"`
	Color *int    `json:"color"`
}

// CreateStateRequest is the request body for creating a vehicle state.
type CreateStateRequest struct {
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
	Fold     bool   `json:"fold"`
}

func (r CreateStateRequest) ToDomain() *fleet.VehicleState {
	return &fleet.VehicleState{Name: i18n.NewTranslation(r.Name), Sequence: r.Sequence, Fold: r.Fold}
}

// UpdateStateRequest is the request body for updating a vehicle state.
type UpdateStateRequest struct {
	Name     *string `json:"name"`
	Sequence *int    `json:"sequence"`
	Fold     *bool   `json:"fold"`
}

// CreateServiceTypeRequest is the request body for creating a service type.
type CreateServiceTypeRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"` // contract, service
}

func (r CreateServiceTypeRequest) ToDomain() *fleet.ServiceType {
	return &fleet.ServiceType{
		Name:     i18n.NewTranslation(r.Name),
		Category: fleet.ServiceTypeCategory(r.Category),
	}
}

// UpdateServiceTypeRequest is the request body for updating a service type.
type UpdateServiceTypeRequest struct {
	Name     *string `json:"name"`
	Category *string `json:"category"`
}

// CreateVehicleRequest is the request body for creating a vehicle.
type CreateVehicleRequest struct {
	Name              string     `json:"name"`
	LicensePlate      string     `json:"license_plate"`
	ModelID           int64      `json:"model_id"`
	DriverID          *int64     `json:"driver_id"`
	FutureDriverID    *int64     `json:"future_driver_id"`
	StateID           *int64     `json:"state_id"`
	ManagerID         *int64     `json:"manager_id"`
	VIN               string     `json:"vin_sn"`
	AcquisitionDate   *time.Time `json:"acquisition_date"`
	FirstContractDate *time.Time `json:"first_contract_date"`
	Odometer          float64    `json:"odometer"`
	OdometerUnit      string     `json:"odometer_unit"`
	FuelType          string     `json:"fuel_type"`
	Horsepower        int        `json:"horsepower"`
	HorsepowerTax     float64    `json:"horsepower_tax"`
	Seats             int        `json:"seats"`
	Doors             int        `json:"doors"`
	Color             string     `json:"color"`
	Location          string     `json:"location"`
	Tags              []int64    `json:"tags"`
}

func (r CreateVehicleRequest) ToDomain(companyID int64) *fleet.Vehicle {
	return &fleet.Vehicle{
		Name:              r.Name,
		LicensePlate:      r.LicensePlate,
		ModelID:           r.ModelID,
		DriverID:          r.DriverID,
		FutureDriverID:    r.FutureDriverID,
		StateID:           r.StateID,
		ManagerID:         r.ManagerID,
		VIN:               r.VIN,
		AcquisitionDate:   r.AcquisitionDate,
		FirstContractDate: r.FirstContractDate,
		Odometer:          r.Odometer,
		OdometerUnit:      r.OdometerUnit,
		FuelType:          r.FuelType,
		Horsepower:        r.Horsepower,
		HorsepowerTax:     r.HorsepowerTax,
		Seats:             r.Seats,
		Doors:             r.Doors,
		Color:             r.Color,
		Location:          r.Location,
		Tags:              r.Tags,
		CompanyID:         companyID,
	}
}

// UpdateVehicleRequest holds optional fields for updating a vehicle.
type UpdateVehicleRequest struct {
	Name              *string     `json:"name"`
	LicensePlate      *string     `json:"license_plate"`
	ModelID           *int64      `json:"model_id"`
	DriverID          *int64      `json:"driver_id"`
	FutureDriverID    *int64      `json:"future_driver_id"`
	StateID           *int64      `json:"state_id"`
	ManagerID         *int64      `json:"manager_id"`
	VIN               *string     `json:"vin_sn"`
	AcquisitionDate   *time.Time  `json:"acquisition_date"`
	FirstContractDate *time.Time  `json:"first_contract_date"`
	Odometer          *float64    `json:"odometer"`
	OdometerUnit      *string     `json:"odometer_unit"`
	FuelType          *string     `json:"fuel_type"`
	Horsepower        *int        `json:"horsepower"`
	HorsepowerTax     *float64    `json:"horsepower_tax"`
	Seats             *int        `json:"seats"`
	Doors             *int        `json:"doors"`
	Color             *string     `json:"color"`
	Location          *string     `json:"location"`
	State             *string     `json:"state"`
	Active            *bool       `json:"active"`
	Tags              []int64     `json:"tags"`
}

// Apply fills the domain entity with the non-nil request fields.
func (r UpdateVehicleRequest) Apply(v *fleet.Vehicle) {
	if r.Name != nil {
		v.Name = *r.Name
	}
	if r.LicensePlate != nil {
		v.LicensePlate = *r.LicensePlate
	}
	if r.ModelID != nil {
		v.ModelID = *r.ModelID
	}
	if r.DriverID != nil {
		v.DriverID = r.DriverID
	}
	if r.FutureDriverID != nil {
		v.FutureDriverID = r.FutureDriverID
	}
	if r.StateID != nil {
		v.StateID = r.StateID
	}
	if r.ManagerID != nil {
		v.ManagerID = r.ManagerID
	}
	if r.VIN != nil {
		v.VIN = *r.VIN
	}
	if r.AcquisitionDate != nil {
		v.AcquisitionDate = r.AcquisitionDate
	}
	if r.FirstContractDate != nil {
		v.FirstContractDate = r.FirstContractDate
	}
	if r.Odometer != nil {
		v.Odometer = *r.Odometer
	}
	if r.OdometerUnit != nil {
		v.OdometerUnit = *r.OdometerUnit
	}
	if r.FuelType != nil {
		v.FuelType = *r.FuelType
	}
	if r.Horsepower != nil {
		v.Horsepower = *r.Horsepower
	}
	if r.HorsepowerTax != nil {
		v.HorsepowerTax = *r.HorsepowerTax
	}
	if r.Seats != nil {
		v.Seats = *r.Seats
	}
	if r.Doors != nil {
		v.Doors = *r.Doors
	}
	if r.Color != nil {
		v.Color = *r.Color
	}
	if r.Location != nil {
		v.Location = *r.Location
	}
	if r.State != nil {
		v.State = *r.State
	}
	if r.Active != nil {
		v.Active = *r.Active
	}
	if r.Tags != nil {
		v.Tags = r.Tags
	}
}

// CreateAssignationRequest is the request body for assigning a driver.
type CreateAssignationRequest struct {
	DriverID int64      `json:"driver_id"`
	DateFrom *time.Time `json:"date_start"`
	DateTo   *time.Time `json:"date_end"`
}

func (r CreateAssignationRequest) ToDomain(vehicleID int64) *fleet.VehicleAssignationLog {
	return &fleet.VehicleAssignationLog{
		VehicleID: vehicleID,
		DriverID:  r.DriverID,
		DateStart: r.DateFrom,
		DateEnd:   r.DateTo,
	}
}

// UpdateAssignationRequest holds optional fields for updating an assignation.
type UpdateAssignationRequest struct {
	DateFrom *time.Time `json:"date_start"`
	DateTo   *time.Time `json:"date_end"`
}

// CreateOdometerRequest is the request body for recording an odometer reading.
type CreateOdometerRequest struct {
	Date  *time.Time `json:"date"`
	Value float64    `json:"value"`
	Unit  string     `json:"unit"`
}

func (r CreateOdometerRequest) ToDomain(vehicleID int64) *fleet.VehicleOdometer {
	odometer := &fleet.VehicleOdometer{
		VehicleID: vehicleID,
		Value:     r.Value,
		Unit:      r.Unit,
	}
	if r.Date != nil {
		odometer.Date = *r.Date
	}
	return odometer
}

// UpdateOdometerRequest holds optional fields for updating an odometer reading.
type UpdateOdometerRequest struct {
	Date  *time.Time `json:"date"`
	Value *float64   `json:"value"`
	Unit  *string    `json:"unit"`
}

// CreateServiceRequest is the request body for logging vehicle service work.
type CreateServiceRequest struct {
	Description   string     `json:"description"`
	Date          *time.Time `json:"date"`
	Amount        float64    `json:"amount"`
	VendorID      *int64     `json:"vendor_id"`
	ServiceTypeID *int64     `json:"service_type_id"`
	State         string     `json:"state"`
	InvRef        string     `json:"inv_ref"`
	Odometer      *float64   `json:"odometer"`
	Notes         string     `json:"notes"`
}

func (r CreateServiceRequest) ToDomain(vehicleID, companyID int64) *fleet.VehicleLogService {
	service := &fleet.VehicleLogService{
		VehicleID:     vehicleID,
		Description:   r.Description,
		Amount:        r.Amount,
		VendorID:      r.VendorID,
		ServiceTypeID: r.ServiceTypeID,
		State:         r.State,
		InvRef:        r.InvRef,
		Odometer:      r.Odometer,
		Notes:         r.Notes,
		CompanyID:     companyID,
	}
	if r.Date != nil {
		service.Date = *r.Date
	}
	return service
}

// UpdateServiceRequest holds optional fields for updating a service log.
type UpdateServiceRequest struct {
	Description   *string    `json:"description"`
	Date          *time.Time `json:"date"`
	Amount        *float64   `json:"amount"`
	VendorID      *int64     `json:"vendor_id"`
	ServiceTypeID *int64     `json:"service_type_id"`
	State         *string    `json:"state"`
	InvRef        *string    `json:"inv_ref"`
	Odometer      *float64   `json:"odometer"`
	Notes         *string    `json:"notes"`
}

// Apply fills the domain entity with the non-nil request fields.
func (r UpdateServiceRequest) Apply(s *fleet.VehicleLogService) {
	if r.Description != nil {
		s.Description = *r.Description
	}
	if r.Date != nil {
		s.Date = *r.Date
	}
	if r.Amount != nil {
		s.Amount = *r.Amount
	}
	if r.VendorID != nil {
		s.VendorID = r.VendorID
	}
	if r.ServiceTypeID != nil {
		s.ServiceTypeID = r.ServiceTypeID
	}
	if r.State != nil {
		s.State = *r.State
	}
	if r.InvRef != nil {
		s.InvRef = *r.InvRef
	}
	if r.Odometer != nil {
		s.Odometer = r.Odometer
	}
	if r.Notes != nil {
		s.Notes = *r.Notes
	}
}

// CreateContractRequest is the request body for creating a vehicle contract.
type CreateContractRequest struct {
	Name           string     `json:"name"`
	UserID         *int64     `json:"user_id"`
	Date           *time.Time `json:"date"`
	StartDate      *time.Time `json:"start_date"`
	ExpirationDate *time.Time `json:"expiration_date"`
	CostGenerated  float64    `json:"cost_generated"`
	CostFrequency  string     `json:"cost_frequency"`
	InsRef         string     `json:"ins_ref"`
	InsurerID      *int64     `json:"insurer_id"`
	State          string     `json:"state"`
	Notes          string     `json:"notes"`
}

func (r CreateContractRequest) ToDomain(vehicleID, companyID int64) *fleet.VehicleLogContract {
	contract := &fleet.VehicleLogContract{
		VehicleID:      vehicleID,
		Name:           r.Name,
		UserID:         r.UserID,
		Date:           r.Date,
		ExpirationDate: r.ExpirationDate,
		CostGenerated:  r.CostGenerated,
		CostFrequency:  r.CostFrequency,
		InsRef:         r.InsRef,
		InsurerID:      r.InsurerID,
		State:          r.State,
		Notes:          r.Notes,
		CompanyID:      companyID,
	}
	if r.StartDate != nil {
		contract.StartDate = *r.StartDate
	}
	return contract
}

// UpdateContractRequest holds optional fields for updating a contract.
type UpdateContractRequest struct {
	Name           *string    `json:"name"`
	UserID         *int64     `json:"user_id"`
	Date           *time.Time `json:"date"`
	StartDate      *time.Time `json:"start_date"`
	ExpirationDate *time.Time `json:"expiration_date"`
	CostGenerated  *float64   `json:"cost_generated"`
	CostFrequency  *string    `json:"cost_frequency"`
	InsRef         *string    `json:"ins_ref"`
	InsurerID      *int64     `json:"insurer_id"`
	State          *string    `json:"state"`
	Notes          *string    `json:"notes"`
}

// Apply fills the domain entity with the non-nil request fields.
func (r UpdateContractRequest) Apply(c *fleet.VehicleLogContract) {
	if r.Name != nil {
		c.Name = *r.Name
	}
	if r.UserID != nil {
		c.UserID = r.UserID
	}
	if r.Date != nil {
		c.Date = r.Date
	}
	if r.StartDate != nil {
		c.StartDate = *r.StartDate
	}
	if r.ExpirationDate != nil {
		c.ExpirationDate = r.ExpirationDate
	}
	if r.CostGenerated != nil {
		c.CostGenerated = *r.CostGenerated
	}
	if r.CostFrequency != nil {
		c.CostFrequency = *r.CostFrequency
	}
	if r.InsRef != nil {
		c.InsRef = *r.InsRef
	}
	if r.InsurerID != nil {
		c.InsurerID = r.InsurerID
	}
	if r.State != nil {
		c.State = *r.State
	}
	if r.Notes != nil {
		c.Notes = *r.Notes
	}
}