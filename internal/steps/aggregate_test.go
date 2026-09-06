package steps

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestAggregateCountCompletion verifies count-based aggregation (M2.1.2)
func TestAggregateCountCompletion(t *testing.T) {
	spec := &AggregateSpec{
		CorrelationKey:     "order_id",
		CompletionStrategy: "count",
		Count:              3,
		TimeoutMs:          10000,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Create 3 messages with same order_id
	messages := []*engine.Message{
		engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "A"}, "test-route", "v1"),
		engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "B"}, "test-route", "v1"),
		engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "C"}, "test-route", "v1"),
	}

	// Execute first two messages (should not output)
	out1, err := step.Execute(ctx, messages[0])
	if err != nil {
		t.Errorf("Execute 1 failed: %v", err)
	}
	if out1 != nil {
		t.Errorf("Expected no output for message 1, got %v", out1)
	}

	out2, err := step.Execute(ctx, messages[1])
	if err != nil {
		t.Errorf("Execute 2 failed: %v", err)
	}
	if out2 != nil {
		t.Errorf("Expected no output for message 2, got %v", out2)
	}

	// Execute third message (should trigger flush)
	out3, err := step.Execute(ctx, messages[2])
	if err != nil {
		t.Errorf("Execute 3 failed: %v", err)
	}
	if out3 == nil {
		t.Errorf("Expected output on 3rd message (count reached), got nil")
	}

	// Verify aggregated output structure
	if out3 != nil {
		body := out3.Body.(map[string]interface{})
		if msgList, ok := body["messages"].([]interface{}); ok {
			if len(msgList) != 3 {
				t.Errorf("Expected 3 aggregated messages, got %d", len(msgList))
			}
		} else {
			t.Errorf("Expected 'messages' field in output")
		}

		if count, ok := body["count"].(int); ok {
			if count != 3 {
				t.Errorf("Expected count=3, got %d", count)
			}
		}

		if trigger, ok := body["completion_trigger"].(string); ok {
			if trigger != "count_reached" {
				t.Errorf("Expected completion_trigger='count_reached', got %s", trigger)
			}
		}
	}
}

// TestAggregateTimeWindowCompletion verifies time-window-based aggregation (M2.1.2)
func TestAggregateTimeWindowCompletion(t *testing.T) {
spec := &AggregateSpec{
		CorrelationKey:     "order_id",
		CompletionStrategy: "time_window",
		WindowMs:           100,
		MaxMessages:        10,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Add one message
	msg := engine.NewMessage(map[string]interface{}{"order_id": "ORD-002", "item": "X"}, "test-route", "v1")
	out, err := step.Execute(ctx, msg)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if out != nil {
		t.Errorf("Expected no output immediately, got %v", out)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Window should have expired and flushed the group
	// (Note: in this implementation, flush is async; we can't easily test the output)
	// Verify group was cleared
	step.mu.RLock()
	groupCount := len(step.groups)
	step.mu.RUnlock()

	if groupCount != 0 {
		t.Errorf("Expected groups to be flushed after window, %d remain", groupCount)
	}
}

// TestAggregateNullCorrelationKey verifies null key handling (M2.1.2)
func TestAggregateNullCorrelationKey(t *testing.T) {
	spec := &AggregateSpec{
		CorrelationKey:     "order_id",
		CompletionStrategy: "count",
		Count:              2,
		OnNullCorrelation:  "error_path",
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Message with null order_id
	msg := engine.NewMessage(map[string]interface{}{"order_id": nil, "item": "A"}, "test-route", "v1")
	out, err := step.Execute(ctx, msg)

	if err == nil {
		t.Errorf("Expected error for null correlation key, got nil")
	}
	if out != nil {
		t.Errorf("Expected no output for null correlation key, got %v", out)
	}
}

// TestAggregateMultipleCorrelationKeys verifies independent groups (M2.1.2)
func TestAggregateMultipleCorrelationKeys(t *testing.T) {
spec := &AggregateSpec{
		CorrelationKey:     "order_id",
		CompletionStrategy: "count",
		Count:              2,
		TimeoutMs:          10000,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Create messages for two different orders
	msg1 := engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "A"}, "test-route", "v1")
	msg2 := engine.NewMessage(map[string]interface{}{"order_id": "ORD-002", "item": "B"}, "test-route", "v1")
	msg3 := engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "C"}, "test-route", "v1")

	// Add first message from ORD-001
	out1, _ := step.Execute(ctx, msg1)
	if out1 != nil {
		t.Errorf("Expected no output, got %v", out1)
	}

	// Add first message from ORD-002
	out2, _ := step.Execute(ctx, msg2)
	if out2 != nil {
		t.Errorf("Expected no output, got %v", out2)
	}

	// Verify both groups exist
	step.mu.RLock()
	groupCount := len(step.groups)
	step.mu.RUnlock()
	if groupCount != 2 {
		t.Errorf("Expected 2 groups, got %d", groupCount)
	}

	// Add second message from ORD-001 (should complete that group)
	out3, _ := step.Execute(ctx, msg3)
	if out3 == nil {
		t.Errorf("Expected output when ORD-001 reaches count=2")
	}

	// Verify ORD-002 group still exists
	step.mu.RLock()
	groupCount = len(step.groups)
	step.mu.RUnlock()
	if groupCount != 1 {
		t.Errorf("Expected 1 remaining group, got %d", groupCount)
	}
}

