package mlflow

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// retryConfig holds the retry configuration for the client.
type retryConfig struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	jitter       float64
}

// defaultRetryConfig returns the default retry configuration.
func defaultRetryConfig() retryConfig {
	return retryConfig{
		initialDelay: 1 * time.Second,
		maxDelay:     10 * time.Second,
		multiplier:   2.0,
		jitter:       0.1, // ±10%
	}
}

// shouldRetry determines if a request should be retried based on the error.
func shouldRetry(err error, attempt int, maxRetries int) bool {
	if attempt >= maxRetries {
		return false
	}

	// Check if it's an APIError
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsRetryable()
	}

	// Check if it's a ConnectionError
	if connErr, ok := err.(*ConnectionError); ok {
		return connErr.IsRetryable()
	}

	// Check if it's a network error
	if isNetworkError(err) {
		return true
	}

	return false
}

// calculateBackoff calculates the backoff duration for a given attempt.
// It uses exponential backoff with jitter to prevent thundering herd.
func calculateBackoff(cfg retryConfig, attempt int) time.Duration {
	// Calculate exponential delay
	delay := float64(cfg.initialDelay) * math.Pow(cfg.multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(cfg.maxDelay) {
		delay = float64(cfg.maxDelay)
	}

	// Apply jitter (±10%)
	jitterAmount := delay * cfg.jitter
	jitterRange := jitterAmount * 2
	jitterOffset := rand.Float64()*jitterRange - jitterAmount

	finalDelay := delay + jitterOffset

	// Ensure we don't go below zero
	if finalDelay < 0 {
		finalDelay = float64(cfg.initialDelay)
	}

	return time.Duration(finalDelay)
}

// sleep waits for the specified duration, respecting context cancellation.
func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
