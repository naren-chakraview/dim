package steps

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestNewWiretapStep verifies that a valid wiretap step can be created
func TestNewWiretapStep(t *testing.T) {
	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}
	if step == nil {
		t.Error("NewWiretapStep returned nil step")
	}
	if step.sinkName != "audit_log" {
		t.Errorf("Expected sink name 'audit_log', got %q", step.sinkName)
	}
}

// TestNewWiretapStepEmptySinkName verifies that empty sink name is rejected
func TestNewWiretapStepEmptySinkName(t *testing.T) {
	step, err := NewWiretapStep("")
	if err == nil {
		t.Error("Expected error for empty sink name, got nil")
	}
	if step != nil {
		t.Error("Expected step to be nil when sink name is empty")
	}
}

// TestWiretapMessageCopiedToTapSink verifies that message is copied to tap sink
func TestWiretapMessageCopiedToTapSink(t *testing.T) {
	ctx := context.Background()

	// Create wiretap step and tap sink channel
	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	tapSink := engine.NewChannel("tap_sink", 10)
	step.SetTapSink(tapSink)

	// Create a test message
	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "order-1",
			"amount": 100.0,
		},
		"test-route",
		"v1",
	)
	msg.Headers["test"] = "value"

	// Execute the wiretap step
	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify original message is returned unchanged
	if result == nil {
		t.Fatal("Result message is nil")
	}
	if result.Body.(map[string]interface{})["id"] != "order-1" {
		t.Error("Original message body was modified")
	}

	// Give the goroutine a moment to send the copy
	time.Sleep(100 * time.Millisecond)

	// Verify the copy was sent to tap sink
	// Use a short timeout to avoid blocking
	tapCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	tapMsg, err := tapSink.Recv(tapCtx)
	if err != nil {
		t.Fatalf("Failed to receive from tap sink: %v", err)
	}

	if tapMsg == nil {
		t.Fatal("No message received from tap sink")
	}

	// Verify the tapped message has the same body
	tappedBody := tapMsg.Body.(map[string]interface{})
	if tappedBody["id"] != "order-1" {
		t.Errorf("Tapped message body incorrect: expected id 'order-1', got %v", tappedBody["id"])
	}
	if tappedBody["amount"] != 100.0 {
		t.Errorf("Tapped message body incorrect: expected amount 100.0, got %v", tappedBody["amount"])
	}
}

// TestWiretapOriginalContinuesDownstream verifies original message continues to downstream
func TestWiretapOriginalContinuesDownstream(t *testing.T) {
	ctx := context.Background()

	// Create wiretap step and tap sink channel
	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	tapSink := engine.NewChannel("tap_sink", 10)
	step.SetTapSink(tapSink)

	// Create a test message
	msg := engine.NewMessage(
		map[string]interface{}{
			"transaction_id": "tx-123",
			"status":         "pending",
		},
		"payment-route",
		"v1",
	)

	// Execute the wiretap step
	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify original message is returned unchanged
	if result == nil {
		t.Fatal("Result message is nil")
	}

	// Verify the exact same message object is returned
	if result.Body.(map[string]interface{})["transaction_id"] != "tx-123" {
		t.Error("Original message body was modified")
	}

	// Verify metadata is preserved
	if result.Metadata.Route != "payment-route" {
		t.Errorf("Route metadata changed: expected 'payment-route', got %q", result.Metadata.Route)
	}

	if result.Metadata.CorrelationID != msg.Metadata.CorrelationID {
		t.Error("CorrelationID was modified")
	}
}

// TestWiretapDeepCopy verifies that the copy made for tap is independent of original
func TestWiretapDeepCopy(t *testing.T) {
	ctx := context.Background()

	// Create wiretap step and tap sink channel
	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	tapSink := engine.NewChannel("tap_sink", 10)
	step.SetTapSink(tapSink)

	// Create a test message with headers
	msg := engine.NewMessage(
		map[string]interface{}{
			"value": 42,
		},
		"test-route",
		"v1",
	)
	msg.Headers["X-User"] = "user-1"
	originalUserValue := msg.Headers["X-User"]

	// Execute the wiretap step
	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// Give the goroutine a moment to send the copy
	time.Sleep(100 * time.Millisecond)

	// Receive the tapped message
	tapCtx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	tapMsg, err := tapSink.Recv(tapCtx2)
	if err != nil {
		t.Fatalf("Failed to receive from tap sink: %v", err)
	}

	if tapMsg == nil {
		t.Fatal("No message received from tap sink")
	}

	// Verify the tapped message has the same header value
	if tapMsg.Headers["X-User"] != originalUserValue {
		t.Errorf("Tapped message headers incorrect: expected 'user-1', got %q", tapMsg.Headers["X-User"])
	}

	// Modify the tapped message headers
	tapMsg.Headers["X-User"] = "user-2"

	// Verify the original message was not affected by the modification of the tap copy
	if result.Headers["X-User"] != originalUserValue {
		t.Errorf("Original message headers were modified by tap copy modification: expected %q, got %q", originalUserValue, result.Headers["X-User"])
	}
}

