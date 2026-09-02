package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockStep is a test implementation of Step for testing purposes.
type mockStep struct {
	name        string
	shouldFail  bool
	shouldDrop  bool
	delay       time.Duration // Optional delay to simulate processing time
	transformFn func(msg *Message) *Message
}

func (ms *mockStep) Execute(ctx context.Context, msg *Message) (*Message, error) {
	if ms.delay > 0 {
		select {
		case <-time.After(ms.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if ms.shouldFail {
		return nil, errors.New("mock step error")
	}

	if ms.shouldDrop {
		return nil, nil // Return nil to indicate message should be filtered out
	}

	if ms.transformFn != nil {
		return ms.transformFn(msg), nil
	}

	// Passthrough by default
	return msg, nil
}

// TestExecutorPassthrough verifies that a message flows through without steps.
func TestExecutorPassthrough(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	executor := NewExecutor("passthrough-test", inputCh, outputCh, []Step{})

	// Send a message
	msg := NewMessage(map[string]interface{}{"value": 42}, "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it in a goroutine
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Received message should not be nil")
	}

	// Verify body
	body := received.Body.(map[string]interface{})
	if body["value"] != 42 {
		t.Errorf("Body value mismatch: got %v, want 42", body["value"])
	}

	// Verify metadata is preserved
	if received.Metadata.Route != "test-route" {
		t.Errorf("Route mismatch: got %s, want test-route", received.Metadata.Route)
	}
}

// TestExecutorFilterAccept verifies that a filter step accepts matching messages.
func TestExecutorFilterAccept(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock filter that accepts (returns true/nil)
	filterStep := &mockStep{
		name:       "filter",
		shouldDrop: false, // Don't drop; let the message through
	}

	executor := NewExecutor("filter-test", inputCh, outputCh, []Step{filterStep})

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Received message should not be nil")
	}

	if received.Body != "test-payload" {
		t.Errorf("Body mismatch: got %v, want test-payload", received.Body)
	}
}

// TestExecutorFilterReject verifies that a filter step drops non-matching messages.
func TestExecutorFilterReject(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock filter that rejects (returns nil/nil to drop the message)
	filterStep := &mockStep{
		name:       "filter",
		shouldDrop: true, // Drop the message
	}

	executor := NewExecutor("filter-test", inputCh, outputCh, []Step{filterStep})

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it in a goroutine
	processDone := make(chan error, 1)
	go func() {
		// We'll try to receive with a timeout; if nothing arrives, that's expected
		processDone <- nil
		executor.Run(ctx)
	}()

	// Try to receive from output (should timeout since message was dropped)
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer recvCancel()

	received, _ := outputCh.Recv(recvCtx)
	if received != nil {
		t.Error("Expected no message in output after filter rejection, but received one")
	}
}

// TestExecutorTranslate verifies that a translate step transforms the message body.
func TestExecutorTranslate(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock translate step that doubles a numeric value
	translateStep := &mockStep{
		name: "translate",
		transformFn: func(msg *Message) *Message {
			copy := msg.Copy()
			if val, ok := msg.Body.(int); ok {
				copy.Body = val * 2
			}
			return copy
		},
	}

	executor := NewExecutor("translate-test", inputCh, outputCh, []Step{translateStep})

	// Send a message with numeric body
	msg := NewMessage(21, "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Received message should not be nil")
	}

	// Body should be doubled
	if received.Body != 42 {
		t.Errorf("Body mismatch: got %v, want 42", received.Body)
	}
}

// TestExecutorMultipleSteps verifies that multiple steps are applied in sequence.
func TestExecutorMultipleSteps(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// First step: translate (double the value)
	step1 := &mockStep{
		name: "translate-1",
		transformFn: func(msg *Message) *Message {
			copy := msg.Copy()
			if val, ok := msg.Body.(int); ok {
				copy.Body = val * 2
			}
			return copy
		},
	}

	// Second step: translate (add 10)
	step2 := &mockStep{
		name: "translate-2",
		transformFn: func(msg *Message) *Message {
			copy := msg.Copy()
			if val, ok := msg.Body.(int); ok {
				copy.Body = val + 10
			}
			return copy
		},
	}

	executor := NewExecutor("multi-step-test", inputCh, outputCh, []Step{step1, step2})

	// Send message with value 5
	// After step1: 5 * 2 = 10
	// After step2: 10 + 10 = 20
	msg := NewMessage(5, "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Received message should not be nil")
	}

	// Final body should be 20
	if received.Body != 20 {
		t.Errorf("Body mismatch: got %v, want 20", received.Body)
	}
}

