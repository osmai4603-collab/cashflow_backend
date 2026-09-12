package httpadapter

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"strings"

	"cashflow_backend/internal/infrastructure/runtime/metrics"
)

// maxRUMBodyBytes caps the telemetry payload; a full sample is a few hundred
// bytes so 16 KiB is generous.
const maxRUMBodyBytes = 16 << 10

// rumFieldToMetric maps the documented API keys onto RUM metric families.
// ms-valued fields are converted to seconds on ingestion; cls is a
// dimensionless layout-shift score and is stored as-is.
var rumFieldToMetric = map[string]string{
	"ttfb_ms":            metrics.MetricRUMTTFB,
	"lcp_ms":             metrics.MetricRUMLCP,
	"inp_ms":             metrics.MetricRUMINP,
	"cls":                metrics.MetricRUMCLS,
	"dom_interactive_ms": metrics.MetricRUMDOMInteractive,
}

// rumPayload mirrors the ingestion contract. Pointer fields distinguish
// "absent" from a legitimate zero; all are optional but at least one must be
// present, and client_type is required.
type rumPayload struct {
	ClientType       string   `json:"client_type"`
	TTFBMs           *float64 `json:"ttfb_ms,omitempty"`
	LCPMs            *float64 `json:"lcp_ms,omitempty"`
	INPMs            *float64 `json:"inp_ms,omitempty"`
	CLS              *float64 `json:"cls,omitempty"`
	DOMInteractiveMs *float64 `json:"dom_interactive_ms,omitempty"`

	unknown map[string]json.RawMessage
}

// UnmarshalJSON captures unknown top-level fields so they are tolerated (but
// never turned into labels) for forward compatibility.
func (p *rumPayload) UnmarshalJSON(data []byte) error {
	type alias rumPayload
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*p = rumPayload(a)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key := range raw {
		switch key {
		case "client_type", "ttfb_ms", "lcp_ms", "inp_ms", "cls", "dom_interactive_ms":
			continue
		}
		if p.unknown == nil {
			p.unknown = make(map[string]json.RawMessage)
		}
		p.unknown[key] = raw[key]
	}
	return nil
}

// NewRUMHandler validates and ingests real-user-monitoring samples (P7). It
// lives on the management listener, so it inherits the Bearer token policy.
//
// Success returns 204 with an empty body; validation failures return a JSON
// error body and one of 400/413/415. Unknown payload fields are ignored and
// never exported as labels.
func NewRUMHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeRUMError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRUMBodyBytes)

		dec := json.NewDecoder(r.Body)
		var payload rumPayload
		if err := dec.Decode(&payload); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeRUMError(w, http.StatusRequestEntityTooLarge, "payload exceeds size limit")
				return
			}
			writeRUMError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
		var trailing json.RawMessage
		if err := dec.Decode(&trailing); err != io.EOF {
			writeRUMError(w, http.StatusBadRequest, "unexpected data after JSON payload")
			return
		}

		clientType := strings.ToLower(strings.TrimSpace(payload.ClientType))
		if !metrics.ValidClientType(clientType) {
			writeRUMError(w, http.StatusBadRequest, "client_type must be one of web, desktop, mobile")
			return
		}

		fields := map[string]*float64{
			"ttfb_ms":            payload.TTFBMs,
			"lcp_ms":             payload.LCPMs,
			"inp_ms":             payload.INPMs,
			"cls":                payload.CLS,
			"dom_interactive_ms": payload.DOMInteractiveMs,
		}

		observed := 0
		for field, fp := range fields {
			if fp == nil {
				continue
			}
			value := *fp
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				writeRUMError(w, http.StatusBadRequest, field+" must be a non-negative finite number")
				return
			}
			observed++
		}
		if observed == 0 {
			writeRUMError(w, http.StatusBadRequest, "at least one metric field is required")
			return
		}

		for field, fp := range fields {
			if fp == nil {
				continue
			}
			family := rumFieldToMetric[field]
			value := *fp
			if field != "cls" {
				value /= 1000
			}
			if err := metrics.ObserveRUM(family, clientType, value); err != nil {
				if logger != nil {
					logger.Error("rum ingestion rejected",
						"error", err,
						"client_type", clientType,
						"field", field,
					)
				}
				writeRUMError(w, http.StatusInternalServerError, "internal ingestion error")
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func writeRUMError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
