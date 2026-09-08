package fleetusecase

import (
	"context"
	"testing"
	"time"

	fleetstorage "cashflow_backend/internal/adapters/storage/fleet"
	"cashflow_backend/internal/domain/fleet"
	platformerrors "cashflow_backend/internal/platform/errors"
)

const companyID = 1

func newTestService(repo *fleetstorage.MemoryRepo) *Service {
	return New(repo, nil, nil)
}

// createTestVehicle builds a brand/model/vehicle for company-scoped tests.
func createTestVehicle(t *testing.T, svc *Service, license string) *fleet.Vehicle {
	t.Helper()
	ctx := context.Background()
	if err := svc.CreateBrand(ctx, &fleet.VehicleBrand{Name: "Toyota"}); err != nil {
		t.Fatalf("failed to create brand: %v", err)
	}
	model := &fleet.VehicleModel{Name: "Camry", BrandID: 1}
	if err := svc.CreateModel(ctx, model); err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	vehicle := &fleet.Vehicle{
		Name:         "Pool Camry",
		LicensePlate: license,
		ModelID:      model.ID,
		OdometerUnit: "kilometers",
		CompanyID:    companyID,
	}
	if err := svc.CreateVehicle(ctx, vehicle); err != nil {
		t.Fatalf("failed to create vehicle: %v", err)
	}
	return vehicle
}

func TestAssignationLogRejectsOverlap(t *testing.T) {
	ctx := context.Background()
	repo := fleetstorage.NewMemoryRepo()
	svc := newTestService(repo)
	vehicle := createTestVehicle(t, svc, "OVERLAP")

	start := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	first := &fleet.VehicleAssignationLog{VehicleID: vehicle.ID, DriverID: 10, DateStart: &start, DateEnd: &end}
	if err := svc.CreateAssignationLog(ctx, first); err != nil {
		t.Fatalf("unexpected error creating first assignation: %v", err)
	}

	// Overlapping window (same vehicle, different driver) must be rejected.
	overlapStart := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	overlapEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	second := &fleet.VehicleAssignationLog{VehicleID: vehicle.ID, DriverID: 11, DateStart: &overlapStart, DateEnd: &overlapEnd}
	err := svc.CreateAssignationLog(ctx, second)
	if err == nil || !isAppErr(err, platformerrors.CodeConflict) {
		t.Fatalf("expected overlap conflict, got %v", err)
	}

	// A window that ends before the first starts must be accepted.
	beforeStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	beforeEnd := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	third := &fleet.VehicleAssignationLog{VehicleID: vehicle.ID, DriverID: 12, DateStart: &beforeStart, DateEnd: &beforeEnd}
	if err := svc.CreateAssignationLog(ctx, third); err != nil {
		t.Fatalf("unexpected error creating non-overlapping assignation: %v", err)
	}
}

func TestOdometerDecreaseBlocked(t *testing.T) {
	ctx := context.Background()
	repo := fleetstorage.NewMemoryRepo()
	svc := newTestService(repo)
	vehicle := createTestVehicle(t, svc, "ODOMETER")

	first := &fleet.VehicleOdometer{VehicleID: vehicle.ID, Date: time.Now().UTC(), Value: 1000, Unit: "kilometers"}
	if err := svc.CreateOdometer(ctx, companyID, first); err != nil {
		t.Fatalf("unexpected error creating odometer: %v", err)
	}
	decrease := &fleet.VehicleOdometer{VehicleID: vehicle.ID, Date: time.Now().UTC(), Value: 900, Unit: "kilometers"}
	err := svc.CreateOdometer(ctx, companyID, decrease)
	if err == nil || !isAppErr(err, platformerrors.CodeValidation) {
		t.Fatalf("expected decrease to be rejected, got %v", err)
	}

	increase := &fleet.VehicleOdometer{VehicleID: vehicle.ID, Date: time.Now().UTC(), Value: 1100, Unit: "kilometers"}
	if err := svc.CreateOdometer(ctx, companyID, increase); err != nil {
		t.Fatalf("unexpected error creating increase: %v", err)
	}
}

func TestCostByVehicleAggregatesServices(t *testing.T) {
	ctx := context.Background()
	repo := fleetstorage.NewMemoryRepo()
	svc := newTestService(repo)
	vehicle := createTestVehicle(t, svc, "COSTS")

	for _, amount := range []float64{150.0, 75.5} {
		service := &fleet.VehicleLogService{
			VehicleID:     vehicle.ID,
			Description:   "Oil change",
			Date:          time.Now().UTC(),
			Amount:        amount,
			CompanyID:     companyID,
		}
		if err := svc.CreateLogService(ctx, service); err != nil {
			t.Fatalf("failed to create service log: %v", err)
		}
	}

	costs, err := svc.CostByVehicle(ctx, companyID)
	if err != nil {
		t.Fatalf("failed to compute cost report: %v", err)
	}
	if len(costs) != 1 {
		t.Fatalf("expected 1 vehicle in report, got %d", len(costs))
	}
	if costs[0].TotalAmount != 225.5 {
		t.Fatalf("expected total 225.5, got %v", costs[0].TotalAmount)
	}
	if costs[0].ServiceCount != 2 {
		t.Fatalf("expected 2 services, got %d", costs[0].ServiceCount)
	}
}

func TestRefreshExpiredContractsTransitionsState(t *testing.T) {
	ctx := context.Background()
	repo := fleetstorage.NewMemoryRepo()
	svc := newTestService(repo)
	vehicle := createTestVehicle(t, svc, "CONTRACT")

	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	past := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	future := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	pastContract := &fleet.VehicleLogContract{
		VehicleID:      vehicle.ID,
		Name:           "Expired policy",
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: &past,
		State:          fleet.ContractStateOpen,
		CompanyID:      companyID,
	}
	if err := svc.CreateLogContract(ctx, pastContract); err != nil {
		t.Fatalf("failed to create expired contract: %v", err)
	}
	futureContract := &fleet.VehicleLogContract{
		VehicleID:      vehicle.ID,
		Name:           "Active policy",
		StartDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: &future,
		State:          fleet.ContractStateOpen,
		CompanyID:      companyID,
	}
	if err := svc.CreateLogContract(ctx, futureContract); err != nil {
		t.Fatalf("failed to create active contract: %v", err)
	}

	refreshed, err := svc.RefreshExpiredContracts(ctx, now)
	if err != nil {
		t.Fatalf("failed to refresh contracts: %v", err)
	}
	if refreshed != 1 {
		t.Fatalf("expected 1 refreshed contract, got %d", refreshed)
	}

	pastReload, err := svc.GetLogContract(ctx, companyID, pastContract.ID)
	if err != nil {
		t.Fatalf("failed to reload past contract: %v", err)
	}
	if pastReload.State != fleet.ContractStateExpired {
		t.Fatalf("expected expired state, got %q", pastReload.State)
	}

	daysLeft := futureContract.DaysLeft(now)
	if daysLeft == nil || *daysLeft != 23 {
		t.Fatalf("expected 23 days left, got %v", daysLeft)
	}
}

func isAppErr(err error, code string) bool {
	appErr, ok := err.(*platformerrors.AppError)
	return ok && appErr.Code == code
}