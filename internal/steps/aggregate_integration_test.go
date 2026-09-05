package steps

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestAggregatorIntegration verifies end-to-end aggregation with lineage (M2.1.3)
func TestAggregatorIntegration(t *testing.T) {
	spec := &AggregateSpec{
		CorrelationKey:     "body.order_id",
		CompletionStrategy: "count",
		Count:              3,
		TimeoutMs:          5000,
		OutputExpr: `{
			"order_id": correlation_key,
			"items": $map(messages, $ -> $.body),
			"item_count": count,
			"total_qty": $sum(messages, $ -> $.body.qty),
			"completion": completion_trigger
		}`,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Simulate 3 line items for order ORD-001
	lineItems := []*engine.Message{
		engine.NewMessage(
			map[string]interface{}{
				"order_id": "ORD-001",
				"sku":      "SKU-A",
				"qty":      2,
			},
			"order-processing",
			"v1",
		),
		engine.NewMessage(
			map[string]interface{}{
				"order_id": "ORD-001",
				"sku":      "SKU-B",
				"qty":      3,
			},
			"order-processing",
			"v1",
		),
		engine.NewMessage(
			map[string]interface{}{
				"order_id": "ORD-001",
				"sku":      "SKU-C",
				"qty":      5,
			},
			"order-processing",
			"v1",
		),
	}

	// Process line items
	var aggregatedOutput *engine.Message
	for i, item := range lineItems {
		out, err := step.Execute(ctx, item)
		if err != nil {
			t.Fatalf("Line item %d failed: %v", i+1, err)
		}

		if out != nil {
			aggregatedOutput = out
		}
	}

	// Verify aggregation occurred
	if aggregatedOutput == nil {
		t.Fatalf("Expected aggregated output, got nil")
	}

	// Verify output structure
	body := aggregatedOutput.Body.(map[string]interface{})

	// Check order_id
	if orderID, ok := body["order_id"].(string); !ok || orderID != "ORD-001" {
		t.Errorf("Expected order_id='ORD-001', got %v", body["order_id"])
	}

	// Check item count
	if itemCount, ok := body["item_count"].(float64); !ok || itemCount != 3 {
		t.Errorf("Expected item_count=3, got %v", body["item_count"])
	}

	// Check total quantity
	if totalQty, ok := body["total_qty"].(float64); !ok || totalQty != 10 {
		t.Errorf("Expected total_qty=10 (2+3+5), got %v", body["total_qty"])
	}

	// Check completion trigger
	if trigger, ok := body["completion"].(string); !ok || trigger != "count_reached" {
		t.Errorf("Expected completion='count_reached', got %v", body["completion"])
	}

	// Verify aggregator facet on metadata
	if facet := aggregatedOutput.Metadata.AggregatorFacet; facet == nil {
		t.Errorf("Expected AggregatorFacet in metadata")
	} else {
		// Verify facet fields
		facetObj := facet.(*AggregatorFacet)
		if facetObj.InputCount != 3 {
			t.Errorf("Expected InputCount=3, got %d", facetObj.InputCount)
		}
		if facetObj.CompletionTrigger != "count_reached" {
			t.Errorf("Expected CompletionTrigger='count_reached', got %s", facetObj.CompletionTrigger)
		}
	}

	// Verify groups were cleared
	step.mu.RLock()
	groupCount := len(step.groups)
	step.mu.RUnlock()
	if groupCount != 0 {
		t.Errorf("Expected groups to be cleared after completion, %d remain", groupCount)
	}
}

// TestAggregatorMultipleOrdersIntegration verifies concurrent aggregation of multiple orders
func TestAggregatorMultipleOrdersIntegration(t *testing.T) {
	spec := &AggregateSpec{
		CorrelationKey:     "body.order_id",
		CompletionStrategy: "count",
		Count:              2,
		TimeoutMs:          5000,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Interleave line items from two orders
	messages := []*engine.Message{
		// ORD-001 item 1
		engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "sku": "A1", "qty": 1}, "route", "v1"),
		// ORD-002 item 1
		engine.NewMessage(map[string]interface{}{"order_id": "ORD-002", "sku": "B1", "qty": 2}, "route", "v1"),
		// ORD-001 item 2 (should complete ORD-001)
		engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "sku": "A2", "qty": 3}, "route", "v1"),
		// ORD-002 item 2 (should complete ORD-002)
		engine.NewMessage(map[string]interface{}{"order_id": "ORD-002", "sku": "B2", "qty": 4}, "route", "v1"),
	}

	completedOrders := []*engine.Message{}

	for _, msg := range messages {
		out, err := step.Execute(ctx, msg)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if out != nil {
			completedOrders = append(completedOrders, out)
		}
	}

	// Should have 2 completed aggregations
	if len(completedOrders) != 2 {
		t.Errorf("Expected 2 completed aggregations, got %d", len(completedOrders))
	}

	// Verify both orders completed
	orders := make(map[string]*engine.Message)
	for _, out := range completedOrders {
		body := out.Body.(map[string]interface{})
		orderID := body["correlation_key"].(string)
		orders[orderID] = out
	}

	if _, ok := orders["ORD-001"]; !ok {
		t.Errorf("Expected ORD-001 in completed orders")
	}
	if _, ok := orders["ORD-002"]; !ok {
		t.Errorf("Expected ORD-002 in completed orders")
	}

	// Verify groups cleared
	step.mu.RLock()
	groupCount := len(step.groups)
	step.mu.RUnlock()
	if groupCount != 0 {
		t.Errorf("Expected 0 groups after completion, got %d", groupCount)
	}
}

