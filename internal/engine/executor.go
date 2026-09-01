package engine

import (
	"context"
	"fmt"
)

// Step is the interface that all pipeline steps must implement.
// A step receives a message, performs an operation (filter, translate, route, etc.),
// and returns either a transformed message or an error.
type Step interface {
	// Execute processes a message through this step.
	// It returns the (possibly modified) message to pass downstream,
	// or an error if the step fails.
	// If the step needs to drop a message (e.g., filter predicate fails),
	// it returns nil as the message and no error.
	Execute(ctx context.Context, msg *Message) (*Message, error)
}

// Executor is a single-worker message processor that runs messages through a pipeline of steps.
// It reads from an input channel, applies each configured step in order,
// and sends results to an output channel.
// No concurrency; messages are processed serially (M0.2 will add worker pools).
type Executor struct {
	// name is a diagnostic identifier for this executor
	name string

	// inputCh receives messages to process
	inputCh *Channel

	// outputCh receives successfully processed messages
	outputCh *Channel

	// steps are applied to each message in order
	steps []Step

	// errorCh receives messages that fail processing
	// nil if error handling is not configured for this executor
	errorCh *Channel
}

// NewExecutor creates a new single-worker executor.
// inputCh and outputCh must not be nil; errorCh may be nil (errors are logged but not sent anywhere).
func NewExecutor(name string, inputCh, outputCh *Channel, steps []Step) *Executor {
	return &Executor{
		name:     name,
		inputCh:  inputCh,
		outputCh: outputCh,
		steps:    steps,
		errorCh:  nil,
	}
}

// NewExecutorWithErrorChannel creates an executor with an error channel for failed messages.
func NewExecutorWithErrorChannel(name string, inputCh, outputCh, errorCh *Channel, steps []Step) *Executor {
	return &Executor{
		name:     name,
		inputCh:  inputCh,
		outputCh: outputCh,
		steps:    steps,
		errorCh:  errorCh,
	}
}

// Run starts the executor's main loop.
// It continuously reads messages from inputCh, processes them through the configured steps,
// and sends results to outputCh (or errorCh if processing fails).
// Run blocks until ctx is cancelled.
func (e *Executor) Run(ctx context.Context) error {
	for {
		// Read a message from the input channel
		msg, err := e.inputCh.Recv(ctx)
		if err != nil {
			return fmt.Errorf("executor %q recv error: %w", e.name, err)
		}

		// If the channel is closed and returns nil, exit gracefully
		if msg == nil {
			return nil
		}

		// Process the message through all steps
		result, procErr := e.processMessage(ctx, msg)

		// Handle errors
		if procErr != nil {
			// If we have an error channel, send the message there
			if e.errorCh != nil {
				if sendErr := e.errorCh.Send(ctx, msg); sendErr != nil {
					return fmt.Errorf("executor %q error send failed: %w", e.name, sendErr)
				}
			}
			// Continue processing next message rather than stopping
			continue
		}

		// If a step returned nil (e.g., filter rejection), don't send to output
		if result == nil {
			continue
		}

		// Send successful result to output
		if err := e.outputCh.Send(ctx, result); err != nil {
			return fmt.Errorf("executor %q output send failed: %w", e.name, err)
		}
	}
}

// ProcessMessage processes a single message through all configured steps.
// This is useful for testing individual message processing without running the full loop.
// Returns the final message (may be nil if a step drops it) or an error if any step fails.
func (e *Executor) ProcessMessage(ctx context.Context, msg *Message) (*Message, error) {
	return e.processMessage(ctx, msg)
}

// processMessage is the internal implementation of message processing.
func (e *Executor) processMessage(ctx context.Context, msg *Message) (*Message, error) {
	current := msg

	// Apply each step in sequence
	for i, step := range e.steps {
		result, err := step.Execute(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("step %d in executor %q failed: %w", i, e.name, err)
		}

		// If a step returns nil (e.g., filter rejection), stop processing and return nil
		if result == nil {
			return nil, nil
		}

		current = result
	}

	return current, nil
}
