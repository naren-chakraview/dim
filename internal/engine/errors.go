package engine

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
)

// ErrorType represents the classification of an error for retry purposes
type ErrorType int

const (
	// ErrorTypeUnknown indicates an error of unknown classification (fail-safe: no retry)
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeTransient indicates a transient error that should be retried
	ErrorTypeTransient
	// ErrorTypePermanent indicates a permanent error that should not be retried
	ErrorTypePermanent
)

// String returns the string representation of an ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeTransient:
		return "transient"
	case ErrorTypePermanent:
		return "permanent"
	default:
		return "unknown"
	}
}

// ClassifyError classifies an error as transient or permanent for retry logic.
// Conservative approach: unknown errors default to permanent (fail-safe, no retry).
// Transient errors (timeout, network issues, context deadline) are retried.
// Permanent errors (validation, auth, not found) fail immediately.
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// Check for context cancellation (not retryable)
	if errors.Is(err, context.Canceled) {
		return ErrorTypePermanent
	}

	// Check for context deadline exceeded (retryable as timeout)
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorTypeTransient
	}

	// Check for timeout errors (net.Error with Timeout() method)
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrorTypeTransient
	}

	// Check for temporary network errors (net.Error with Temporary() method)
	// Note: Temporary() is deprecated in Go 1.18+ but still useful for compatibility
	if errors.As(err, &netErr) && netErr.Temporary() {
		return ErrorTypeTransient
	}

	// Check for operation timeout errors
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return ErrorTypeTransient
	}

	// Check error message content for common transient patterns
	errMsg := strings.ToLower(err.Error())
	transientPatterns := []string{
		"timeout",
		"temporary",
		"connection refused",
		"connection reset",
		"broken pipe",
		"i/o timeout",
		"too many open files",
		"resource temporarily unavailable",
		"connection attempt failed",
		"remote error: handshake failure",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errMsg, pattern) {
			return ErrorTypeTransient
		}
	}

	// Default to permanent (fail-safe: don't retry unknown errors)
	return ErrorTypePermanent
}
