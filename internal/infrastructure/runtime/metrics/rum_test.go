package metrics

import (
	"math"
	"testing"

	dto "github.com/prometheus/client_model/go"
)

func TestObserveRUM_RecordsFamilyAndClosedLabels(t *testing.T) {
	r := NewRegistry()

	cases := []struct {
		family, clientType string
		value              float64
	}{
		{MetricRUMTTFB, ClientTypeWeb, 0.150},
		{MetricRUMLCP, ClientTypeDesktop, 1.234},
		{MetricRUMINP, ClientTypeMobile, 0.007},
		{MetricRUMCLS, ClientTypeWeb, 0.023},
		{MetricRUMDOMInteractive, ClientTypeMobile, 0.456},
	}
	for _, tc := range cases {
		if err := r.observeRUM(tc.family, tc.clientType, tc.value); err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.family, err)
		}
	}

	families, err := r.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, tc := range cases {
		fam := findFamily(families, tc.family)
		if fam == nil {
			t.Fatalf("expected family %s in gather", tc.family)
		}
		if len(fam.Metric) != 1 {
			t.Fatalf("%s: expected 1 time series, got %d", tc.family, len(fam.Metric))
		}
		m := fam.Metric[0]
		if len(m.Label) != 1 || m.Label[0].GetName() != LabelClientType || m.Label[0].GetValue() != tc.clientType {
			t.Errorf("%s: expected sole label %s=%q, got %+v", tc.family, LabelClientType, tc.clientType, m.Label)
		}
		h := m.GetHistogram()
		if h.GetSampleCount() != 1 || h.GetSampleSum() != tc.value {
			t.Errorf("%s: expected count=1 sum=%v, got count=%d sum=%v",
				tc.family, tc.value, h.GetSampleCount(), h.GetSampleSum())
		}
	}
}

func TestObserveRUM_RejectsInvalidInputWithoutPoisoningLabels(t *testing.T) {
	r := NewRegistry()

	if err := r.observeRUM(MetricRUMTTFB, ClientTypeWeb, 0.5); err != nil {
		t.Fatalf("seed sample: %v", err)
	}

	bad := []struct {
		family, clientType string
		value              float64
	}{
		{"not_a_rum_family", ClientTypeWeb, 0.5},
		{MetricRUMTTFB, "tablet", 0.5},
		{MetricRUMTTFB, ClientTypeWeb, -0.5},
		{MetricRUMTTFB, ClientTypeWeb, math.NaN()},
		{MetricRUMTTFB, ClientTypeWeb, math.Inf(1)},
	}
	for _, tc := range bad {
		if err := r.observeRUM(tc.family, tc.clientType, tc.value); err == nil {
			t.Errorf("%s/%s/%v: expected an error", tc.family, tc.clientType, tc.value)
		}
	}

	families, err := r.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, f := range families {
		for _, m := range f.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == LabelClientType && lp.GetValue() != ClientTypeWeb {
					t.Errorf("unexpected client_type %q leaked from a rejected sample into %s",
						lp.GetValue(), f.GetName())
				}
			}
		}
	}

	fam := findFamily(families, MetricRUMTTFB)
	if fam == nil || len(fam.Metric) != 1 {
		t.Fatalf("expected exactly the seeded series to remain, got %d", len(fam.GetMetric()))
	}
	if fam.Metric[0].GetHistogram().GetSampleCount() != 1 {
		t.Errorf("rejected samples changed the stored count")
	}
}

func TestValidClientType_ClosedSet(t *testing.T) {
	for _, ok := range []string{ClientTypeWeb, ClientTypeDesktop, ClientTypeMobile} {
		if !ValidClientType(ok) {
			t.Errorf("expected %q to be valid", ok)
		}
	}
	for _, bad := range []string{"tablet", "car", "", "Web"} {
		if ValidClientType(bad) {
			t.Errorf("expected %q to be invalid", bad)
		}
	}
}

func findFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}
