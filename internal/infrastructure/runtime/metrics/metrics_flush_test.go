package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// flushRecorder records whether the underlying writer was asked to flush.
type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed int
}

func (f *flushRecorder) Flush() {
	f.flushed++
}

func TestStatusWriter_ImplementsFlusher(t *testing.T) {
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	sw := &statusWriter{ResponseWriter: rec, status: http.StatusOK}

	if _, ok := interface{}(sw).(http.Flusher); !ok {
		t.Fatal("statusWriter must implement http.Flusher so SSE streams survive the middleware")
	}
	sw.Flush()
	if rec.flushed != 1 {
		t.Errorf("expected underlying Flush to be called once, got %d", rec.flushed)
	}
}

func TestStatusWriter_FlushWithoutPanic(t *testing.T) {
	// The wrapper must never panic when the underlying writer has no flusher;
	// when it does support flushing (httptest.ResponseRecorder) the call is
	// simply delegated.
	sw := &statusWriter{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	sw.Flush()
}

func TestStatusWriter_FlushInMiddlewareStream(t *testing.T) {
	reg := NewRegistry()
	handler := reg.httpMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("handler should see a writer that supports Flush through the metrics middleware")
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(": connected\n\n"))
		f.Flush()
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/stream", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for streaming handler, got %d", rr.Code)
	}
}