// TestWiretapNilMessage verifies that nil messages are handled gracefully
func TestWiretapNilMessage(t *testing.T) {
	ctx := context.Background()

	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	tapSink := engine.NewChannel("tap_sink", 10)
	step.SetTapSink(tapSink)

	// Execute with nil message
	result, err := step.Execute(ctx, nil)
	if err != nil {
		t.Errorf("Expected no error for nil message, got %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for nil message")
	}

	// No message should be sent to tap sink
	tapMsg := tapSink.TryRecv()
	if tapMsg != nil {
		t.Error("Unexpected message received from tap sink for nil input")
	}
}

// TestWiretapNoTapSinkConfigured verifies graceful handling when tap sink not configured
func TestWiretapNoTapSinkConfigured(t *testing.T) {
	ctx := context.Background()

	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	// Don't configure the tap sink

	// Create a test message
	msg := engine.NewMessage(
		map[string]interface{}{
			"id": "order-1",
		},
		"test-route",
		"v1",
	)

	// Execute should still succeed and return the original message
	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	if result.Body.(map[string]interface{})["id"] != "order-1" {
		t.Error("Original message body was modified")
	}
}

// TestWiretapBufferFull verifies non-blocking behavior when tap sink buffer is full
func TestWiretapBufferFull(t *testing.T) {
	ctx := context.Background()

	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	// Create a small tap sink with buffer size 1
	tapSink := engine.NewChannel("tap_sink", 1)
	step.SetTapSink(tapSink)

	// Fill the tap sink buffer with one message
	preMsg := engine.NewMessage(
		map[string]interface{}{"id": "pre-msg"},
		"test-route",
		"v1",
	)
	if err := tapSink.Send(context.Background(), preMsg); err != nil {
		t.Fatalf("Failed to fill tap sink buffer: %v", err)
	}

	// Create another message for wiretap
	msg := engine.NewMessage(
		map[string]interface{}{"id": "order-1"},
		"test-route",
		"v1",
	)

	// Execute should still succeed and return immediately (non-blocking)
	// The tap send happens in a goroutine, so it won't block the main flow
	start := time.Now()
	result, err := step.Execute(ctx, msg)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// Verify it returned quickly (much less than 1 second)
	if elapsed > 500*time.Millisecond {
		t.Errorf("Execute took too long (%v), suggests it may have blocked", elapsed)
	}

	// Verify the original message is still correct
	if result.Body.(map[string]interface{})["id"] != "order-1" {
		t.Error("Original message body was modified")
	}
}

// TestWiretapMultipleMessagesSequential verifies wiretap works with multiple messages
func TestWiretapMultipleMessagesSequential(t *testing.T) {
	ctx := context.Background()

	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	tapSink := engine.NewChannel("tap_sink", 10)
	step.SetTapSink(tapSink)

	// Send multiple messages through wiretap
	messages := []map[string]interface{}{
		{"id": "msg-1", "value": 100},
		{"id": "msg-2", "value": 200},
		{"id": "msg-3", "value": 300},
	}

	results := make([]*engine.Message, 0)

	for _, msgBody := range messages {
		msg := engine.NewMessage(msgBody, "test-route", "v1")
		result, err := step.Execute(ctx, msg)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
		if result == nil {
			t.Fatal("Result message is nil")
		}
		results = append(results, result)
	}

	// Give goroutines time to send
	time.Sleep(200 * time.Millisecond)

	// Verify all originals were returned
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify all copies were sent to tap sink
	// Note: Due to concurrent goroutines, messages may arrive out of order
	// So we collect all messages and verify they're all there
	receivedIDs := make(map[string]bool)
	for i := 0; i < 3; i++ {
		tapCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		tapMsg, err := tapSink.Recv(tapCtx)
		if err != nil {
			t.Fatalf("Failed to receive message %d from tap sink: %v", i, err)
		}

		if tapMsg == nil {
			t.Fatalf("No message %d received from tap sink", i)
		}

		actualID := tapMsg.Body.(map[string]interface{})["id"].(string)
		receivedIDs[actualID] = true
	}

	// Verify we got all three messages
	expectedIDs := map[string]bool{"msg-1": true, "msg-2": true, "msg-3": true}
	for id := range expectedIDs {
		if !receivedIDs[id] {
			t.Errorf("Expected message %q not received", id)
		}
	}
	for id := range receivedIDs {
		if !expectedIDs[id] {
			t.Errorf("Unexpected message %q received", id)
		}
	}
}

