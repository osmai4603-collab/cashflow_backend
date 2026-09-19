package examples

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// CircuitState represents the current operating state of the breaker.
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open: request rejected")
)

// CircuitBreaker guards remote calls against cascading failures.
type CircuitBreaker struct {
	mu           sync.RWMutex
	state        CircuitState
	failureCount int
	threshold    int
	cooldown     time.Duration
	openedAt     time.Time
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     StateClosed,
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// Execute wraps an external network operation with circuit breaker state management.
func (cb *CircuitBreaker) Execute(ctx context.Context, op func(ctx context.Context) error) error {
	cb.mu.Lock()
	if cb.state == StateOpen {
		if time.Since(cb.openedAt) > cb.cooldown {
			cb.state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	cb.mu.Unlock()

	err := op(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		if cb.failureCount >= cb.threshold || cb.state == StateHalfOpen {
			cb.state = StateOpen
			cb.openedAt = time.Now()
		}
		return err
	}

	// Operation succeeded
	cb.failureCount = 0
	cb.state = StateClosed
	return nil
}

// ResilientCaller combines circuit breaking with exponential backoff and full jitter.
type ResilientCaller struct {
	cb         *CircuitBreaker
	httpClient *http.Client
	maxRetries int
	baseDelay  time.Duration
}

func NewResilientCaller(threshold int, cooldown time.Duration, maxRetries int, baseDelay time.Duration) *ResilientCaller {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &ResilientCaller{
		cb: NewCircuitBreaker(threshold, cooldown),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		},
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// DoIdempotentRequest executes a safe GET or idempotent request with retries and jitter.
func (rc *ResilientCaller) DoIdempotentRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response

	for attempt := 0; attempt <= rc.maxRetries; attempt++ {
		err := rc.cb.Execute(ctx, func(execCtx context.Context) error {
			r := req.Clone(execCtx)
			res, doErr := rc.httpClient.Do(r)
			if doErr != nil {
				return doErr
			}
			if res.StatusCode >= 500 {
				_ = res.Body.Close()
				return fmt.Errorf("remote server error: %d", res.StatusCode)
			}
			resp = res
			return nil
		})

		if err == nil {
			return resp, nil
		}

		if errors.Is(err, ErrCircuitOpen) {
			return nil, err
		}

		if attempt == rc.maxRetries {
			return nil, fmt.Errorf("request failed after %d attempts: %w", rc.maxRetries, err)
		}

		// Calculate Exponential Backoff with Full Jitter
		multiplier := 1 << attempt
		maxDelay := rc.baseDelay * time.Duration(multiplier)
		jitter := time.Duration(rand.Int63n(int64(maxDelay)))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(jitter):
		}
	}

	return nil, errors.New("exhausted retries")
}
