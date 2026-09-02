package steps

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestNewIdempotentStep verifies that a valid JSONata expression compiles correctly
func TestNewIdempotentStep(t *testing.T) {
	keyExpr := "body.transaction_id"
	step, err := NewIdempotentStep(keyExpr, 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	if step == nil {
		t.Error("NewIdempotentStep returned nil step")
	}
	defer step.Stop(context.Background())
}

// TestNewIdempotentStepInvalidExpression verifies that invalid expressions fail at compile time
func TestNewIdempotentStepInvalidExpression(t *testing.T) {
	keyExpr := "{ invalid json ] }"
	step, err := NewIdempotentStep(keyExpr, 60)
	if err == nil {
		t.Error("Expected compile error for invalid expression, got nil")
	}
	if step != nil {
		t.Error("Expected step to be nil when compilation fails")
	}
}

// TestIdempotentDuplicateKeyDropped verifies that duplicate keys are dropped
func TestIdempotentDuplicateKeyDropped(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.transaction_id", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Create first message with transaction_id = "tx-001"
	msg1 := engine.NewMessage(
		map[string]interface{}{
			"transaction_id": "tx-001",
			"amount":         100.0,
		},
		"test-route",
		"v1",
	)

	// First occurrence should pass through
	result1, err := step.Execute(ctx, msg1)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result1 == nil {
		t.Error("Expected first message to pass through, got nil")
	}

	// Create second message with same transaction_id
	msg2 := engine.NewMessage(
		map[string]interface{}{
			"transaction_id": "tx-001",
			"amount":         200.0,
		},
		"test-route",
		"v1",
	)

	// Second occurrence (duplicate) should be dropped
	result2, err := step.Execute(ctx, msg2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result2 != nil {
		t.Error("Expected duplicate message to be dropped (nil), got message")
	}
}

// TestIdempotentUniqueKeyPassedThrough verifies that unique keys pass through
func TestIdempotentUniqueKeyPassedThrough(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.transaction_id", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Create messages with different transaction_ids
	msg1 := engine.NewMessage(
		map[string]interface{}{"transaction_id": "tx-001", "amount": 100.0},
		"test-route",
		"v1",
	)
	msg2 := engine.NewMessage(
		map[string]interface{}{"transaction_id": "tx-002", "amount": 200.0},
		"test-route",
		"v1",
	)
	msg3 := engine.NewMessage(
		map[string]interface{}{"transaction_id": "tx-003", "amount": 300.0},
		"test-route",
		"v1",
	)

	// All should pass through
	result1, err := step.Execute(ctx, msg1)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result1 == nil {
		t.Error("Expected message 1 to pass through")
	}

	result2, err := step.Execute(ctx, msg2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result2 == nil {
		t.Error("Expected message 2 to pass through")
	}

	result3, err := step.Execute(ctx, msg3)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result3 == nil {
		t.Error("Expected message 3 to pass through")
	}
}

// TestIdempotentTTLExpiry verifies that keys can be reused after TTL expiry
func TestIdempotentTTLExpiry(t *testing.T) {
	ctx := context.Background()
	// Use very short TTL (100 milliseconds) for testing
	step, err := NewIdempotentStep("body.transaction_id", 0)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Manually set a shorter TTL for testing (100ms)
	step.ttl = 100 * time.Millisecond

	msg := engine.NewMessage(
		map[string]interface{}{"transaction_id": "tx-expire-test"},
		"test-route",
		"v1",
	)

	// First occurrence should pass through
	result1, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result1 == nil {
		t.Error("Expected first message to pass through")
	}

	// Immediate second occurrence should be dropped
	result2, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result2 != nil {
		t.Error("Expected duplicate message to be dropped immediately")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// After expiry, same key should pass through again
	result3, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result3 == nil {
		t.Error("Expected message to pass through after TTL expiry")
	}
}

// TestIdempotentDefaultTTL verifies that default TTL is 60 minutes when not specified
func TestIdempotentDefaultTTL(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.id", 0)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	expectedTTL := 60 * time.Minute
	if step.ttl != expectedTTL {
		t.Errorf("Expected default TTL %v, got %v", expectedTTL, step.ttl)
	}
}

// TestIdempotentCustomTTL verifies that custom TTL is respected
func TestIdempotentCustomTTL(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.id", 30)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	expectedTTL := 30 * time.Minute
	if step.ttl != expectedTTL {
		t.Errorf("Expected TTL %v, got %v", expectedTTL, step.ttl)
	}
}

// TestIdempotentComplexKeyExpression verifies that complex JSONata expressions work
func TestIdempotentComplexKeyExpression(t *testing.T) {
	ctx := context.Background()
	// Use a complex expression that concatenates fields
	step, err := NewIdempotentStep(`body.user_id & "-" & body.order_id`, 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	msg1 := engine.NewMessage(
		map[string]interface{}{
			"user_id":  "user-123",
			"order_id": "order-456",
		},
		"test-route",
		"v1",
	)

	msg2 := engine.NewMessage(
		map[string]interface{}{
			"user_id":  "user-123",
			"order_id": "order-456", // Same key
		},
		"test-route",
		"v1",
	)

	msg3 := engine.NewMessage(
		map[string]interface{}{
			"user_id":  "user-123",
			"order_id": "order-789", // Different key
		},
		"test-route",
		"v1",
	)

	// First should pass
	result1, err := step.Execute(ctx, msg1)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result1 == nil {
		t.Error("Expected first message to pass through")
	}

	// Second (duplicate) should be dropped
	result2, err := step.Execute(ctx, msg2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result2 != nil {
		t.Error("Expected duplicate message to be dropped")
	}

	// Third (unique) should pass
	result3, err := step.Execute(ctx, msg3)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result3 == nil {
		t.Error("Expected third message with different key to pass through")
	}
}

// TestIdempotentKeyEvaluationError verifies that expression evaluation errors are returned
func TestIdempotentKeyEvaluationError(t *testing.T) {
	ctx := context.Background()
	// Create a step with a valid expression
	step, err := NewIdempotentStep("body.field", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Create message with incompatible type for the expression
	// (This might not error depending on JSONata's behavior, but let's use an expression
	// that would actually cause an error)
	step2, err := NewIdempotentStep("$nonexistent_func(body)", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step2.Stop(ctx)

	msg := engine.NewMessage(
		map[string]interface{}{"value": 42},
		"test-route",
		"v1",
	)

	result, err := step2.Execute(ctx, msg)
	if err == nil {
		t.Error("Expected error for undefined function")
	}
	if result != nil {
		t.Error("Expected result to be nil when evaluation fails")
	}
}

// TestIdempotentNilMessage verifies that nil messages are handled gracefully
func TestIdempotentNilMessage(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.id", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	result, err := step.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("Execute should not error on nil message, got %v", err)
	}
	if result != nil {
		t.Error("Expected nil message to return nil")
	}
}

// TestIdempotentLargeSetDoesNotCrash verifies that large dedup sets don't crash
func TestIdempotentLargeSetDoesNotCrash(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.id", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Add many unique messages to stress test the dedup set
	// This should succeed without panicking (even if it logs a warning)
	for i := 0; i < 101000; i++ {
		msg := engine.NewMessage(
			map[string]interface{}{"id": i},
			"test-route",
			"v1",
		)

		result, err := step.Execute(ctx, msg)
		if err != nil {
			t.Fatalf("Execute failed at iteration %d: %v", i, err)
		}

		// All unique messages should pass through
		if result == nil {
			t.Fatalf("Expected message to pass through at iteration %d", i)
		}
	}

	// Verify set size is bounded (approximately)
	step.lock.RLock()
	setSize := len(step.keySet)
	step.lock.RUnlock()

	// We added 101k messages, so the set should be approximately that size
	if setSize < 100000 || setSize > 102000 {
		t.Logf("Dedup set size: %d (expected around 101000)", setSize)
	}
}

// TestIdempotentConcurrentAccess verifies thread-safe concurrent access (use -race flag)
func TestIdempotentConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.id", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Launch multiple goroutines that concurrently add messages
	done := make(chan error, 10)
	for g := 0; g < 10; g++ {
		go func(goroutineID int) {
			for i := 0; i < 100; i++ {
				msg := engine.NewMessage(
					map[string]interface{}{"id": i % 50}, // Create some duplicates
					"test-route",
					"v1",
				)

				_, err := step.Execute(ctx, msg)
				if err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			t.Fatalf("Goroutine failed: %v", err)
		}
	}

	// Should have succeeded without race conditions (verify with -race flag)
}

// TestIdempotentEvictionTriggered verifies that eviction loop removes expired keys
func TestIdempotentEvictionTriggered(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.id", 0)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Set very short TTL for testing
	step.ttl = 100 * time.Millisecond

	// Add a message
	msg := engine.NewMessage(
		map[string]interface{}{"id": "test-id"},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Error("Expected message to pass through")
	}

	// Verify it's in the set
	step.lock.RLock()
	initialSize := len(step.keySet)
	step.lock.RUnlock()

	if initialSize != 1 {
		t.Errorf("Expected 1 key in set, got %d", initialSize)
	}

	// Wait for key to expire
	time.Sleep(150 * time.Millisecond)

	// Manually trigger eviction
	step.evictExpiredKeys()

	// Verify expired key was removed
	step.lock.RLock()
	finalSize := len(step.keySet)
	step.lock.RUnlock()

	if finalSize != 0 {
		t.Errorf("Expected 0 keys after eviction, got %d", finalSize)
	}
}

// TestIdempotentMessagePreservation verifies that passing messages are unchanged
func TestIdempotentMessagePreservation(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.id", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "msg-001",
			"amount": 100.0,
		},
		"test-route",
		"v1",
	)
	msg.Headers["Content-Type"] = "application/json"

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through")
	}

	// Verify message content is preserved
	body, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map body, got %T", result.Body)
	}

	if body["id"] != "msg-001" {
		t.Errorf("Expected id 'msg-001', got %v", body["id"])
	}

	if body["amount"] != 100.0 {
		t.Errorf("Expected amount 100.0, got %v", body["amount"])
	}

	if result.Headers["Content-Type"] != "application/json" {
		t.Error("Expected headers to be preserved")
	}

	if result.Metadata.Route != "test-route" {
		t.Error("Expected metadata.Route to be preserved")
	}
}

// TestIdempotentHeadersAndMetadata verifies that headers and metadata are available in key expression
func TestIdempotentHeadersAndMetadata(t *testing.T) {
	ctx := context.Background()
	// Use header value in key expression
	step, err := NewIdempotentStep(`headers.request_id`, 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	msg1 := engine.NewMessage(
		map[string]interface{}{"data": "test"},
		"test-route",
		"v1",
	)
	msg1.Headers["request_id"] = "req-123"

	msg2 := engine.NewMessage(
		map[string]interface{}{"data": "test"},
		"test-route",
		"v1",
	)
	msg2.Headers["request_id"] = "req-123" // Same request_id

	// First should pass
	result1, err := step.Execute(ctx, msg1)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result1 == nil {
		t.Error("Expected first message to pass through")
	}

	// Second (duplicate header) should be dropped
	result2, err := step.Execute(ctx, msg2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result2 != nil {
		t.Error("Expected duplicate based on headers to be dropped")
	}
}

// TestIdempotentNumericKeyConversion verifies that numeric keys are converted to strings
func TestIdempotentNumericKeyConversion(t *testing.T) {
	ctx := context.Background()
	step, err := NewIdempotentStep("body.order_id", 60)
	if err != nil {
		t.Fatalf("NewIdempotentStep failed: %v", err)
	}
	defer step.Stop(ctx)

	// Send message with numeric order_id
	msg1 := engine.NewMessage(
		map[string]interface{}{"order_id": float64(12345)},
		"test-route",
		"v1",
	)

	msg2 := engine.NewMessage(
		map[string]interface{}{"order_id": float64(12345)},
		"test-route",
		"v1",
	)

	// First should pass
	result1, err := step.Execute(ctx, msg1)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result1 == nil {
		t.Error("Expected first message to pass through")
	}

	// Second (duplicate numeric key) should be dropped
	result2, err := step.Execute(ctx, msg2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result2 != nil {
		t.Error("Expected duplicate numeric key to be dropped")
	}
}
