package maintenancehttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	maintenancestorage "cashflow_backend/internal/adapters/storage/maintenance"
	"cashflow_backend/internal/platform/auth"
	maintenanceusecase "cashflow_backend/internal/usecase/maintenance"

	"github.com/go-chi/chi/v5"
)

func newRouter() (*Handler, chi.Router) {
	repo := maintenancestorage.NewMemoryRepo()
	handler := NewHandler(maintenanceusecase.New(repo, nil, nil), nil)
	router := chi.NewRouter()
	RegisterRoutes(router, handler)
	return handler, router
}

func doRequest(router chi.Router, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request = request.WithContext(auth.WithClaims(context.Background(), &auth.UserClaims{UserID: 7, CompanyID: 42}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestCreateEquipmentUsesAuthenticatedCompany(t *testing.T) {
	_, router := newRouter()
	response := doRequest(router, http.MethodPost, "/maintenance/equipments/", `{"name":"Pump","employee_id":11,"technician_user_id":5}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data struct {
			CompanyID int64 `json:"company_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.CompanyID != 42 {
		t.Fatalf("expected company 42, got %d", envelope.Data.CompanyID)
	}
}

func TestCreateRequestReturnsEquipmentLinkedResource(t *testing.T) {
	_, router := newRouter()

	equipmentResponse := doRequest(router, http.MethodPost, "/maintenance/equipments/", `{"name":"CNC Unit","employee_id":11}`)
	if equipmentResponse.Code != http.StatusCreated {
		t.Fatalf("failed to create equipment: %s", equipmentResponse.Body.String())
	}
	var equipmentBody struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(equipmentResponse.Body.Bytes(), &equipmentBody); err != nil {
		t.Fatal(err)
	}

	requestBody := `{"name":"MT/100","equipment_id":` + itoa(equipmentBody.Data.ID) + `,"maintenance_type":"corrective"}`
	requestResponse := doRequest(router, http.MethodPost, "/maintenance/requests/", requestBody)
	if requestResponse.Code != http.StatusCreated {
		t.Fatalf("failed to create request: %s", requestResponse.Body.String())
	}
	var requestBodyOut struct {
		Data struct {
			ID        int64 `json:"id"`
			Equipment *int64 `json:"equipment_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(requestResponse.Body.Bytes(), &requestBodyOut); err != nil {
		t.Fatal(err)
	}
	if requestBodyOut.Data.Equipment == nil || *requestBodyOut.Data.Equipment != equipmentBody.Data.ID {
		t.Fatalf("expected request linked to equipment %d, got %+v", equipmentBody.Data.ID, requestBodyOut.Data)
	}

	getResponse := doRequest(router, http.MethodGet, "/maintenance/requests/"+itoa(requestBodyOut.Data.ID), "")
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200 for get request, got %d", getResponse.Code)
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}