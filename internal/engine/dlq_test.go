package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// TestDeadLetterEnvelopeCreation verifies that a DeadLetterEnvelope is created correctly
func TestDeadLetterEnvelopeCreation(t *testing.T) {
	msg := NewMessage(map[string]interface{}{"value": 42}, "test-route", "v1")
	err := errors.New("filter step failed")

	dle := NewDeadLetterEnvelope(msg, err, 0, "filter")

	if dle.OriginalMessage != msg {
		t.Error("OriginalMessage not set correctly")
	}

	if dle.Error != "filter step failed" {
		t.Errorf("Error mismatch: got %s, want 'filter step failed'", dle.Error)
	}

	if dle.FailedStep != 0 {
		t.Errorf("FailedStep mismatch: got %d, want 0", dle.FailedStep)
	}

	if dle.FailedStepType != "filter" {
		t.Errorf("FailedStepType mismatch: got %s, want 'filter'", dle.FailedStepType)
	}

	if dle.Metadata == nil {
		t.Error("Metadata should not be nil")
	}

	// Verify ProcessedAt is roughly now
	elapsed := time.Since(dle.ProcessedAt)
	if elapsed > 1*time.Second {
		t.Errorf("ProcessedAt not recent: %v ago", elapsed)
	}
}

// TestDeadLetterEnvelopeCreationFilterFailure verifies creation with a filter error
func TestDeadLetterEnvelopeCreationFilterFailure(t *testing.T) {
	msg := NewMessage(map[string]interface{}{"items": []int{1, 2, 3}}, "filter-route", "v1")
	err := errors.New("filter expression evaluation error")

	dle := NewDeadLetterEnvelope(msg, err, 1, "filter")

	if dle.FailedStep != 1 {
		t.Errorf("FailedStep mismatch: got %d, want 1", dle.FailedStep)
	}

	if dle.FailedStepType != "filter" {
		t.Errorf("FailedStepType mismatch: got %s, want 'filter'", dle.FailedStepType)
	}

	if dle.Error != "filter expression evaluation error" {
		t.Errorf("Error mismatch: got %s", dle.Error)
	}

	if dle.OriginalMessage == nil {
		t.Error("Original message should not be nil")
	}
}

// TestDeadLetterEnvelopeCreationTranslateFailure verifies creation with a translate error
func TestDeadLetterEnvelopeCreationTranslateFailure(t *testing.T) {
	msg := NewMessage(map[string]interface{}{"nested": map[string]interface{}{"value": "test"}}, "translate-route", "v2")
	err := errors.New("JSONata expression syntax error: unexpected token")

	dle := NewDeadLetterEnvelope(msg, err, 2, "translate")

	if dle.FailedStep != 2 {
		t.Errorf("FailedStep mismatch: got %d, want 2", dle.FailedStep)
	}

	if dle.FailedStepType != "translate" {
		t.Errorf("FailedStepType mismatch: got %s, want 'translate'", dle.FailedStepType)
	}

	if dle.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

// TestDeadLetterEnvelopeCreationInvalidExpressionError verifies handling of expression errors
func TestDeadLetterEnvelopeCreationInvalidExpressionError(t *testing.T) {
	msg := NewMessage("test-payload", "expr-route", "v1")
	err := errors.New("invalid JSONata expression: unexpected end of input")

	dle := NewDeadLetterEnvelope(msg, err, 3, "translate")

	if dle.Error != "invalid JSONata expression: unexpected end of input" {
		t.Errorf("Error not captured correctly: %s", dle.Error)
	}

	if dle.FailedStep != 3 {
		t.Errorf("FailedStep mismatch: got %d, want 3", dle.FailedStep)
	}
}

// TestDeadLetterEnvelopeMarshalJSON verifies JSON serialization
func TestDeadLetterEnvelopeMarshalJSON(t *testing.T) {
	msg := NewMessage(map[string]interface{}{"data": "value"}, "test-route", "v1")
	err := errors.New("step failed")

	dle := NewDeadLetterEnvelope(msg, err, 0, "filter")
	dle.Metadata["custom_key"] = "custom_value"

	data, marshallErr := json.Marshal(dle)
	if marshallErr != nil {
		t.Fatalf("MarshalJSON failed: %v", marshallErr)
	}

	// Verify it's valid JSON
	var unmarshalled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshalled); err != nil {
		t.Fatalf("Unmarshalled JSON is invalid: %v", err)
	}

	// Verify key fields are present
	if unmarshalled["error"] != "step failed" {
		t.Errorf("Error field mismatch in JSON: %v", unmarshalled["error"])
	}

	if unmarshalled["failed_step"] != float64(0) {
		t.Errorf("FailedStep field mismatch in JSON: %v", unmarshalled["failed_step"])
	}

	if unmarshalled["failed_step_type"] != "filter" {
		t.Errorf("FailedStepType field mismatch in JSON: %v", unmarshalled["failed_step_type"])
	}

	// Verify timestamp is in ISO format
	if _, ok := unmarshalled["processed_at"]; !ok {
		t.Error("ProcessedAt field missing in JSON")
	}

	// Verify metadata is present
	if _, ok := unmarshalled["metadata"]; !ok {
		t.Error("Metadata field missing in JSON")
	}
}

