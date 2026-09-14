package apidesign

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// Idempotency Middleware for Financial Mutations
// =============================================================================

type cachedResponse struct {
	status    int
	body      []byte
	headers   http.Header
	inFlight  bool
	createdAt time.Time
}

type IdempotencyStore struct {
	mu      sync.RWMutex
	records map[string]*cachedResponse
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		records: make(map[string]*cachedResponse),
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Middleware returns an HTTP handler enforcing idempotency on mutating operations.
func (s *IdempotencyStore) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check mutating requests (POST, PUT, PATCH)
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			// If not provided, continue normally (or return 400 if required for this endpoint)
			next.ServeHTTP(w, r)
			return
		}

		s.mu.Lock()
		cached, exists := s.records[key]
		if exists {
			if cached.inFlight {
				s.mu.Unlock()
				http.Error(w, `{"error":"request currently in flight"}`, http.StatusConflict)
				return
			}
			s.mu.Unlock()

			// Replay cached response
			for k, v := range cached.headers {
				for _, val := range v {
					w.Header().Add(k, val)
				}
			}
			w.Header().Set("X-Cache-Lookup", "HIT-IDEMPOTENT")
			w.WriteHeader(cached.status)
			_, _ = w.Write(cached.body)
			return
		}

		// Reserve key
		s.records[key] = &cachedResponse{inFlight: true, createdAt: time.Now()}
		s.mu.Unlock()

		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		// Cache final result
		s.mu.Lock()
		s.records[key] = &cachedResponse{
			status:    recorder.statusCode,
			body:      recorder.body.Bytes(),
			headers:   w.Header().Clone(),
			inFlight:  false,
			createdAt: time.Now(),
		}
		s.mu.Unlock()
	})
}
