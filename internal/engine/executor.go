package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
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

// RetryPolicy defines retry behavior for failed steps
type RetryPolicy struct {
	MaxAttempts int // Maximum number of retry attempts (>= 1)
	BackoffMs   int // Base backoff in milliseconds (>= 0, default 1000)
	JitterMs    int // Jitter range in milliseconds (>= 0, default 100)
}

// Executor is a configurable-worker message processor that runs messages through a pipeline of steps.
// It reads from an input channel, applies each configured step in order,
// and sends results to an output channel.
// Supports 1 to N workers for concurrent processing (M0.2+).
// Supports configurable retry logic with exponential backoff (M0.2.2+).
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

	// numWorkers is the number of concurrent workers (default 1)
	// M0.1 compatible: 1 worker runs serially
	// M0.2+: configurable 1-N workers for concurrent processing
	numWorkers int

	// retryPolicy defines retry behavior for transient errors (M0.2.2+)
	// nil means no retry (MaxAttempts = 1, fail-fast behavior)
	retryPolicy *RetryPolicy

	// inFlightCount tracks active messages across all workers
	// Incremented on message receipt, decremented on completion
	inFlightCount int32
}

// NewExecutor creates a new single-worker executor (backward compatible with M0.1).
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
	return NewExecutorWithWorkers(name, inputCh, outputCh, errorCh, steps, stepNames, 1)
}

// NewExecutorWithWorkers creates an executor with configurable worker count (M0.2+).
// numWorkers must be >= 1; defaults to 1 if invalid (backward compatible).
// Workers share the steps slice (read-only) and coordinate via inputCh and outputCh.
func NewExecutorWithWorkers(name string, inputCh, outputCh, errorCh *Channel, steps []Step, stepNames []string, numWorkers int) *Executor {
	return NewExecutorWithRetry(name, inputCh, outputCh, errorCh, steps, stepNames, numWorkers, nil)
}

// NewExecutorWithRetry creates an executor with retry policy support (M0.2.2+).
// retryPolicy may be nil (no retry, MaxAttempts=1).
// retryPolicy fields: MaxAttempts (>= 1), BackoffMs (default 1000), JitterMs (default 100)
// numWorkers must be >= 1; defaults to 1 if invalid (backward compatible).
func NewExecutorWithRetry(name string, inputCh, outputCh, errorCh *Channel, steps []Step, stepNames []string, numWorkers int, retryPolicy *RetryPolicy) *Executor {
	if numWorkers < 1 {
		numWorkers = 1
	}

	names := make([]string, len(steps))

	// Populate from provided stepNames
	for i := 0; i < len(steps) && i < len(stepNames); i++ {
		names[i] = stepNames[i]
	}

	// Fill missing entries with inferred types
	for i := len(stepNames); i < len(steps); i++ {
		names[i] = inferStepType(steps[i])
	}

	// Normalize retry policy: nil or invalid defaults to no retry
	normalizedPolicy := normalizeRetryPolicy(retryPolicy)

	return &Executor{
		name:        name,
		inputCh:     inputCh,
		outputCh:    outputCh,
		steps:       steps,
		errorCh:     errorCh,
		stepNames:   names,
		numWorkers:  numWorkers,
		retryPolicy: normalizedPolicy,
	}
}