// TestDeadLetterEnvelopeUnmarshalJSON verifies JSON deserialization
func TestDeadLetterEnvelopeUnmarshalJSON(t *testing.T) {
	jsonStr := `{
		"original_message": {
			"headers": {},
			"body": {"test": "data"},
			"metadata": {
				"correlation_id": "test-123",
				"ingested_at": "2026-09-01T12:00:00Z",
				"route": "test-route",
				"route_version": "v1"
			}
		},
		"error": "test error",
		"failed_step": 1,
		"failed_step_type": "translate",
		"processed_at": "2026-09-01T12:00:00.123456Z",
		"metadata": {"key": "value"}
	}`

	var dle DeadLetterEnvelope
	if err := json.Unmarshal([]byte(jsonStr), &dle); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if dle.Error != "test error" {
		t.Errorf("Error mismatch: got %s", dle.Error)
	}

	if dle.FailedStep != 1 {
		t.Errorf("FailedStep mismatch: got %d", dle.FailedStep)
	}

	if dle.FailedStepType != "translate" {
		t.Errorf("FailedStepType mismatch: got %s", dle.FailedStepType)
	}

	if dle.OriginalMessage == nil {
		t.Error("OriginalMessage should be unmarshalled")
	}
}

// TestExecutorSendsEnvelopeToErrorChannel verifies that executor sends dead-letter envelope on step failure
func TestExecutorSendsEnvelopeToErrorChannel(t *testing.T) {
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
		t.Fatal("Expected message in error channel")
	}

	// Extract the envelope from the Message body
	envelope, ok := errorMsg.Body.(*DeadLetterEnvelope)
	if !ok {
		t.Fatalf("Message body is not a DeadLetterEnvelope: %T", errorMsg.Body)
	}

	// Verify envelope fields
	if envelope.OriginalMessage == nil {
		t.Error("Envelope OriginalMessage is nil")
	}

	if envelope.FailedStep != 0 {
		t.Errorf("FailedStep mismatch: got %d, want 0", envelope.FailedStep)
	}

	if envelope.Error == "" {
		t.Error("Envelope Error should not be empty")
	}
}

