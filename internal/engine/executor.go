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

	// stepNames maps step index to step type name (e.g., "filter", "translate")
	// Used for generating informative dead-letter envelopes
	// If empty, step types are inferred (or set to "unknown")
	stepNames []string
}

// NewExecutor creates a new single-worker executor.
// inputCh and outputCh must not be nil; errorCh may be nil (errors are logged but not sent anywhere).
func NewExecutor(name string, inputCh, outputCh *Channel, steps []Step) *Executor {
	return NewExecutorWithStepNames(name, inputCh, outputCh, nil, steps, nil)
}

// NewExecutorWithErrorChannel creates an executor with an error channel for failed messages.
func NewExecutorWithErrorChannel(name string, inputCh, outputCh, errorCh *Channel, steps []Step) *Executor {
	return NewExecutorWithStepNames(name, inputCh, outputCh, errorCh, steps, nil)
}

// NewExecutorWithStepNames creates an executor with explicit step type names.
// stepNames maps step indices to their type names (e.g., "filter", "translate").
// If stepNames is nil or has fewer entries than steps, missing entries are inferred or set to "unknown".
func NewExecutorWithStepNames(name string, inputCh, outputCh, errorCh *Channel, steps []Step, stepNames []string) *Executor {
	names := make([]string, len(steps))

	// Populate from provided stepNames
	for i := 0; i < len(steps) && i < len(stepNames); i++ {
		names[i] = stepNames[i]
	}

	// Fill missing entries with inferred types
	for i := len(stepNames); i < len(steps); i++ {
		names[i] = inferStepType(steps[i])
	}

	return &Executor{
		name:      name,
		inputCh:   inputCh,
		outputCh:  outputCh,
		steps:     steps,
		errorCh:   errorCh,
		stepNames: names,
	}
}

// inferStepType attempts to infer the step type from the step's concrete type.
// Returns "unknown" if the type cannot be determined.
func inferStepType(step Step) string {
	// Try to extract a meaningful type name from the type string
	// e.g., "*steps.FilterStep" -> "filter", "*steps.TranslateStep" -> "translate"
	// For now, just return "unknown" as a safe default for Phase 0
	// In M0.2+ this can be enhanced with actual type introspection
	_ = fmt.Sprintf("%T", step) // inspect type, but defer parsing to future work

	return "unknown"
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
		result, procErr, failedStepIndex := e.processMessageWithErrorTracking(ctx, msg)

		// Handle errors
		if procErr != nil {
			// If we have an error channel, send a dead-letter envelope
			if e.errorCh != nil {
				stepType := "unknown"
				if failedStepIndex >= 0 && failedStepIndex < len(e.stepNames) {
					stepType = e.stepNames[failedStepIndex]
				}

				envelope := NewDeadLetterEnvelope(msg, procErr, failedStepIndex, stepType)
				// Wrap envelope in a Message for the channel
				envelopeMsg := &Message{
					Headers: make(map[string]interface{}),
					Body:    envelope,
					Metadata: Metadata{
						CorrelationID:  msg.Metadata.CorrelationID,
						IngestedAt:     msg.Metadata.IngestedAt,
						Route:          msg.Metadata.Route,
						RouteVersion:   msg.Metadata.RouteVersion,
						ContractVersion: msg.Metadata.ContractVersion,
						Stage:          "error_path",
						Principal:      msg.Metadata.Principal,
					},
				}

				if sendErr := e.errorCh.Send(ctx, envelopeMsg); sendErr != nil {
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
	result, err, _ := e.processMessageWithErrorTracking(ctx, msg)
	return result, err
}

// processMessageWithErrorTracking is the internal implementation that tracks which step failed.
// Returns (result, error, failedStepIndex).
// failedStepIndex is -1 if no error, or the 0-based index of the failed step if an error occurred.
func (e *Executor) processMessageWithErrorTracking(ctx context.Context, msg *Message) (*Message, error, int) {
	current := msg

	// Apply each step in sequence
	for i, step := range e.steps {
		result, err := step.Execute(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("step %d in executor %q failed: %w", i, e.name, err), i
		}

		// If a step returns nil (e.g., filter rejection), stop processing and return nil
		if result == nil {
			return nil, nil, -1
		}

		current = result
	}

	return current, nil, -1
}