// TestAggregatorTimeoutCompletion verifies timeout-based flush (M2.1.3)
func TestAggregatorTimeoutCompletion(t *testing.T) {
	spec := &AggregateSpec{
		CorrelationKey:     "body.order_id",
		CompletionStrategy: "count",
		Count:              10, // High count to ensure timeout triggers
		TimeoutMs:          100,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Add one message
	msg := engine.NewMessage(map[string]interface{}{"order_id": "ORD-001", "item": "X"}, "route", "v1")
	out, err := step.Execute(ctx, msg)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if out != nil {
		t.Errorf("Expected no output immediately")
	}

	// Wait for timeout to occur
	time.Sleep(200 * time.Millisecond)

	// Verify group was flushed
	step.mu.RLock()
	groupCount := len(step.groups)
	step.mu.RUnlock()
	if groupCount != 0 {
		t.Errorf("Expected group to be flushed after timeout, %d remain", groupCount)
	}
}

// TestAggregatorWithComplexMessages verifies aggregation of complex message structures
func TestAggregatorWithComplexMessages(t *testing.T) {
	spec := &AggregateSpec{
		CorrelationKey:     "body.metadata.order_id",
		CompletionStrategy: "count",
		Count:              2,
		TimeoutMs:          5000,
	}

	step, err := NewAggregateStep(spec)
	if err != nil {
		t.Fatalf("Failed to create aggregator: %v", err)
	}

	ctx := context.Background()

	// Create messages with nested structures
	messages := []*engine.Message{
		engine.NewMessage(
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"order_id": "ORD-999",
					"timestamp": "2026-09-04T12:00:00Z",
				},
				"line_item": map[string]interface{}{
					"sku": "PROD-1",
					"qty": 5,
				},
			},
			"route", "v1",
		),
		engine.NewMessage(
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"order_id": "ORD-999",
					"timestamp": "2026-09-04T12:00:01Z",
				},
				"line_item": map[string]interface{}{
					"sku": "PROD-2",
					"qty": 3,
				},
			},
			"route", "v1",
		),
	}

	for i, msg := range messages {
		out, err := step.Execute(ctx, msg)
		if err != nil {
			t.Fatalf("Message %d failed: %v", i+1, err)
		}

		if out != nil {
			// Verify output
			body := out.Body.(map[string]interface{})
			if key, ok := body["correlation_key"].(string); !ok || key != "ORD-999" {
				t.Errorf("Expected correlation_key='ORD-999', got %v", body["correlation_key"])
			}
		}
	}
}
