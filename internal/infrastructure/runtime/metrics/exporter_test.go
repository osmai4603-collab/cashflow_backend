package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

func TestPrometheusHandler_Exposition(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterServerURL("http://127.0.0.1:8070")
	reg.RegisterDB(&fakeDB{})
	reg.ObserveHTTP("GET", "/api/v1/sale", Status2xx, 12*time.Millisecond, 300)
	reg.ObserveHTTP("GET", "/api/v1/sale", Status4xx, 5*time.Millisecond, 40)
	reg.ObserveProbe(ProbeReadyz, 3*time.Millisecond)

	h := reg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != PrometheusContentType {
		t.Errorf("expected Content-Type %q, got %q", PrometheusContentType, ct)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", cc)
	}

	// Official parser must accept the whole body without errors.
	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(rr.Body.String()))
	if err != nil {
		t.Fatalf("parser rejected exposition: %v", err)
	}

	// Every exported family must be named and valid.
	for name := range families {
		if !strings.HasPrefix(name, "cashflow_") {
			t.Errorf("unexpected family without cashflow_ prefix: %q", name)
		}
	}

	for _, want := range []string{
		MetricHTTPRequests,
		MetricHTTPRequestDuration,
		MetricHTTPResponseSize,
		MetricProbeDuration,
		MetricUptimeSeconds,
		MetricGoGoroutines,
		MetricGoGCSecondsTotal,
		MetricGoMemoryAlloc,
		MetricGoHeapInuse,
		MetricGoHeapIdle,
		MetricDBMaxConns,
		MetricDBActiveConns,
		MetricDBIdleConns,
		MetricDBWaitCount,
	} {
		if _, ok := families[want]; !ok {
			t.Errorf("missing exported family %q", want)
		}
	}

	// The counter must carry the closed label set.
	reqFam, ok := families[MetricHTTPRequests]
	if !ok {
		t.Fatal("missing HTTP requests family")
	}
	if len(reqFam.Metric) != 2 {
		t.Fatalf("expected 2 series (2xx, 4xx), got %d", len(reqFam.Metric))
	}
	for _, m := range reqFam.Metric {
		labels := map[string]string{}
		for _, lp := range m.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		if labels[LabelRoute] != "/api/v1/sale" {
			t.Errorf("unexpected route label %q", labels[LabelRoute])
		}
		var wantClass, wantCount string
		switch labels[LabelStatusClass] {
		case Status2xx:
			wantCount = "1"
		case Status4xx:
			wantCount = "1"
		default:
			t.Errorf("unexpected status_class %q", labels[LabelStatusClass])
		}
		_ = wantClass
		if got := m.GetCounter().GetValue(); got != 1 {
			t.Errorf("expected counter %s, got %v", wantCount, got)
		}
	}

	// Probe duration must carry the closed probe label.
	probeFam, ok := families[MetricProbeDuration]
	if !ok {
		t.Fatal("missing probe duration family")
	}
	if got := probeFam.Metric[0].GetLabel()[0].GetName(); got != LabelProbe {
		t.Errorf("expected probe label, got %q", got)
	}
	if got := probeFam.Metric[0].GetLabel()[0].GetValue(); got != ProbeReadyz {
		t.Errorf("expected probe value %q, got %q", ProbeReadyz, got)
	}
}

// TestPrometheusHandler_FamilyOrder asserts the category ordering decorator
// (process → go → http → probe → db) survives in the raw body.
func TestPrometheusHandler_FamilyOrder(t *testing.T) {
	reg := NewRegistry()
	// Vec-based families only materialise after an observation, so seed
	// traffic and probes to make every category emit.
	reg.ObserveHTTP("GET", "/x", Status2xx, time.Millisecond, 1)
	reg.ObserveProbe(ProbeLivez, time.Microsecond)

	h := reg.PrometheusHandler()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := rr.Body.String()
	names := []string{}
	for _, mf := range strings.Split(body, "\n") {
		if !strings.HasPrefix(mf, "# HELP ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(mf, "# HELP "))
		if len(fields) == 0 {
			continue
		}
		names = append(names, fields[0])
	}
	if len(names) == 0 {
		t.Fatal("no # HELP lines found")
	}
	ordered := []string{
		MetricUptimeSeconds,
		MetricGoGoroutines,
		MetricHTTPRequests,
		MetricProbeDuration,
		MetricDBMaxConns,
	}
	idx := map[string]int{}
	for i, n := range names {
		idx[n] = i
	}
	prev := -1
	for _, want := range ordered {
		i, ok := idx[want]
		if !ok {
			t.Errorf("family %q not found in order list", want)
			continue
		}
		if i < prev {
			t.Errorf("family %q out of category order", want)
		}
		prev = i
	}
}

// TestPrometheusHandler_EmptyRegistryIsValid locks the "empty registry" HTTP
// handler checklist item: with zero observations the exporter still serves a
// parseable text body (always-present gauge families) instead of failing or
// panicking, and the JSON handler omits the optional db section.
func TestPrometheusHandler_EmptyRegistryIsValid(t *testing.T) {
	reg := NewRegistry()

	rr := httptest.NewRecorder()
	reg.PrometheusHandler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on empty registry, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != PrometheusContentType {
		t.Fatalf("expected Prometheus content type on empty registry, got %q", ct)
	}

	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(rr.Body.String()))
	if err != nil {
		t.Fatalf("empty-registry exposition rejected by official parser: %v", err)
	}
	if len(families) == 0 {
		t.Fatal("expected at least the gauge families on an empty registry")
	}
	for _, want := range []string{
		MetricUptimeSeconds, MetricGoGoroutines, MetricGoGCSecondsTotal,
		MetricGoMemoryAlloc, MetricGoHeapInuse, MetricGoHeapIdle,
		MetricDBMaxConns, MetricDBActiveConns, MetricDBIdleConns, MetricDBWaitCount,
	} {
		if _, ok := families[want]; !ok {
			t.Errorf("missing always-present family %q on empty registry", want)
		}
	}
	for name := range families {
		if !strings.HasPrefix(name, "cashflow_") {
			t.Errorf("unexpected non-cashflow family on empty registry: %q", name)
		}
	}

	rr = httptest.NewRecorder()
	reg.jsonHandler(rr)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on empty JSON handler, got %d", rr.Code)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("empty JSON output must parse: %v", err)
	}
	if _, hasDB := m["db"]; hasDB {
		t.Error("expected db section omitted when no provider is registered")
	}
}

func TestIndexPage_ServesHTMLWithLinks(t *testing.T) {
	rr := httptest.NewRecorder()
	IndexPage().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html, got %q", ct)
	}
	body := rr.Body.String()
	for _, want := range []string{"/metrics", "/metrics/json", "/debug/pprof/"} {
		if !strings.Contains(body, want) {
			t.Errorf("index page missing link %q", want)
		}
	}
}
