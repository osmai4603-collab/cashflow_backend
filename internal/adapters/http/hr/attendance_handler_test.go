package hrhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	hrhttp "cashflow_backend/internal/adapters/http/hr"
	"cashflow_backend/internal/domain/hr"

	"github.com/go-chi/chi/v5"
)

type mockAttendanceUC struct {
	checkInCalled  bool
	checkOutCalled bool
}

func (m *mockAttendanceUC) CheckIn(ctx context.Context, employeeID int64, info hr.Attendance) (*hr.Attendance, error) {
	m.checkInCalled = true
	return &hr.Attendance{ID: 1, EmployeeID: employeeID, CheckIn: time.Now()}, nil
}

func (m *mockAttendanceUC) CheckOut(ctx context.Context, employeeID int64, info hr.Attendance) (*hr.Attendance, error) {
	m.checkOutCalled = true
	now := time.Now()
	return &hr.Attendance{ID: 1, EmployeeID: employeeID, CheckOut: &now}, nil
}

func (m *mockAttendanceUC) GetKioskConfig(ctx context.Context, companyID int64) (map[string]any, error) {
	return map[string]any{"kiosk_mode": "barcode"}, nil
}

func (m *mockAttendanceUC) ApproveOvertime(ctx context.Context, lineID int64, managerID int64) error {
	return nil
}

func (m *mockAttendanceUC) RefuseOvertime(ctx context.Context, lineID int64, managerID int64) error {
	return nil
}

func (m *mockAttendanceUC) GetAttendanceReport(ctx context.Context, employeeID int64, from, to time.Time) ([]hr.Attendance, error) {
	return []hr.Attendance{}, nil
}

func TestAttendanceHandler_Endpoints(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockUC := &mockAttendanceUC{}
	h := hrhttp.NewHandler(nil, mockUC, logger)

	r := chi.NewRouter()
	r.Post("/check-in", h.CheckIn)
	r.Post("/check-out", h.CheckOut)
	r.Get("/kiosk-config", h.GetKioskConfig)

	t.Run("CheckIn", func(t *testing.T) {
		reqBody := hrhttp.CheckInRequest{Mode: "manual"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/check-in?employee_id=100", bytes.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !mockUC.checkInCalled {
			t.Error("expected CheckIn usecase to be called")
		}
	})

	t.Run("CheckOut", func(t *testing.T) {
		reqBody := hrhttp.CheckOutRequest{Mode: "manual"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/check-out?employee_id=100", bytes.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !mockUC.checkOutCalled {
			t.Error("expected CheckOut usecase to be called")
		}
	})
}