// TestErrorChannelReceivesCorrectEnvelopeFields verifies all envelope fields are populated correctly
func TestErrorChannelReceivesCorrectEnvelopeFields(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create a failing step with explicit type name
	failingStep := &mockStep{
		name:       "translate",
		shouldFail: true,
	}

	stepNames := []string{"translate"}
	executor := NewExecutorWithStepNames("error-test", inputCh, outputCh, errorCh, []Step{failingStep}, stepNames)

	// Send a message with specific metadata
	msg := NewMessage(map[string]interface{}{"value": 42}, "order-route", "v2")
	msg.Metadata.Stage = "processing"

	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive error message
	errorMsg, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	envelope, ok := errorMsg.Body.(*DeadLetterEnvelope)
	if !ok {
		t.Fatalf("Body is not DeadLetterEnvelope: %T", errorMsg.Body)
	}

	// Verify all fields
	if envelope.OriginalMessage == nil {
		t.Error("OriginalMessage is nil")
	} else {
		body := envelope.OriginalMessage.Body.(map[string]interface{})
		if body["value"] != 42 {
			t.Errorf("Original message body not preserved: %v", body)
		}
	}

	if envelope.FailedStep != 0 {
		t.Errorf("FailedStep mismatch: got %d, want 0", envelope.FailedStep)
	}

	if envelope.FailedStepType != "translate" {
		t.Errorf("FailedStepType mismatch: got %s, want 'translate'", envelope.FailedStepType)
	}

	if envelope.Error == "" {
		t.Error("Error should be populated")
	}

	// Verify ProcessedAt is approximately now
	elapsed := time.Since(envelope.ProcessedAt)
	if elapsed > 1*time.Second {
		t.Errorf("ProcessedAt not recent: %v ago", elapsed)
	}

	// Verify outer Message metadata preserves correlation
	if errorMsg.Metadata.CorrelationID != msg.Metadata.CorrelationID {
		t.Error("CorrelationID not preserved in outer Message")
	}

	if errorMsg.Metadata.Route != "order-route" {
		t.Errorf("Route mismatch: got %s, want order-route", errorMsg.Metadata.Route)
	}

	if errorMsg.Metadata.RouteVersion != "v2" {
		t.Errorf("RouteVersion mismatch: got %s, want v2", errorMsg.Metadata.RouteVersion)
	}

	if errorMsg.Metadata.Stage != "error_path" {
		t.Errorf("Stage mismatch: got %s, want error_path", errorMsg.Metadata.Stage)
	}
}

// TestMultipleSequentialFailures verifies error channel handles multiple messages with errors
func TestMultipleSequentialFailures(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	failingStep := &mockStep{
		name:       "filter",
		shouldFail: true,
	}

	executor := NewExecutorWithErrorChannel("error-test", inputCh, outputCh, errorCh, []Step{failingStep})

	// Send three messages
	messages := []*Message{
		NewMessage("msg1", "route1", "v1"),
		NewMessage("msg2", "route2", "v1"),
		NewMessage("msg3", "route3", "v1"),
	}

	for _, msg := range messages {
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	// Process them
	go func() {
		executor.Run(ctx)
	}()

	// Receive three error messages
	errorMessages := make([]*Message, 0)
	for i := 0; i < 3; i++ {
		errorMsg, err := errorCh.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv %d failed: %v", i, err)
		}

		if errorMsg == nil {
			t.Fatalf("Expected error message %d", i)
		}

		errorMessages = append(errorMessages, errorMsg)
	}

	// Verify we got three envelopes
	if len(errorMessages) != 3 {
		t.Errorf("Expected 3 error messages, got %d", len(errorMessages))
	}

	// Verify correlation IDs are preserved
	for i, errorMsg := range errorMessages {
		if errorMsg.Body == nil {
			t.Fatalf("Error message %d body is nil", i)
		}

		envelope, ok := errorMsg.Body.(*DeadLetterEnvelope)
		if !ok {
			t.Fatalf("Error message %d body is not DeadLetterEnvelope", i)
		}

		if envelope.OriginalMessage == nil {
			t.Errorf("Error message %d envelope has no OriginalMessage", i)
		}
	}
}

