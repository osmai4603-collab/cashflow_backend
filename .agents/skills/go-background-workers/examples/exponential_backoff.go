package workers

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// =============================================================================
// Exponential Backoff with Full Jitter and Context Cancellation
// =============================================================================

type BackoffConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxRetries      int
}

func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		MaxRetries:      5,
	}
}

// RetryWithBackoff executes an operation, retrying on error with exponential backoff and jitter.
func RetryWithBackoff(ctx context.Context, op func(context.Context) error, cfg BackoffConfig) error {
	interval := cfg.InitialInterval
	var lastErr error

	for attempt := 0; attempt < cfg.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := op(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		// Compute jittered sleep duration: random between 0 and interval
		sleepDuration := applyFullJitter(interval)

		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		// Calculate next exponential interval bounded by MaxInterval
		next := float64(interval) * cfg.Multiplier
		if next > float64(cfg.MaxInterval) {
			interval = cfg.MaxInterval
		} else {
			interval = time.Duration(next)
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", cfg.MaxRetries, lastErr)
}

func applyFullJitter(interval time.Duration) time.Duration {
	if interval <= 0 {
		return 0
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	n := binary.BigEndian.Uint64(b[:])
	randomFraction := float64(n) / float64(^uint64(0))
	return time.Duration(randomFraction * float64(interval))
}
