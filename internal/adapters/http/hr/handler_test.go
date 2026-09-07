package hrhttp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	hrhttp "cashflow_backend/internal/adapters/http/hr"
	companystorage "cashflow_backend/internal/adapters/storage/company"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	hrusecase "cashflow_backend/internal/usecase/hr"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*chi.Mux, *hrhttp.Handler, *hrusecase.UseCase) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hrRepo := hrstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()

	useCase := hrusecase.New(hrRepo, partnerRepo, logger)
	companyRepo := companystorage.NewMemoryRepo()
	attendanceUseCase := hrusecase.NewAttendanceUseCase(hrRepo, companyRepo, logger)
	handler := hrhttp.NewHandler(useCase, attendanceUseCase, logger)

	r := chi.NewRouter()
	r.Route("/api/v1", func(v1 chi.Router) {
		hrhttp.RegisterRoutes(v1, handler)
	})

	return r, handler, useCase
}

func TestHRHandler_EndToEnd(t *testing.T) {
	r, _, _ := setupTestServer()

	// 1. Create Department
	deptReq := hrhttp.CreateDepartmentRequest{
		Name: "Customer Support",
	}
	body, _ := json.Marshal(deptReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create department status = %d, body = %s", w.Code, w.Body.String())
	}

	var deptResp struct {
		Success bool                      `json:"success"`
		Data    hrhttp.DepartmentResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&deptResp); err != nil {
		t.Fatalf("failed to decode department response: %v", err)
	}
	deptID := deptResp.Data.ID

	// 2. Create Job
	jobReq := hrhttp.CreateJobRequest{
		Name:              "Support Specialist",
		DepartmentID:      &deptID,
		ExpectedEmployees: 3,
	}
	body, _ = json.Marshal(jobReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create job status = %d, body = %s", w.Code, w.Body.String())
	}

	var jobResp struct {
		Success bool               `json:"success"`
		Data    hrhttp.JobResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&jobResp); err != nil {
		t.Fatalf("failed to decode job response: %v", err)
	}
	jobID := jobResp.Data.ID

	// 3. Create Employee
	empReq := hrhttp.CreateEmployeeRequest{
		Name:              "Layla Mahmoud",
		DepartmentID:      &deptID,
		JobID:             &jobID,
		WorkEmail:         "layla@example.com",
		AutoCreatePartner: true,
	}
	body, _ = json.Marshal(empReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/employees", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create employee status = %d, body = %s", w.Code, w.Body.String())
	}

	var empResp struct {
		Success bool                    `json:"success"`
		Data    hrhttp.EmployeeResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&empResp); err != nil {
		t.Fatalf("failed to decode employee response: %v", err)
	}
	empID := empResp.Data.ID

	// 4. Allocate Leave (15 days annual)
	allocReq := hrhttp.CreateAllocationRequest{
		EmployeeID:    empID,
		LeaveType:     "annual",
		AllocatedDays: 15,
		Year:          2026,
	}
	body, _ = json.Marshal(allocReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/leave-allocations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create allocation status = %d, body = %s", w.Code, w.Body.String())
	}

	// 5. Submit Leave Request (4 days)
	dFrom := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	dTo := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	leaveReq := hrhttp.CreateLeaveRequestRequest{
		EmployeeID:  empID,
		LeaveType:   "annual",
		DateFrom:    dFrom,
		DateTo:      dTo,
		Description: "Family summer trip",
		AutoConfirm: true,
	}
	body, _ = json.Marshal(leaveReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/leave-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create leave request status = %d, body = %s", w.Code, w.Body.String())
	}

	var lrResp struct {
		Success bool                        `json:"success"`
		Data    hrhttp.LeaveRequestResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&lrResp); err != nil {
		t.Fatalf("failed to decode leave request response: %v", err)
	}
	lrID := lrResp.Data.ID

	// 6. Approve Leave Request
	approveReq := hrhttp.ApproveLeaveRequestRequest{
		ApproverID: 1,
	}
	body, _ = json.Marshal(approveReq)
	url := fmt.Sprintf("/api/v1/leave-requests/%d/approve", lrID)
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("approve leave request status = %d, body = %s", w.Code, w.Body.String())
	}

	// 7. Check Leave Balance
	balanceURL := fmt.Sprintf("/api/v1/employees/%d/leave-balance?year=2026", empID)
	req = httptest.NewRequest(http.MethodGet, balanceURL, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get leave balance status = %d, body = %s", w.Code, w.Body.String())
	}

	var balResp struct {
		Success bool                                `json:"success"`
		Data    hrhttp.EmployeeLeaveSummaryResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&balResp); err != nil {
		t.Fatalf("failed to decode balance response: %v", err)
	}

	var annualBal *hrhttp.LeaveBalanceResponse
	for i := range balResp.Data.Balances {
		if balResp.Data.Balances[i].LeaveType == "annual" {
			annualBal = &balResp.Data.Balances[i]
			break
		}
	}
	if annualBal == nil || annualBal.UsedDays != 4.0 || annualBal.RemainingDays != 11.0 {
		t.Fatalf("unexpected leave balance response: %+v", annualBal)
	}
}