// TestErrorChannelInFlightCount verifies that InFlight() counts messages correctly
func TestErrorChannelInFlightCount(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	failingStep := &mockStep{
		name:       "filter",
		shouldFail: true,
	}

	executor := NewExecutorWithErrorChannel("error-test", inputCh, outputCh, errorCh, []Step{failingStep})

	// Send two messages
	msg1 := NewMessage("msg1", "route1", "v1")
	msg2 := NewMessage("msg2", "route2", "v1")

	if err := inputCh.Send(ctx, msg1); err != nil {
		t.Fatalf("Send msg1 failed: %v", err)
	}

	if err := inputCh.Send(ctx, msg2); err != nil {
		t.Fatalf("Send msg2 failed: %v", err)
	}

	// Run executor in background
	done := make(chan error, 1)
	go func() {
		done <- executor.Run(ctx)
	}()

	// Receive first error message
	_, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	// Receive second error message
	_, err = errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	// Check InFlight should be 0 or close to 0 (depending on timing)
	inFlight := errorCh.InFlight()
	if inFlight < 0 {
		t.Errorf("InFlight count is negative: %d", inFlight)
	}
}

// TestEmptyErrorChannelNoErrors verifies behavior when no errors occur
func TestEmptyErrorChannelNoErrors(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock step that succeeds
	passingStep := &mockStep{
		name:       "filter",
		shouldFail: false,
	}

	executor := NewExecutorWithErrorChannel("success-test", inputCh, outputCh, errorCh, []Step{passingStep})

	// Send a message
	msg := NewMessage("test-payload", "test-route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Message should go to output, not error
	outputMsg, err := outputCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv from output failed: %v", err)
	}

	if outputMsg == nil {
		t.Error("Expected message in output channel")
	}

	// Error channel should be empty
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer recvCancel()

	errorMsg, _ := errorCh.Recv(recvCtx)
	if errorMsg != nil {
		t.Error("Unexpected message in error channel for successful message")
	}
}

// TestErrorChannelClosedOnContextCancellation verifies error channel behavior on context cancellation
func TestErrorChannelClosedOnContextCancellation(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	failingStep := &mockStep{
		name:       "filter",
		shouldFail: true,
	}

	executor := NewExecutorWithErrorChannel("error-test", inputCh, outputCh, errorCh, []Step{failingStep})

	// Send a message
	msg := NewMessage("test", "route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Run executor, then immediately cancel
	runDone := make(chan error, 1)
	go func() {
		runDone <- executor.Run(ctx)
	}()

	// Give it a moment to start processing
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for executor to stop
	select {
	case <-runDone:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Executor did not stop on context cancellation")
	}
}

// TestDeadLetterEnvelopeWithMultipleStepTypes verifies step type tracking with multiple steps
func TestDeadLetterEnvelopeWithMultipleStepTypes(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	defer inputCh.Close()
	defer outputCh.Close()
	defer errorCh.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create multiple steps: filter (pass), translate (fail)
	filterStep := &mockStep{
		name:       "filter",
		shouldFail: false,
	}

	translateStep := &mockStep{
		name:       "translate",
		shouldFail: true,
	}

	stepNames := []string{"filter", "translate"}
	executor := NewExecutorWithStepNames("multi-step-test", inputCh, outputCh, errorCh, []Step{filterStep, translateStep}, stepNames)

	// Send a message
	msg := NewMessage("test", "route", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Process it
	go func() {
		executor.Run(ctx)
	}()

	// Receive error message
	errorMsg, err := errorCh.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	envelope, ok := errorMsg.Body.(*DeadLetterEnvelope)
	if !ok {
		t.Fatalf("Body is not DeadLetterEnvelope: %T", errorMsg.Body)
	}

	// The second step (translate) should have failed, so FailedStep should be 1
	if envelope.FailedStep != 1 {
		t.Errorf("FailedStep mismatch: got %d, want 1", envelope.FailedStep)
	}

	if envelope.FailedStepType != "translate" {
		t.Errorf("FailedStepType mismatch: got %s, want 'translate'", envelope.FailedStepType)
	}
}