// TestAggregateDrain verifies hot-reload drain behavior (M2.1.2)
func TestAggregateDrain(t *testing.T) {
spec := &AggregateSpec{
		CorrelationKey:     "order_id",
		CompletionStrategy: "count",
		Count:              10, // High count so groups don't auto-flush
		TimeoutMs:          60000,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Add messages to multiple groups
	for i := 1; i <= 3; i++ {
		msg := engine.NewMessage(
			map[string]interface{}{"order_id": "ORD-001", "item": i},
			"test-route", "v1",
		)
		step.Execute(ctx, msg)
	}

	// Verify groups exist
	step.mu.RLock()
	initialGroups := len(step.groups)
	step.mu.RUnlock()
	if initialGroups != 1 {
		t.Errorf("Expected 1 group, got %d", initialGroups)
	}

	// Drain the aggregator (hot reload)
	drainCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = step.Drain(drainCtx, 1*time.Second)
	if err != nil {
		t.Errorf("Drain failed: %v", err)
	}

	// Verify groups were flushed
	step.mu.RLock()
	remainingGroups := len(step.groups)
	step.mu.RUnlock()
	if remainingGroups != 0 {
		t.Errorf("Expected 0 groups after drain, got %d", remainingGroups)
	}
}

// TestAggregateSpec validation verifies configuration validation (M2.1.2)
func TestAggregateSpecValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    *AggregateSpec
		wantErr bool
	}{
		{
			name: "valid count strategy",
			spec: &AggregateSpec{
				CorrelationKey:     "id",
				CompletionStrategy: "count",
				Count:              5,
			},
			wantErr: false,
		},
		{
			name: "valid time_window strategy",
			spec: &AggregateSpec{
				CorrelationKey:     "id",
				CompletionStrategy: "time_window",
				WindowMs:           1000,
			},
			wantErr: false,
		},
		{
			name: "missing correlation_key",
			spec: &AggregateSpec{
				CompletionStrategy: "count",
				Count:              5,
			},
			wantErr: true,
		},
		{
			name: "invalid completion_strategy",
			spec: &AggregateSpec{
				CorrelationKey:     "id",
				CompletionStrategy: "invalid",
			},
			wantErr: true,
		},
		{
			name: "count strategy with count=0",
			spec: &AggregateSpec{
				CorrelationKey:     "id",
				CompletionStrategy: "count",
				Count:              0,
			},
			wantErr: true,
		},
		{
			name: "time_window strategy with window_ms=0",
			spec: &AggregateSpec{
				CorrelationKey:     "id",
				CompletionStrategy: "time_window",
				WindowMs:           0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAggregateStep(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAggregateStep() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAggregateOutputStructure verifies output message structure (M2.1.2)
func TestAggregateOutputStructure(t *testing.T) {
spec := &AggregateSpec{
		CorrelationKey:     "order_id",
		CompletionStrategy: "count",
		Count:              2,
	}

	step, _ := NewAggregateStep(spec)
	ctx := context.Background()

	msg1 := engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "A"}, "test-route", "v1")
	msg2 := engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "B"}, "test-route", "v1")

	step.Execute(ctx, msg1)
	out, _ := step.Execute(ctx, msg2)

	if out == nil {
		t.Fatalf("Expected output, got nil")
	}

	body := out.Body.(map[string]interface{})

	// Verify structure
	if _, ok := body["messages"]; !ok {
		t.Errorf("Missing 'messages' field")
	}
	if _, ok := body["correlation_key"]; !ok {
		t.Errorf("Missing 'correlation_key' field")
	}
	if _, ok := body["count"]; !ok {
		t.Errorf("Missing 'count' field")
	}
	if _, ok := body["first_ingested"]; !ok {
		t.Errorf("Missing 'first_ingested' field")
	}
	if _, ok := body["last_ingested"]; !ok {
		t.Errorf("Missing 'last_ingested' field")
	}
	if _, ok := body["completion_trigger"]; !ok {
		t.Errorf("Missing 'completion_trigger' field")
	}

	// Verify aggregator facet on metadata
	if out.Metadata.AggregatorFacet == nil {
		t.Errorf("Expected AggregatorFacet in metadata")
	}
}
