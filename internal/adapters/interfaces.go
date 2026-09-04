package adapters

import (
	"context"

	"github.com/naren-chakraview/dim/internal/engine"
)

// Result represents the outcome of writing a single message.
type Result struct {
	Message *engine.Message
	Error   error
	Success bool
}

// Source is the adapter interface for reading messages from external systems.
type Source interface {
	// Start begins consuming messages from the source and sends them to the output channel.
	// Blocks until context is canceled or an error occurs.
	Start(ctx context.Context) error

	// HealthCheck returns the current health status of the source connection.
	HealthCheck(ctx context.Context) error

	// Checkpoint returns the current offset/cursor position for resume on reconnect.
	Checkpoint() (interface{}, error)

	// Close closes the source and releases resources.
	Close() error
}

// Sink is the adapter interface for writing messages to external systems.
type Sink interface {
	// Write writes one or more messages to the sink and returns results for each.
	// Returns a slice of Result, one per input message, even if an error occurs.
	Write(ctx context.Context, messages ...*engine.Message) []Result

	// HealthCheck returns the current health status of the sink connection.
	HealthCheck(ctx context.Context) error

	// Close closes the sink and releases resources.
	Close() error
}