// normalizeRetryPolicy ensures retry policy has valid defaults
// nil policy returns nil (no retry)
// policy with MaxAttempts < 1 is treated as no retry (returns nil)
// policy with BackoffMs < 0 defaults to 1000ms
// policy with JitterMs < 0 defaults to 100ms
func normalizeRetryPolicy(policy *RetryPolicy) *RetryPolicy {
	if policy == nil {
		return nil
	}

	// No retry if max attempts < 1
	if policy.MaxAttempts < 1 {
		return nil
	}

	// Set defaults for backoff and jitter
	backoffMs := policy.BackoffMs
	if backoffMs <= 0 {
		backoffMs = 1000 // Default 1 second
	}

	jitterMs := policy.JitterMs
	if jitterMs < 0 {
		jitterMs = 100 // Default 100ms jitter
	}

	return &RetryPolicy{
		MaxAttempts: policy.MaxAttempts,
		BackoffMs:   backoffMs,
		JitterMs:    jitterMs,
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

// Run starts the executor's worker pool.
// It spawns numWorkers goroutines that concurrently read messages from inputCh,
// process them through the configured steps, and send results to outputCh or errorCh.
//
// Lifecycle:
// 1. Spawn N worker goroutines
// 2. Each worker reads from inputCh until closed
// 3. On context cancel: stop accepting new messages, wait for workers to finish (30s timeout)
// 4. Return nil on successful shutdown, error on timeout or internal error
//
// Backpressure: Handled by inputCh buffer size (bounded, FIFO fairness).
// Message ordering: Preserved at channel level; workers process in parallel but dequeue FIFO.
func (e *Executor) Run(ctx context.Context) error {
	log.Printf("[DEBUG] Executor %q starting with %d workers", e.name, e.numWorkers)

	var wg sync.WaitGroup
	wg.Add(e.numWorkers)

	// Spawn worker goroutines
	for workerID := 1; workerID <= e.numWorkers; workerID++ {
		go e.worker(ctx, workerID, &wg)
	}

	// Wait for all workers to finish
	wg.Wait()

	// After all workers exit, wait for in-flight messages to complete
	deadline := time.Now().Add(30 * time.Second)
	for {
		inFlight := int(atomic.LoadInt32(&e.inFlightCount))
		if inFlight == 0 {
			log.Printf("[DEBUG] Executor %q shutdown complete, all workers exited", e.name)
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("executor %q graceful shutdown timeout: %d messages still in flight", e.name, inFlight)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// worker is the main loop for a single worker goroutine.
// It reads messages from inputCh, processes them, and sends to output/error channels.
// Exits when inputCh is closed (receives nil).
func (e *Executor) worker(ctx context.Context, workerID int, wg *sync.WaitGroup) {
	defer func() {
		log.Printf("[DEBUG] Executor %q worker %d exiting", e.name, workerID)
		wg.Done()
	}()

	log.Printf("[DEBUG] Executor %q worker %d started", e.name, workerID)

	for {
		// Read a message from the input channel
		msg, err := e.inputCh.Recv(ctx)
		if err != nil {
			// Context cancelled or channel error; exit gracefully
			log.Printf("[DEBUG] Executor %q worker %d recv error: %v", e.name, workerID, err)
			return
		}

		// If the channel is closed, msg is nil; exit gracefully
		if msg == nil {
			return
		}

		// Increment in-flight counter
		atomic.AddInt32(&e.inFlightCount, 1)

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

				// Extract retry attempt count and original error
				retryAttempt := 0
				actualErr := procErr
				if see, ok := procErr.(*stepExecutionError); ok {
					retryAttempt = see.retryAttempt
					actualErr = see.originalError
				}

				envelope := NewDeadLetterEnvelopeWithRetryAttempt(msg, actualErr, failedStepIndex, stepType, retryAttempt)
				// Wrap envelope in a Message for the channel
				envelopeMsg := &Message{
					Headers: make(map[string]interface{}),
					Body:    envelope,
					Metadata: Metadata{
						CorrelationID:   msg.Metadata.CorrelationID,
						IngestedAt:      msg.Metadata.IngestedAt,
						Route:           msg.Metadata.Route,
						RouteVersion:    msg.Metadata.RouteVersion,
						ContractVersion: msg.Metadata.ContractVersion,
						Stage:           "error_path",
						Principal:       msg.Metadata.Principal,
					},
				}

				if sendErr := e.errorCh.Send(ctx, envelopeMsg); sendErr != nil {
					log.Printf("[DEBUG] Executor %q worker %d error send failed: %v", e.name, workerID, sendErr)
					// Decrement in-flight on error
					atomic.AddInt32(&e.inFlightCount, -1)
					return
				}
			}
			// Decrement in-flight and continue processing next message
			atomic.AddInt32(&e.inFlightCount, -1)
			continue
		}

		// If a step returned nil (e.g., filter rejection), don't send to output
		if result == nil {
			// Decrement in-flight and continue
			atomic.AddInt32(&e.inFlightCount, -1)
			continue
		}

		// Send successful result to output
		if err := e.outputCh.Send(ctx, result); err != nil {
			log.Printf("[DEBUG] Executor %q worker %d output send failed: %v", e.name, workerID, err)
			// Decrement in-flight on error
			atomic.AddInt32(&e.inFlightCount, -1)
			return
		}

		// Decrement in-flight on successful send
		atomic.AddInt32(&e.inFlightCount, -1)
	}
}

// stepExecutionError wraps an error with retry attempt tracking
// This is used to pass retry attempt count from executeStepWithRetry to error handling
type stepExecutionError struct {
	originalError error
	retryAttempt  int
	stepIndex     int
}

// Error implements the error interface
func (see *stepExecutionError) Error() string {
	return see.originalError.Error()
}

// Unwrap returns the underlying error
func (see *stepExecutionError) Unwrap() error {
	return see.originalError
}

// ProcessMessage processes a single message through all configured steps.
// This is useful for testing individual message processing without running the full loop.
// Returns the final message (may be nil if a step drops it) or an error if any step fails.
func (e *Executor) ProcessMessage(ctx context.Context, msg *Message) (*Message, error) {
	result, err, _ := e.processMessageWithErrorTracking(ctx, msg)
	return result, err
}

// InFlightCount returns the number of messages currently being processed across all workers.
func (e *Executor) InFlightCount() int {
	return int(atomic.LoadInt32(&e.inFlightCount))
}

// GetInputChannel returns the executor's input channel.
// Used for routing messages from a source to this executor.
func (e *Executor) GetInputChannel() *Channel {
	return e.inputCh
}

// executeStepWithRetry executes a step with configurable retry logic.
// On transient errors: retries with exponential backoff
// On permanent errors: returns immediately (no retry)
// On success: returns the result
//
// Returns:
// - result: the output message (may be nil if step dropped it)
// - error: any error that occurred (wrapped with retry attempt count)
// - retryAttempts: number of retry attempts made (0 if no retry)
func (e *Executor) executeStepWithRetry(ctx context.Context, step Step, msg *Message) (*Message, error, int) {
	// Determine max attempts from retry policy
	maxAttempts := 1 // Default: no retry
	if e.retryPolicy != nil && e.retryPolicy.MaxAttempts > 1 {
		maxAttempts = e.retryPolicy.MaxAttempts
	}

	// Get backoff parameters
	backoffMs := 1000
	jitterMs := 100
	if e.retryPolicy != nil {
		backoffMs = e.retryPolicy.BackoffMs
		jitterMs = e.retryPolicy.JitterMs
	}

	var lastErr error
	var retryAttempts int

	// Try executing the step up to maxAttempts times
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check if context is cancelled before attempting
		select {
		case <-ctx.Done():
			return nil, ctx.Err(), retryAttempts
		default:
		}

		// Execute the step
		result, err := step.Execute(ctx, msg)

		// Success: return immediately
		if err == nil {
			return result, nil, retryAttempts
		}

		// Classify the error
		errType := ClassifyError(err)

		// Log the attempt
		if attempt > 1 || maxAttempts > 1 {
			log.Printf("[DEBUG] Executor %q step execution attempt %d/%d failed (error type: %s): %v",
				e.name, attempt, maxAttempts, errType, err)
		}

		// If it's a permanent error, don't retry
		if errType == ErrorTypePermanent {
			return nil, &stepExecutionError{originalError: err, retryAttempt: retryAttempts}, retryAttempts
		}

		// Store the error for potential final return
		lastErr = err

		// If this is the last attempt, break out of loop
		if attempt == maxAttempts {
			break
		}

		// For transient errors, sleep before retry
		backoff := CalculateBackoff(attempt, backoffMs, jitterMs)
		log.Printf("[DEBUG] Executor %q backing off for %v before retry attempt %d/%d",
			e.name, backoff, attempt+1, maxAttempts)

		// Check context again before sleeping
		select {
		case <-ctx.Done():
			return nil, ctx.Err(), retryAttempts
		case <-time.After(backoff):
			retryAttempts++
		}
	}

	// All attempts exhausted
	if lastErr == nil {
		lastErr = fmt.Errorf("step execution failed after %d attempts", maxAttempts)
	}

	return nil, &stepExecutionError{originalError: lastErr, retryAttempt: retryAttempts}, retryAttempts
}

// processMessageWithErrorTracking is the internal implementation that tracks which step failed.
// Returns (result, error, failedStepIndex).
// failedStepIndex is -1 if no error, or the 0-based index of the failed step if an error occurred.
// M0.2.2+: Implements retry logic for transient errors using executeStepWithRetry
func (e *Executor) processMessageWithErrorTracking(ctx context.Context, msg *Message) (*Message, error, int) {
	current := msg

	// Apply each step in sequence
	for i, step := range e.steps {
		// Execute step with retry logic for transient errors
		result, err, _ := e.executeStepWithRetry(ctx, step, current)
		if err != nil {
			// Preserve stepExecutionError type to maintain retry attempt information
			if see, ok := err.(*stepExecutionError); ok {
				see.stepIndex = i
				return nil, see, i
			}
			// For other error types, wrap with context
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
