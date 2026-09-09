package odoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDatabaseServiceList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/web/database/list" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":["cashflow","demo"]}`))
	}))
	defer server.Close()

	service := NewDatabaseService(server.URL, "admin", 0)
	databases, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list databases: %v", err)
	}
	if len(databases) != 2 || databases[0] != "cashflow" || databases[1] != "demo" {
		t.Fatalf("unexpected databases: %#v", databases)
	}
}

func TestDatabaseServiceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/web/database/create" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		want := url.Values{
			"master_pwd":   {"admin"},
			"name":         {"demo"},
			"lang":         {"ar_SY"},
			"password":     {"secret"},
			"login":        {"admin@example.com"},
			"phone":        {"+966500000000"},
			"country_code": {"SA"},
			"demo":         {"1"},
		}
		if r.Form.Encode() != want.Encode() {
			t.Fatalf("unexpected form: %s", r.Form.Encode())
		}
		w.WriteHeader(http.StatusSeeOther)
	}))
	defer server.Close()

	err := NewDatabaseService(server.URL, "admin", 0).Create(
		context.Background(),
		CreateDatabaseInput{
			MasterPassword: "admin",
			Name:           "demo",
			Language:       "ar_SY",
			Password:       "secret",
			Login:          "admin@example.com",
			Phone:          "+966500000000",
			CountryCode:    "SA",
			Demo:           true,
		},
	)
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
}