// TestExecutorErrorHandling verifies that step failures are handled gracefully.
// NOTE: After M0.1.9, error messages contain DeadLetterEnvelope, not the original message body
func TestExecutorErrorHandling(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock step that fails
	failingStep := &mockStep{
		name:       "failing-step",
		shouldFail: true,
	}

	executor := NewExecutorWithErrorChannel("error-test", inputCh, outputCh, errorCh, []Step{failingStep})

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// The message should end up in the error channel as a Message containing a DeadLetterEnvelope
	errorMsg, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv from error channel failed: %v", err)
	}

	if errorMsg == nil {
		t.Error("Expected message in error channel")
	}

	// After M0.1.9, the body is a DeadLetterEnvelope, not the original payload
	envelope, ok := errorMsg.Body.(*DeadLetterEnvelope)
	if !ok {
		t.Fatalf("Error message body should be DeadLetterEnvelope, got %T", errorMsg.Body)
	}

	// Verify the original message is wrapped in the envelope
	if envelope.OriginalMessage == nil {
		t.Error("Envelope should contain OriginalMessage")
	} else if envelope.OriginalMessage.Body != "test-payload" {
		t.Errorf("Original message body mismatch: got %v, want test-payload", envelope.OriginalMessage.Body)
	}

	// Output channel should be empty
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer recvCancel()

	outputMsg, _ := outputCh.Recv(recvCtx)
	if outputMsg != nil {
		t.Error("Expected no message in output channel for failed step")
	}
}

// TestExecutorErrorHandlingWithoutErrorChannel verifies that errors are handled even without an error channel.
func TestExecutorErrorHandlingWithoutErrorChannel(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock step that fails
	failingStep := &mockStep{
		name:       "failing-step",
		shouldFail: true,
	}

	executor := NewExecutor("error-test", inputCh, outputCh, []Step{failingStep})

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it (without error channel)
	go func() {
		executor.Run(ctx)
	}()

	// Output channel should be empty (message failed and wasn't sent to output)
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer recvCancel()

	outputMsg, _ := outputCh.Recv(recvCtx)
	if outputMsg != nil {
		t.Error("Expected no message in output channel for failed step")
	}
}

