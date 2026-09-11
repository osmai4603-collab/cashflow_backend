package zatca

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientClearInvoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/invoices/clearance/single" || r.Header.Get("Authorization") != "Basic token" {
			t.Errorf("unexpected request: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"reportingStatus":"CLEARED"}`))
	}))
	defer server.Close()
	response, err := NewClient(server.URL, "token", server.Client()).ClearInvoice(context.Background(), &ClearanceRequest{InvoiceHash: "hash", UUID: "uuid"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != "CLEARED" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
