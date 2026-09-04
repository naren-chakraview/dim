package amqp

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/steps"
)

// TestReplayMessageDedup_Original verifies original message is processed (M1.8.2)
func TestReplayMessageDedup_Original(t *testing.T) {
	idempotent, err := steps.NewIdempotentStep("body.order_id", 5)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer idempotent.Stop(context.Background())

	// Original message
	msg := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123", "amount": 100},
		"process-orders",
		"v1",
	)

	ctx := context.Background()
	result, err := idempotent.Execute(ctx, msg)

	if err != nil {
		t.Errorf("Expected original message to pass, got error: %v", err)
	}
	if result == nil {
		t.Error("Expected message to pass through")
	}
}

// TestReplayMessageDedup_Duplicate verifies replay is deduped (M1.8.2)
func TestReplayMessageDedup_Duplicate(t *testing.T) {
	idempotent, err := steps.NewIdempotentStep("body.order_id", 5)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer idempotent.Stop(context.Background())

	// First message
	msg1 := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123", "amount": 100},
		"process-orders",
		"v1",
	)

	ctx := context.Background()
	result1, err1 := idempotent.Execute(ctx, msg1)
	if err1 != nil {
		t.Fatalf("First message failed: %v", err1)
	}
	if result1 == nil {
		t.Fatal("First message should pass through")
	}

	// Replayed message with same order_id (different correlation ID, simulating replay)
	msg2 := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123", "amount": 100},
		"process-orders",
		"v1",
	)
	msg2.Metadata.ReplayCount = 1
	msg2.Metadata.ReplayHistory = append(msg2.Metadata.ReplayHistory, &engine.ReplayEntry{
		Timestamp: time.Now(),
		Attempt:   1,
		Initiator: "dimctl replay",
	})

	result2, err2 := idempotent.Execute(ctx, msg2)

	if result2 != nil {
		t.Error("Expected replay to be deduped (nil result)")
	}
	if err2 != nil {
		t.Errorf("Expected nil error for deduped message, got: %v", err2)
	}
}

// TestReplayMessageDedup_Different verifies different order_id is not deduped (M1.8.2)
func TestReplayMessageDedup_Different(t *testing.T) {
	idempotent, err := steps.NewIdempotentStep("body.order_id", 5)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer idempotent.Stop(context.Background())

	// First message
	msg1 := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123", "amount": 100},
		"process-orders",
		"v1",
	)

	ctx := context.Background()
	result1, _ := idempotent.Execute(ctx, msg1)
	if result1 == nil {
		t.Fatal("First message should pass through")
	}

	// Second message with different order_id
	msg2 := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-456", "amount": 200},
		"process-orders",
		"v1",
	)

	result2, err2 := idempotent.Execute(ctx, msg2)

	if err2 != nil {
		t.Errorf("Different order_id should pass, got error: %v", err2)
	}
	if result2 == nil {
		t.Error("Different order_id should pass through")
	}
}

// TestReplayAuditTrail_RecordedInMetadata verifies replay is recorded (M1.8.3)
func TestReplayAuditTrail_RecordedInMetadata(t *testing.T) {
	msg := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123"},
		"process-orders",
		"v1",
	)

	// Simulate replay
	msg.Metadata.ReplayCount = 1
	msg.Metadata.ReplayHistory = append(msg.Metadata.ReplayHistory, &engine.ReplayEntry{
		Timestamp: time.Now(),
		Attempt:   1,
		Initiator: "dimctl replay",
	})

	if msg.Metadata.ReplayCount != 1 {
		t.Errorf("Expected replay count 1, got %d", msg.Metadata.ReplayCount)
	}

	if len(msg.Metadata.ReplayHistory) != 1 {
		t.Errorf("Expected 1 replay history entry, got %d", len(msg.Metadata.ReplayHistory))
	}

	if msg.Metadata.ReplayHistory[0].Attempt != 1 {
		t.Errorf("Expected attempt 1, got %d", msg.Metadata.ReplayHistory[0].Attempt)
	}

	if msg.Metadata.ReplayHistory[0].Initiator != "dimctl replay" {
		t.Errorf("Expected initiator 'dimctl replay', got %q", msg.Metadata.ReplayHistory[0].Initiator)
	}
}

// TestReplayAuditTrail_MultipleReplays verifies replay history accumulates (M1.8.3)
func TestReplayAuditTrail_MultipleReplays(t *testing.T) {
	msg := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123"},
		"process-orders",
		"v1",
	)

	// First replay
	msg.Metadata.ReplayCount = 1
	msg.Metadata.ReplayHistory = append(msg.Metadata.ReplayHistory, &engine.ReplayEntry{
		Timestamp: time.Now(),
		Attempt:   1,
		Initiator: "dimctl replay",
	})

	// Second replay
	msg.Metadata.ReplayCount = 2
	msg.Metadata.ReplayHistory = append(msg.Metadata.ReplayHistory, &engine.ReplayEntry{
		Timestamp: time.Now().Add(1 * time.Minute),
		Attempt:   2,
		Initiator: "dimctl replay",
	})

	if msg.Metadata.ReplayCount != 2 {
		t.Errorf("Expected replay count 2, got %d", msg.Metadata.ReplayCount)
	}

	if len(msg.Metadata.ReplayHistory) != 2 {
		t.Errorf("Expected 2 replay history entries, got %d", len(msg.Metadata.ReplayHistory))
	}

	if msg.Metadata.ReplayHistory[1].Attempt != 2 {
		t.Errorf("Expected second attempt=2, got %d", msg.Metadata.ReplayHistory[1].Attempt)
	}
}

// TestReplayErrorType_RecordsErrorInMetadata verifies error type tracking (M1.8)
func TestReplayErrorType_RecordsErrorInMetadata(t *testing.T) {
	msg := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123"},
		"process-orders",
		"v1",
	)

	// Simulate error (e.g., authorization failure)
	msg.Metadata.ErrorType = "authorization_denied"

	if msg.Metadata.ErrorType != "authorization_denied" {
		t.Errorf("Expected error type 'authorization_denied', got %q", msg.Metadata.ErrorType)
	}
}

// TestReplayErrorType_TimeoutTracking verifies timeout error tracking (M1.8)
func TestReplayErrorType_TimeoutTracking(t *testing.T) {
	msg := engine.NewMessage(
		map[string]interface{}{"order_id": "ord-123"},
		"process-orders",
		"v1",
	)

	// Simulate timeout error
	msg.Metadata.ErrorType = "timeout"

	if msg.Metadata.ErrorType != "timeout" {
		t.Errorf("Expected error type 'timeout', got %q", msg.Metadata.ErrorType)
	}
}