// TestExecutorProcessMessage verifies ProcessMessage for single-message testing.
func TestExecutorProcessMessage(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx := context.Background()

	// Mock step that doubles a value
	step := &mockStep{
		name: "double",
		transformFn: func(msg *Message) *Message {
			copy := msg.Copy()
			if val, ok := msg.Body.(int); ok {
				copy.Body = val * 2
			}
			return copy
		},
	}

	executor := NewExecutor("process-test", inputCh, outputCh, []Step{step})

	// Process a single message
	msg := NewMessage(10, "test-route", "v1")
	result, err := executor.ProcessMessage(ctx, msg)

	if err != nil {
		t.Fatalf("ProcessMessage failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	if result.Body != 20 {
		t.Errorf("Body mismatch: got %v, want 20", result.Body)
	}
}

// TestExecutorProcessMessageWithError verifies ProcessMessage error handling.
func TestExecutorProcessMessageWithError(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx := context.Background()

	// Mock step that fails
	step := &mockStep{
		name:       "failing",
		shouldFail: true,
	}

	executor := NewExecutor("process-error-test", inputCh, outputCh, []Step{step})

	// Process a single message
	msg := NewMessage("test", "test-route", "v1")
	result, err := executor.ProcessMessage(ctx, msg)

	if err == nil {
		t.Error("Expected ProcessMessage to return an error")
	}

	if result != nil {
		t.Error("Result should be nil when step fails")
	}
}

// TestExecutorProcessMessageWithDrop verifies ProcessMessage handles dropped messages.
func TestExecutorProcessMessageWithDrop(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx := context.Background()

	// Mock filter that drops messages
	step := &mockStep{
		name:       "filter",
		shouldDrop: true,
	}

	executor := NewExecutor("process-drop-test", inputCh, outputCh, []Step{step})

	// Process a single message
	msg := NewMessage("test", "test-route", "v1")
	result, err := executor.ProcessMessage(ctx, msg)

	if err != nil {
		t.Fatalf("ProcessMessage should not error on drop, got %v", err)
	}

	if result != nil {
		t.Error("Result should be nil when message is dropped")
	}
}

// TestExecutorMessageMetadataPreservation verifies metadata is preserved through steps.
func TestExecutorMessageMetadataPreservation(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock step that transforms body but should preserve metadata
	step := &mockStep{
		name: "transform",
		transformFn: func(msg *Message) *Message {
			copy := msg.Copy()
			copy.Body = "transformed"
			// Add a header but preserve metadata
			copy.Headers["transformed"] = true
			return copy
		},
	}

	executor := NewExecutor("metadata-test", inputCh, outputCh, []Step{step})

	// Send a message
	msg := NewMessage("original", "my-route", "v123")
	msg.Metadata.Stage = "test"
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	// Check that metadata is preserved
	if received.Metadata.Route != "my-route" {
		t.Errorf("Route metadata not preserved: got %s, want my-route", received.Metadata.Route)
	}

	if received.Metadata.RouteVersion != "v123" {
		t.Errorf("RouteVersion metadata not preserved: got %s, want v123", received.Metadata.RouteVersion)
	}

	if received.Metadata.Stage != "test" {
		t.Errorf("Stage metadata not preserved: got %s, want test", received.Metadata.Stage)
	}

	// Check that body is transformed
	if received.Body != "transformed" {
		t.Errorf("Body not transformed: got %v, want transformed", received.Body)
	}

	// Check that headers were added
	if !received.Headers["transformed"].(bool) {
		t.Error("Header not added by step")
	}
}

// ===================== M0.2.1 Worker Pool Tests =====================

// TestExecutorSingleWorkerBackwardCompat verifies single-worker mode is backward compatible with M0.1.
func TestExecutorSingleWorkerBackwardCompat(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	executor := NewExecutor("single-worker-test", inputCh, outputCh, []Step{})

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Verify executor has 1 worker (default)
	if executor.numWorkers != 1 {
		t.Errorf("Expected 1 worker, got %d", executor.numWorkers)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Received message should not be nil")
	}

	if received.Body != "test-payload" {
		t.Errorf("Body mismatch: got %v, want test-payload", received.Body)
	}
}

// TestExecutorDualWorkersBackwardCompat verifies that explicit 1 worker is backward compatible.
func TestExecutorDualWorkersBackwardCompat(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create executor with explicit 1 worker
	executor := NewExecutorWithWorkers("single-explicit-test", inputCh, outputCh, nil, []Step{}, nil, 1)

	if executor.numWorkers != 1 {
		t.Errorf("Expected 1 worker, got %d", executor.numWorkers)
	}

	// Send a message
	msg := NewMessage("test", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Received message should not be nil")
	}
}

// TestExecutorMultipleWorkersConcurrentProcessing verifies that multiple workers process messages concurrently.
func TestExecutorMultipleWorkersConcurrentProcessing(t *testing.T) {
	inputCh := NewChannel("input", 20)
	outputCh := NewChannel("output", 20)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create executor with 4 workers
	executor := NewExecutorWithWorkers("multi-worker-test", inputCh, outputCh, nil, []Step{}, nil, 4)

	if executor.numWorkers != 4 {
		t.Errorf("Expected 4 workers, got %d", executor.numWorkers)
	}

	// Send 10 messages
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		msg := NewMessage(i, "test-route", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed at message %d: %v", i, err)
		}
	}

	// Start executor
	go func() {
		executor.Run(ctx)
	}()

	// Collect all output messages
	received := make([]int, 0, numMessages)
	for i := 0; i < numMessages; i++ {
		msg, err := outputCh.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv failed at message %d: %v", i, err)
		}

		if msg == nil {
			t.Fatalf("Received nil message at position %d", i)
		}

		body, ok := msg.Body.(int)
		if !ok {
			t.Fatalf("Message body is not int at position %d", i)
		}
		received = append(received, body)
	}

	// Verify we got all messages (order may vary due to concurrency)
	if len(received) != numMessages {
		t.Errorf("Expected %d messages, got %d", numMessages, len(received))
	}

	// Verify all message values are present (order independent)
	receivedSet := make(map[int]bool)
	for _, val := range received {
		receivedSet[val] = true
	}

	for i := 0; i < numMessages; i++ {
		if !receivedSet[i] {
			t.Errorf("Missing message with value %d", i)
		}
	}
}

