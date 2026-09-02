package engine

import (
	"math"
	"math/rand"
	"time"
)

const (
	// maxBackoffDuration is the maximum backoff duration to prevent excessive delays
	maxBackoffDuration = 30 * time.Second
)

// CalculateBackoff computes the exponential backoff duration for a given attempt.
// Formula: baseMs * (2 ^ (attempt - 1)) + random(0, jitterMs)
// Example:
//   - attempt 1: baseMs * 1 = baseMs + jitter
//   - attempt 2: baseMs * 2 = 2*baseMs + jitter
//   - attempt 3: baseMs * 4 = 4*baseMs + jitter
//
// The backoff duration is capped at maxBackoffDuration (30s) to prevent excessive delays.
// attempt must be >= 1; if attempt < 1, it's treated as 1.
func CalculateBackoff(attempt int, baseMs int, jitterMs int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	// Calculate exponential part: baseMs * (2 ^ (attempt - 1))
	// Using float64 to avoid integer overflow for large exponents
	exponent := float64(attempt - 1)
	exponentialMs := float64(baseMs) * math.Pow(2, exponent)

	// Add jitter: random value between 0 and jitterMs
	jitter := time.Duration(0)
	if jitterMs > 0 {
		jitter = time.Duration(rand.Intn(jitterMs)) * time.Millisecond
	}

	// Combine exponential backoff and jitter
	backoffMs := time.Duration(exponentialMs) * time.Millisecond
	totalBackoff := backoffMs + jitter

	// Cap at maximum to prevent excessive delays
	if totalBackoff > maxBackoffDuration {
		totalBackoff = maxBackoffDuration
	}

	return totalBackoff
}
