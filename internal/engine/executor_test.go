package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockStep is a test implementation of Step for testing purposes.
type mockStep struct {
	name        string
	shouldFail  bool
	shouldDrop  bool
	transformFn func(msg *Message) *Message
}

func (ms *mockStep) Execute(ctx context.Context, msg *Message) (*Message, error) {
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

	// The message should end up in the error channel, not the output channel
	errorMsg, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv from error channel failed: %v", err)
	}

	if errorMsg == nil {
		t.Error("Expected message in error channel")
	}

	if errorMsg.Body != "test-payload" {
		t.Errorf("Error message body mismatch: got %v, want test-payload", errorMsg.Body)
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