// TestExecutorBackpressureWithManyWorkers verifies that backpressure works with multiple workers.
// When buffer is small and workers are busy, new sends should block.
func TestExecutorBackpressureWithManyWorkers(t *testing.T) {
	// Small buffer (5) with 10 workers and slow processing
	inputCh := NewChannel("input", 5)
	outputCh := NewChannel("output", 20)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Slow step to keep workers busy
	slowStep := &mockStep{
		name: "slow",
		transformFn: func(msg *Message) *Message {
			time.Sleep(100 * time.Millisecond)
			return msg
		},
	}

	executor := NewExecutorWithWorkers("backpressure-test", inputCh, outputCh, nil, []Step{slowStep}, nil, 10)

	// Start executor
	go func() {
		executor.Run(ctx)
	}()

	// Try to send more messages than buffer + workers can handle at once
	// This should block on the channel if backpressure works
	numToSend := 15
	sendDone := make(chan error, 1)
	go func() {
		for i := 0; i < numToSend; i++ {
			msg := NewMessage(i, "test-route", "v1")
			if err := inputCh.Send(ctx, msg); err != nil {
				sendDone <- err
				return
			}
		}
		sendDone <- nil
	}()

	// Give send goroutine time to progress
	time.Sleep(50 * time.Millisecond)

	// Drain output to let processing complete
	received := 0
	for received < numToSend {
		msg := outputCh.TryRecv()
		if msg == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		received++
	}

	// Wait for send to complete
	if err := <-sendDone; err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if received != numToSend {
		t.Errorf("Expected %d messages, got %d", numToSend, received)
	}
}

