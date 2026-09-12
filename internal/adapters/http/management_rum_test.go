package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platconfig "cashflow_backend/internal/platform/config"
)

func postRUM(t *testing.T, router http.Handler, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestRUMHandler_IngestsValidSample(t *testing.T) {
	router := managementTestServer(t, nil)

	body := `{
	  "client_type": "web",
	  "ttfb_ms": 150,
	  "lcp_ms": 900,
	  "inp_ms": 18,
	  "cls": 0.023,
	  "dom_interactive_ms": 480,
	  "app_version": "v1.2.0"
	}`
	rr := postRUM(t, router, "/rum", body)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (%s)", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Errorf("expected empty body on 204, got %q", rr.Body.String())
	}

	// The samples must be visible on the text exporter with the closed label.
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	text := rr.Body.String()
	for _, want := range []string{
		"cashflow_rum_ttfb_bucket{client_type=\"web\"",
		"cashflow_rum_lcp_bucket{client_type=\"web\"",
		"cashflow_rum_inp_bucket{client_type=\"web\"",
		"cashflow_rum_cls_bucket{client_type=\"web\"",
		"cashflow_rum_dom_interactive_bucket{client_type=\"web\"",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %s in /metrics output", want)
		}
	}
}

func TestRUMHandler_ClientTypeClosedSet(t *testing.T) {
	cases := []struct{ name, want string }{
		{"web", "web"},
		{"desktop", "desktop"},
		{"mobile", "mobile"},
		{"Mobile", "mobile"}, // normalised case-insensitively
	}
	for _, tc := range cases {
		router := managementTestServer(t, nil)
		body, _ := json.Marshal(map[string]any{"client_type": tc.name, "inp_ms": 22})
		rr := postRUM(t, router, "/rum", string(body))
		if rr.Code != http.StatusNoContent {
			t.Errorf("client_type=%q: expected 204, got %d", tc.name, rr.Code)
			continue
		}
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if !strings.Contains(rr.Body.String(), "client_type=\""+tc.want+"\"") {
			t.Errorf("client_type=%q: expected label %q in /metrics", tc.name, tc.want)
		}
	}

	router := managementTestServer(t, nil)
	for _, bad := range []string{"tablet", "", "car,horse"} {
		body, _ := json.Marshal(map[string]any{"client_type": bad, "inp_ms": 22})
		rr := postRUM(t, router, "/rum", string(body))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("client_type=%q: expected 400, got %d", bad, rr.Code)
		}
	}
}

func TestRUMHandler_ValidationFailures(t *testing.T) {
	router := managementTestServer(t, nil)

	cases := []struct {
		name     string
		body     string
		ct       string
		wantCode int
	}{
		{name: "no metrics", body: `{"client_type":"web"}`},
		{name: "no client type", body: `{"inp_ms":22}`},
		{name: "negative value", body: `{"client_type":"web","lcp_ms":-1}`},
		{name: "non-finite exponent", body: `{"client_type":"web","lcp_ms":1e999}`},
		{name: "malformed json", body: `{"client_type":"web","lcp_ms":`},
	}
	for _, tc := range cases {
		rr := postRUM(t, router, "/rum", tc.body)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", tc.name, rr.Code)
		}
		var e map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &e); err != nil {
			t.Errorf("%s: expected JSON error body, got %q", tc.name, rr.Body.String())
		}
	}

	rr := postRUM(t, router, "/rum", `{"client_type":"web","inp_ms":22}`)
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rum", strings.NewReader(`{"client_type":"web","inp_ms":22}`))
	req.Header.Set("Content-Type", "text/plain")
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Errorf("wrong content type: expected 415, got %d", rr.Code)
	}

	rr = postRUM(t, router, "/rum", `{"client_type":"web","inp_ms":22} "extra garbage"`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("trailing JSON: expected 400, got %d", rr.Code)
	}
}

func TestRUMHandler_PayloadTooLarge(t *testing.T) {
	router := managementTestServer(t, nil)
	body := `{"client_type":"web","inp_ms":22,` + strings.Repeat(`"x":0,`, 7000) + `"end":1}`
	rr := postRUM(t, router, "/rum", body)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rr.Code)
	}
}

func TestRUMHandler_UnknownFieldsTolerated(t *testing.T) {
	router := managementTestServer(t, nil)
	body := `{"client_type":"desktop","inp_ms":12,"brand":"cashflow","nested":{"a":1}}`
	rr := postRUM(t, router, "/rum", body)
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 with unknown top-level fields, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestRUMHandler_RespectsAuthAndMetricsFlag(t *testing.T) {
	router := managementTestServer(t, func(cfg *platconfig.Configuration) {
		cfg.Management.RequireAuth = true
		cfg.Management.AuthToken = "rum-token"
	})
	body := `{"client_type":"web","inp_ms":22}`
	if rr := postRUM(t, router, "/rum", body); rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rr.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/rum", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer rum-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 with token, got %d (%s)", rr.Code, rr.Body.String())
	}

	router = managementTestServer(t, func(cfg *platconfig.Configuration) {
		cfg.Management.MetricsEnabled = false
	})
	if rr := postRUM(t, router, "/rum", body); rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when metrics disabled, got %d", rr.Code)
	}
}