// TestWiretapPreservesMessageMetadata verifies all message metadata is preserved
func TestWiretapPreservesMessageMetadata(t *testing.T) {
	ctx := context.Background()

	step, err := NewWiretapStep("audit_log")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	tapSink := engine.NewChannel("tap_sink", 10)
	step.SetTapSink(tapSink)

	// Create a message with metadata
	msg := engine.NewMessage(
		map[string]interface{}{"data": "test"},
		"important-route",
		"v1",
	)

	originalCorrID := msg.Metadata.CorrelationID
	originalIngestedAt := msg.Metadata.IngestedAt

	// Add principal to metadata
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user-123",
		Roles:   []string{"admin"},
	}

	// Execute
	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// Verify result metadata
	if result.Metadata.CorrelationID != originalCorrID {
		t.Error("CorrelationID not preserved in result")
	}

	if result.Metadata.IngestedAt != originalIngestedAt {
		t.Error("IngestedAt not preserved in result")
	}

	if result.Metadata.Route != "important-route" {
		t.Errorf("Route not preserved: expected 'important-route', got %q", result.Metadata.Route)
	}

	// Give goroutine time to send
	time.Sleep(100 * time.Millisecond)

	// Verify tapped message metadata
	tapCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	tapMsg, err := tapSink.Recv(tapCtx)
	if err != nil {
		t.Fatalf("Failed to receive from tap sink: %v", err)
	}

	if tapMsg == nil {
		t.Fatal("No message received from tap sink")
	}

	if tapMsg.Metadata.CorrelationID != originalCorrID {
		t.Error("CorrelationID not preserved in tapped message")
	}

	if tapMsg.Metadata.Principal == nil || tapMsg.Metadata.Principal.Subject != "user-123" {
		t.Error("Principal not preserved in tapped message")
	}
}

// TestWiretapIntegrationWithExecutor verifies wiretap works in executor pipeline
func TestWiretapIntegrationWithExecutor(t *testing.T) {
	ctx := context.Background()

	// Create wiretap step
	wiretapStep, err := NewWiretapStep("audit_sink")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	// Create channels
	inputCh := engine.NewChannel("input", 10)
	outputCh := engine.NewChannel("output", 10)
	auditCh := engine.NewChannel("audit_sink", 10)

	// Wire up the audit sink
	wiretapStep.SetTapSink(auditCh)

	// Create executor with just the wiretap step
	executor := engine.NewExecutor("test", inputCh, outputCh, []engine.Step{wiretapStep})

	// Start executor in background
	go func() {
		if err := executor.Run(ctx); err != nil {
			t.Logf("Executor error: %v", err)
		}
	}()

	// Send a message
	msg := engine.NewMessage(
		map[string]interface{}{"order": "O123"},
		"test-route",
		"v1",
	)

	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Receive from output
	outCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	outputMsg, err := outputCh.Recv(outCtx)
	if err != nil {
		t.Fatalf("Failed to receive from output: %v", err)
	}

	if outputMsg == nil {
		t.Fatal("No message received from output")
	}

	if outputMsg.Body.(map[string]interface{})["order"] != "O123" {
		t.Error("Output message body incorrect")
	}

	// Give time for audit send to complete
	time.Sleep(200 * time.Millisecond)

	// Receive from audit sink
	auditCtx, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	auditMsg, err := auditCh.Recv(auditCtx)
	if err != nil {
		t.Fatalf("Failed to receive from audit: %v", err)
	}

	if auditMsg == nil {
		t.Fatal("No message received from audit sink")
	}

	if auditMsg.Body.(map[string]interface{})["order"] != "O123" {
		t.Error("Audit message body incorrect")
	}

	// Close input to signal completion
	inputCh.Close()

	// Wait a bit for executor to finish
	time.Sleep(500 * time.Millisecond)
}

// TestWiretapSetTapSink verifies SetTapSink method works correctly
func TestWiretapSetTapSink(t *testing.T) {
	step, err := NewWiretapStep("test_sink")
	if err != nil {
		t.Fatalf("NewWiretapStep failed: %v", err)
	}

	// Initially, sink channel should be nil
	if step.sinkChannel != nil {
		t.Error("Sink channel should initially be nil")
	}

	// Set the tap sink
	ch := engine.NewChannel("test_sink", 10)
	step.SetTapSink(ch)

	// Now it should be set
	if step.sinkChannel != ch {
		t.Error("Sink channel not set correctly")
	}
}
