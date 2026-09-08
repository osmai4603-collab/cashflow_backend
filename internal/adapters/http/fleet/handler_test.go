package fleethttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fleetstorage "cashflow_backend/internal/adapters/storage/fleet"
	"cashflow_backend/internal/platform/auth"
	fleetusecase "cashflow_backend/internal/usecase/fleet"

	"github.com/go-chi/chi/v5"
)

func newFleetRouter() chi.Router {
	repo := fleetstorage.NewMemoryRepo()
	handler := NewHandler(fleetusecase.New(repo, nil, nil), nil)
	router := chi.NewRouter()
	RegisterRoutes(router, handler)
	return router
}

func fleetRequest(router chi.Router, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request = request.WithContext(auth.WithClaims(context.Background(), &auth.UserClaims{UserID: 7, CompanyID: 42}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func createVehicle(router chi.Router) int64 {
	brand := fleetRequest(router, http.MethodPost, "/fleet/brands", `{"name":"Toyota"}`)
	if brand.Code != http.StatusCreated {
		panic(fmt.Sprintf("failed to create brand: %d %s", brand.Code, brand.Body.String()))
	}
	var brandBody struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(brand.Body.Bytes(), &brandBody); err != nil {
		panic(err)
	}
	model := fleetRequest(router, http.MethodPost, "/fleet/model-categories", `{"name":"Sedan"}`)
	if model.Code != http.StatusCreated {
		panic(fmt.Sprintf("failed to create model category: %d %s", model.Code, model.Body.String()))
	}
	var categoryBody struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(model.Body.Bytes(), &categoryBody); err != nil {
		panic(err)
	}
	modelResponse := fleetRequest(router, http.MethodPost, "/fleet/models",
		fmt.Sprintf(`{"name":"Camry","brand_id":%d,"category_id":%d}`, brandBody.Data.ID, categoryBody.Data.ID))
	if modelResponse.Code != http.StatusCreated {
		panic(fmt.Sprintf("failed to create model: %d %s", modelResponse.Code, modelResponse.Body.String()))
	}
	var modelBody struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(modelResponse.Body.Bytes(), &modelBody); err != nil {
		panic(err)
	}

	response := fleetRequest(router, http.MethodPost, "/fleet/vehicles", fmt.Sprintf(`{
		"name": "Pool Camry",
		"license_plate": "AB123CD",
		"model_id": %d,
		"odometer_unit": "kilometers"
	}`, modelBody.Data.ID))
	if response.Code != http.StatusCreated {
		panic(fmt.Sprintf("failed to create vehicle: %d %s", response.Code, response.Body.String()))
	}
	var body struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		panic(err)
	}
	return body.Data.ID
}

func TestCreateVehicleEnforcesOverlapConflict(t *testing.T) {
	router := newFleetRouter()
	vehicleID := createVehicle(router)

	first := fleetRequest(router, http.MethodPost,
		fmt.Sprintf("/fleet/vehicles/%d/assignations", vehicleID),
		`{"driver_id":1,"date_start":"2026-01-10T00:00:00Z","date_end":"2026-01-20T00:00:00Z"}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first assignation, got %d: %s", first.Code, first.Body.String())
	}

	overlap := fleetRequest(router, http.MethodPost,
		fmt.Sprintf("/fleet/vehicles/%d/assignations", vehicleID),
		`{"driver_id":2,"date_start":"2026-01-15T00:00:00Z","date_end":"2026-02-01T00:00:00Z"}`)
	if overlap.Code != http.StatusConflict {
		t.Fatalf("expected 409 for overlapping assignation, got %d: %s", overlap.Code, overlap.Body.String())
	}
}

func TestCostByVehicleReport(t *testing.T) {
	router := newFleetRouter()
	vehicleID := createVehicle(router)

	for _, payload := range []string{
		`{"description":"Oil change","date":"2026-01-15T00:00:00Z","amount":150.0}`,
		`{"description":"Brake pads","date":"2026-02-01T00:00:00Z","amount":75.5}`,
	} {
		response := fleetRequest(router, http.MethodPost,
			fmt.Sprintf("/fleet/vehicles/%d/services", vehicleID), payload)
		if response.Code != http.StatusCreated {
			t.Fatalf("failed to create service: %d %s", response.Code, response.Body.String())
		}
	}

	report := fleetRequest(router, http.MethodGet, "/fleet/cost-by-vehicle", "")
	if report.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", report.Code, report.Body.String())
	}
	var envelope struct {
		Data []struct {
			VehicleName  string  `json:"vehicle_name"`
			TotalAmount  float64 `json:"total_amount"`
			ServiceCount int     `json:"service_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(report.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 1 || envelope.Data[0].TotalAmount != 225.5 || envelope.Data[0].ServiceCount != 2 {
		t.Fatalf("unexpected cost report: %+v", envelope.Data)
	}
}