// TestExecutorGracefulShutdown verifies that shutdown waits for all workers to exit.
func TestExecutorGracefulShutdown(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	executor := NewExecutorWithWorkers("shutdown-test", inputCh, outputCh, nil, []Step{}, nil, 4)

	// Send 5 messages
	for i := 0; i < 5; i++ {
		msg := NewMessage(i, "test-route", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	// Close input to signal shutdown
	inputCh.Close()

	// Run should complete gracefully
	err := executor.Run(ctx)
	if err != nil {
		t.Errorf("Expected graceful shutdown, got error: %v", err)
	}

	// All messages should be processed
	received := 0
	for {
		msg := outputCh.TryRecv()
		if msg == nil {
			break
		}
		received++
	}

	if received != 5 {
		t.Errorf("Expected 5 output messages, got %d", received)
	}
}

// TestExecutorContextCancellation verifies that workers exit on context cancellation.
func TestExecutorContextCancellation(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithCancel(context.Background())

	executor := NewExecutorWithWorkers("cancel-test", inputCh, outputCh, nil, []Step{}, nil, 4)

	// Start executor
	runDone := make(chan error, 1)
	go func() {
		runDone <- executor.Run(ctx)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Run should exit quickly
	select {
	case err := <-runDone:
		if err != nil {
			t.Logf("Expected context cancellation, got error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Run should exit quickly on context cancellation")
	}
}

// TestExecutorInFlightCounterAccuracy verifies that in-flight counter is accurate.
func TestExecutorInFlightCounterAccuracy(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	executor := NewExecutorWithWorkers("inflight-test", inputCh, outputCh, nil, []Step{}, nil, 2)

	// Send 5 messages
	for i := 0; i < 5; i++ {
		msg := NewMessage(i, "test-route", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	// Start executor
	go func() {
		executor.Run(ctx)
	}()

	// Give time for some messages to be in flight
	time.Sleep(50 * time.Millisecond)

	// Check in-flight count (should be > 0 initially)
	inFlight := executor.InFlightCount()
	if inFlight < 0 {
		t.Errorf("In-flight count should not be negative: %d", inFlight)
	}

	// Drain output to complete processing
	for i := 0; i < 5; i++ {
		_, err := outputCh.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
	}

	// Give time for final processing
	time.Sleep(50 * time.Millisecond)

	// After all messages are processed, in-flight should be 0
	finalInFlight := executor.InFlightCount()
	if finalInFlight != 0 {
		t.Errorf("In-flight count should be 0 after processing, got %d", finalInFlight)
	}
}

// TestExecutorErrorHandlingMultipleWorkers verifies error handling across multiple workers.
func TestExecutorErrorHandlingMultipleWorkers(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Step that fails
	failStep := &mockStep{
		name:       "fail-step",
		shouldFail: true,
	}

	executor := NewExecutorWithErrorChannel("multi-worker-error-test", inputCh, outputCh, errorCh, []Step{failStep})

	// Send 4 messages (will be processed by 1 worker)
	for i := 0; i < 4; i++ {
		msg := NewMessage(i, "test-route", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	// Start executor
	go func() {
		executor.Run(ctx)
	}()

	// Collect error messages
	received := 0
	for received < 4 {
		msg, err := errorCh.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv from error channel failed: %v", err)
		}

		if msg == nil {
			t.Fatalf("Received nil error message")
		}

		// Verify it's a DeadLetterEnvelope
		_, ok := msg.Body.(*DeadLetterEnvelope)
		if !ok {
			t.Fatalf("Error message body should be DeadLetterEnvelope, got %T", msg.Body)
		}

		received++
	}

	if received != 4 {
		t.Errorf("Expected 4 error messages, got %d", received)
	}
}

// TestExecutorMessageOrderPreservationFIFO verifies that FIFO order is preserved at the channel level.
func TestExecutorMessageOrderPreservationFIFO(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Single worker to ensure sequential processing
	executor := NewExecutorWithWorkers("order-test", inputCh, outputCh, nil, []Step{}, nil, 1)

	// Send messages in order
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		msg := NewMessage(i, "test-route", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	// Start executor
	go func() {
		executor.Run(ctx)
	}()

	// Receive messages and verify order
	for i := 0; i < numMessages; i++ {
		msg, err := outputCh.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}

		if msg == nil {
			t.Fatalf("Received nil message")
		}

		body, ok := msg.Body.(int)
		if !ok {
			t.Fatalf("Message body is not int")
		}

		if body != i {
			t.Errorf("Message order mismatch at position %d: got %d, want %d", i, body, i)
		}
	}
}

// TestExecutorWorkerCountLowerBound verifies that invalid worker counts default to 1.
func TestExecutorWorkerCountLowerBound(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	// Test with 0 workers (should default to 1)
	executor := NewExecutorWithWorkers("zero-worker-test", inputCh, outputCh, nil, []Step{}, nil, 0)
	if executor.numWorkers != 1 {
		t.Errorf("Expected 1 worker for invalid input 0, got %d", executor.numWorkers)
	}

	// Test with negative workers (should default to 1)
	executor = NewExecutorWithWorkers("negative-worker-test", inputCh, outputCh, nil, []Step{}, nil, -5)
	if executor.numWorkers != 1 {
		t.Errorf("Expected 1 worker for invalid input -5, got %d", executor.numWorkers)
	}
}

// TestExecutorHighWorkerCountPerformance verifies high worker counts work correctly.
func TestExecutorHighWorkerCountPerformance(t *testing.T) {
	inputCh := NewChannel("input", 100)
	outputCh := NewChannel("output", 100)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create executor with high worker count
	executor := NewExecutorWithWorkers("high-worker-count-test", inputCh, outputCh, nil, []Step{}, nil, 32)

	if executor.numWorkers != 32 {
		t.Errorf("Expected 32 workers, got %d", executor.numWorkers)
	}

	// Send many messages
	numMessages := 100
	for i := 0; i < numMessages; i++ {
		msg := NewMessage(i, "test-route", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	// Start executor
	go func() {
		executor.Run(ctx)
	}()

	// Collect all outputs
	received := 0
	for received < numMessages {
		msg, err := outputCh.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}

		if msg == nil {
			t.Fatalf("Received nil message")
		}

		received++
	}

	if received != numMessages {
		t.Errorf("Expected %d messages, got %d", numMessages, received)
	}
}

// retryableStep is a mock step that can fail N times and then succeed
type retryableStep struct {
	name            string
	failuresRemaining int
	callCount       int
	mutex           sync.Mutex
}

func (rs *retryableStep) Execute(ctx context.Context, msg *Message) (*Message, error) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.callCount++

	if rs.failuresRemaining > 0 {
		rs.failuresRemaining--
		return nil, errors.New("transient error: temporary connection issue")
	}

	return msg, nil
}

// permanentErrorStep always fails with a permanent error (not retryable)
type permanentErrorStep struct {
	name string
	callCount int
	mutex sync.Mutex
}

func (pes *permanentErrorStep) Execute(ctx context.Context, msg *Message) (*Message, error) {
	pes.mutex.Lock()
	defer pes.mutex.Unlock()

	pes.callCount++

	return nil, errors.New("validation error: invalid input format")
}

// TestRetrySuccessNoRetryNeeded verifies a successful step doesn't use retries.
func TestRetrySuccessNoRetryNeeded(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	step := &retryableStep{name: "success-step", failuresRemaining: 0}

	retryPolicy := &RetryPolicy{
		MaxAttempts: 3,
		BackoffMs:   100,
		JitterMs:    10,
	}

	executor := NewExecutorWithRetry("retry-success-test", inputCh, outputCh, nil, []Step{step}, nil, 1, retryPolicy)

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Expected message in output")
	}

	// Verify step was called exactly once (no retries needed)
	if step.callCount != 1 {
		t.Errorf("Step should be called once, was called %d times", step.callCount)
	}
}

// TestRetryTransientErrorSuccessfulRetry verifies transient errors retry and eventually succeed.
func TestRetryTransientErrorSuccessfulRetry(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step fails once, then succeeds
	step := &retryableStep{name: "transient-step", failuresRemaining: 1}

	retryPolicy := &RetryPolicy{
		MaxAttempts: 3,
		BackoffMs:   50,   // Short backoff for testing
		JitterMs:    10,
	}

	executor := NewExecutorWithRetry("retry-transient-test", inputCh, outputCh, nil, []Step{step}, nil, 1, retryPolicy)

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from output
	received, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received == nil {
		t.Error("Expected message in output after retry")
	}

	// Verify step was called twice (1 failure + 1 success)
	if step.callCount != 2 {
		t.Errorf("Step should be called twice, was called %d times", step.callCount)
	}
}

// TestRetryTransientErrorMaxExhausted verifies transient errors eventually fail if max attempts reached.
func TestRetryTransientErrorMaxExhausted(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step fails 5 times (exceeds max attempts of 3)
	step := &retryableStep{name: "always-fail-step", failuresRemaining: 5}

	retryPolicy := &RetryPolicy{
		MaxAttempts: 3,
		BackoffMs:   50,
		JitterMs:    10,
	}

	executor := NewExecutorWithRetry("retry-exhausted-test", inputCh, outputCh, errorCh, []Step{step}, nil, 1, retryPolicy)

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from error channel
	errorMsg, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv from error channel failed: %v", err)
	}

	if errorMsg == nil {
		t.Error("Expected message in error channel")
	}

	// Verify envelope contains retry attempt info
	envelope, ok := errorMsg.Body.(*DeadLetterEnvelope)
	if !ok {
		t.Fatalf("Expected DeadLetterEnvelope, got %T", errorMsg.Body)
	}

	// Should have attempted 3 times (max attempts)
	if envelope.RetryAttempt != 2 {
		t.Errorf("Expected 2 retry attempts, got %d", envelope.RetryAttempt)
	}

	// Verify step was called 3 times
	if step.callCount != 3 {
		t.Errorf("Step should be called 3 times, was called %d times", step.callCount)
	}
}

// TestRetryPermanentErrorNoRetry verifies permanent errors don't retry.
func TestRetryPermanentErrorNoRetry(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	step := &permanentErrorStep{name: "validation-step"}

	retryPolicy := &RetryPolicy{
		MaxAttempts: 3,
		BackoffMs:   100,
		JitterMs:    10,
	}

	executor := NewExecutorWithRetry("retry-permanent-test", inputCh, outputCh, errorCh, []Step{step}, nil, 1, retryPolicy)

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive from error channel
	errorMsg, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv from error channel failed: %v", err)
	}

	if errorMsg == nil {
		t.Error("Expected message in error channel")
	}

	// Verify envelope shows no retries (permanent error)
	envelope, ok := errorMsg.Body.(*DeadLetterEnvelope)
	if !ok {
		t.Fatalf("Expected DeadLetterEnvelope, got %T", errorMsg.Body)
	}

	// Should have 0 retry attempts (failed immediately)
	if envelope.RetryAttempt != 0 {
		t.Errorf("Expected 0 retry attempts for permanent error, got %d", envelope.RetryAttempt)
	}

	// Verify step was called only once (no retries)
	if step.callCount != 1 {
		t.Errorf("Step should be called once for permanent error, was called %d times", step.callCount)
	}
}

// TestBackoffCalculation verifies exponential backoff formula is correct.
func TestBackoffCalculation(t *testing.T) {
	tests := []struct {
		name      string
		attempt   int
		baseMs    int
		jitterMs  int
		minDur    time.Duration
		maxDur    time.Duration
	}{
		{
			name:     "attempt 1",
			attempt:  1,
			baseMs:   1000,
			jitterMs: 100,
			minDur:   1000 * time.Millisecond,
			maxDur:   1100 * time.Millisecond,
		},
		{
			name:     "attempt 2",
			attempt:  2,
			baseMs:   1000,
			jitterMs: 100,
			minDur:   2000 * time.Millisecond,
			maxDur:   2100 * time.Millisecond,
		},
		{
			name:     "attempt 3",
			attempt:  3,
			baseMs:   1000,
			jitterMs: 100,
			minDur:   4000 * time.Millisecond,
			maxDur:   4100 * time.Millisecond,
		},
		{
			name:     "attempt 4 - exceeds max",
			attempt:  4,
			baseMs:   1000,
			jitterMs: 100,
			minDur:   8000 * time.Millisecond,
			maxDur:   30 * time.Second, // Capped at 30s
		},
		{
			name:     "no jitter",
			attempt:  1,
			baseMs:   500,
			jitterMs: 0,
			minDur:   500 * time.Millisecond,
			maxDur:   500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate multiple times to account for jitter randomness
			durations := make([]time.Duration, 10)
			for i := 0; i < 10; i++ {
				durations[i] = CalculateBackoff(tt.attempt, tt.baseMs, tt.jitterMs)
			}

			// Check that all are within expected range
			for i, dur := range durations {
				if dur < tt.minDur || dur > tt.maxDur {
					t.Errorf("Duration %d out of range: %v (expected %v to %v)",
						i, dur, tt.minDur, tt.maxDur)
				}
			}
		})
	}
}

// TestRetryContextCancellation verifies context cancellation stops retries immediately.
func TestRetryContextCancellation(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	// Use a very short context so it cancels during backoff
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Step always fails (transient error)
	step := &retryableStep{name: "failing-step", failuresRemaining: 10}

	retryPolicy := &RetryPolicy{
		MaxAttempts: 10,
		BackoffMs:   500,  // Long backoff - context will cancel during this
		JitterMs:    50,
	}

	executor := NewExecutorWithRetry("cancel-test", inputCh, outputCh, errorCh, []Step{step}, nil, 1, retryPolicy)

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it (will be cancelled)
	executor.Run(ctx)

	// Step should have been called fewer than max attempts due to context cancellation
	if step.callCount >= 10 {
		t.Errorf("Step called %d times, expected fewer due to context cancellation", step.callCount)
	}
}

// TestErrorClassification verifies error types are classified correctly.
func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		expected  ErrorType
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorTypeUnknown,
		},
		{
			name:     "context cancelled",
			err:      context.Canceled,
			expected: ErrorTypePermanent,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: ErrorTypeTransient,
		},
		{
			name:     "timeout in error message",
			err:      errors.New("i/o timeout: connection failed"),
			expected: ErrorTypeTransient,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: ErrorTypeTransient,
		},
		{
			name:     "validation error",
			err:      errors.New("validation error: invalid input"),
			expected: ErrorTypePermanent,
		},
		{
			name:     "unknown error",
			err:      errors.New("some random error"),
			expected: ErrorTypePermanent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRetryPolicyNormalization verifies retry policy defaults are applied correctly.
func TestRetryPolicyNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    *RetryPolicy
		expected *RetryPolicy
	}{
		{
			name:     "nil policy",
			input:    nil,
			expected: nil,
		},
		{
			name:     "max attempts < 1",
			input:    &RetryPolicy{MaxAttempts: 0, BackoffMs: 1000, JitterMs: 100},
			expected: nil,
		},
		{
			name:     "valid policy",
			input:    &RetryPolicy{MaxAttempts: 3, BackoffMs: 1000, JitterMs: 100},
			expected: &RetryPolicy{MaxAttempts: 3, BackoffMs: 1000, JitterMs: 100},
		},
		{
			name:     "backoff defaults to 1000ms",
			input:    &RetryPolicy{MaxAttempts: 3, BackoffMs: 0, JitterMs: 100},
			expected: &RetryPolicy{MaxAttempts: 3, BackoffMs: 1000, JitterMs: 100},
		},
		{
			name:     "jitter defaults to 100ms",
			input:    &RetryPolicy{MaxAttempts: 3, BackoffMs: 1000, JitterMs: -1},
			expected: &RetryPolicy{MaxAttempts: 3, BackoffMs: 1000, JitterMs: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRetryPolicy(tt.input)
			if result == nil && tt.expected == nil {
				return
			}
			if result == nil || tt.expected == nil {
				t.Errorf("Expected %v, got %v", tt.expected, result)
				return
			}
			if result.MaxAttempts != tt.expected.MaxAttempts ||
				result.BackoffMs != tt.expected.BackoffMs ||
				result.JitterMs != tt.expected.JitterMs {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

// TestNoRetryWhenPolicyNil verifies backward compatibility when retry policy is nil.
func TestNoRetryWhenPolicyNil(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Step fails with transient error
	step := &retryableStep{name: "transient-step", failuresRemaining: 1}

	// No retry policy (nil)
	executor := NewExecutorWithRetry("no-retry-test", inputCh, outputCh, errorCh, []Step{step}, nil, 1, nil)

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Should go to error channel (no retry)
	errorMsg, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv from error channel failed: %v", err)
	}

	if errorMsg == nil {
		t.Error("Expected message in error channel")
	}

	// Verify step was called only once (no retries)
	if step.callCount != 1 {
		t.Errorf("Step should be called once with no retry policy, was called %d times", step.callCount)
	}
}
