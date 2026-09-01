package steps

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestNewTranslateStep verifies that a valid JSONata expression compiles correctly
func TestNewTranslateStep(t *testing.T) {
	expr := "{ \"order_id\": body.id, \"total\": body.amount / 100 }"
	step, err := NewTranslateStep(expr)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}
	if step == nil {
		t.Error("NewTranslateStep returned nil step")
	}
}

// TestNewTranslateStepInvalidExpression verifies that invalid expressions fail at compile time
func TestNewTranslateStepInvalidExpression(t *testing.T) {
	// Invalid JSONata syntax
	expr := "{ invalid json ] }"
	step, err := NewTranslateStep(expr)
	if err == nil {
		t.Error("Expected compile error for invalid expression, got nil")
	}
	if step != nil {
		t.Error("Expected step to be nil when compilation fails")
	}
}

// TestTranslateSimpleFieldAccess tests accessing and transforming a simple field
func TestTranslateSimpleFieldAccess(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep("body.amount / 100")
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "o1",
			"amount": float64(1500),
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// The result should be 15.0
	if result.Body.(float64) != 15.0 {
		t.Errorf("Expected 15.0, got %v", result.Body)
	}
}

// TestTranslateObjectConstruction tests constructing a new object from message fields
func TestTranslateObjectConstruction(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`{ "order_id": body.id, "total": body.amount / 100 }`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "o1",
			"amount": float64(1500),
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// Verify the body is the transformed object
	body, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result.Body)
	}

	if body["order_id"] != "o1" {
		t.Errorf("Expected order_id 'o1', got %v", body["order_id"])
	}

	if body["total"] != 15.0 {
		t.Errorf("Expected total 15.0, got %v", body["total"])
	}
}

// TestTranslateSelectiveFieldExtraction tests picking specific fields while ignoring others
func TestTranslateSelectiveFieldExtraction(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`{ "id": body.id, "amount": body.amount }`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":    "o2",
			"amount": float64(500),
			"extra": "ignored",
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	body, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result.Body)
	}

	if body["id"] != "o2" {
		t.Errorf("Expected id 'o2', got %v", body["id"])
	}

	if body["amount"] != 500.0 {
		t.Errorf("Expected amount 500.0, got %v", body["amount"])
	}

	if _, exists := body["extra"]; exists {
		t.Error("Expected 'extra' field to be absent, but it exists")
	}
}

// TestTranslateHeadersPreserved verifies that headers are copied to the output message
func TestTranslateHeadersPreserved(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep("{ \"transformed\": true }")
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"original": "body"},
		"test-route",
		"v1",
	)
	msg.Headers["Content-Type"] = "application/json"
	msg.Headers["X-Custom-Header"] = "custom-value"

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	if result.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header to be preserved")
	}

	if result.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Expected X-Custom-Header to be preserved")
	}
}

// TestTranslateMetadataPreserved verifies that metadata is preserved
func TestTranslateMetadataPreserved(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep("body.value * 2")
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"value": 10},
		"test-route",
		"v1",
	)
	originalMetadata := msg.Metadata

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	if result.Metadata.Route != originalMetadata.Route {
		t.Errorf("Route not preserved: got %s, want %s", result.Metadata.Route, originalMetadata.Route)
	}

	if result.Metadata.CorrelationID != originalMetadata.CorrelationID {
		t.Errorf("CorrelationID not preserved")
	}

	if result.Metadata.IngestedAt != originalMetadata.IngestedAt {
		t.Errorf("IngestedAt not preserved")
	}
}

// TestTranslateArrayConstruction tests constructing arrays
func TestTranslateArrayConstruction(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`[body.id, body.amount, body.status]`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "order-123",
			"amount": 99.99,
			"status": "pending",
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	arr, ok := result.Body.([]interface{})
	if !ok {
		t.Fatalf("Expected array, got %T", result.Body)
	}

	if len(arr) != 3 {
		t.Errorf("Expected array of length 3, got %d", len(arr))
	}

	if arr[0] != "order-123" {
		t.Errorf("Expected first element 'order-123', got %v", arr[0])
	}

	if arr[1] != 99.99 {
		t.Errorf("Expected second element 99.99, got %v", arr[1])
	}

	if arr[2] != "pending" {
		t.Errorf("Expected third element 'pending', got %v", arr[2])
	}
}

// TestTranslateExpressionWithHeaders tests using header values in expression
func TestTranslateExpressionWithHeaders(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`{ "order_id": body.id, "user": headers.user_id }`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"id": "o1"},
		"test-route",
		"v1",
	)
	msg.Headers["user_id"] = "user-42"

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	body, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result.Body)
	}

	if body["user"] != "user-42" {
		t.Errorf("Expected user 'user-42', got %v", body["user"])
	}
}

// TestTranslateNilMessage tests handling of nil message
func TestTranslateNilMessage(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep("body.value")
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	result, err := step.Execute(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil message")
	}

	if result != nil {
		t.Error("Expected result to be nil for nil message")
	}
}

// TestTranslateEvaluationError tests handling of expression evaluation failures
func TestTranslateEvaluationError(t *testing.T) {
	ctx := context.Background()
	// This expression tries to call a non-existent function, which should error
	step, err := NewTranslateStep("$nonexistent_function(body)")
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"value": 42},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	// This should fail during evaluation
	if err == nil {
		t.Fatal("Expected evaluation error for undefined function")
	}

	if result != nil {
		t.Error("Expected result to be nil when evaluation fails")
	}
}

// TestTranslateComplexExpression tests a more complex JSONata expression
func TestTranslateComplexExpression(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`
		{
			"order": body.id,
			"total": $sum(body.items.(price * quantity)),
			"item_count": $count(body.items),
			"status": body.status = "paid" ? "processing" : "pending_payment"
		}
	`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "order-1",
			"status": "paid",
			"items": []interface{}{
				map[string]interface{}{"price": 10.0, "quantity": 2},
				map[string]interface{}{"price": 5.0, "quantity": 3},
			},
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	body, ok := result.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result.Body)
	}

	if body["order"] != "order-1" {
		t.Errorf("Expected order 'order-1', got %v", body["order"])
	}

	// total should be 10*2 + 5*3 = 35
	if body["total"] != 35.0 {
		t.Errorf("Expected total 35.0, got %v", body["total"])
	}

	// $count returns an integer (or float64 if it went through JSON)
	itemCount := body["item_count"]
	if itemCount != 2 && itemCount != 2.0 {
		t.Errorf("Expected item_count 2 or 2.0, got %v (%T)", itemCount, itemCount)
	}

	if body["status"] != "processing" {
		t.Errorf("Expected status 'processing', got %v", body["status"])
	}
}

// TestTranslateStringResult tests when the result is a string
func TestTranslateStringResult(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`"Order " & body.id & " is " & body.status`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "o1",
			"status": "shipped",
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	if result.Body != "Order o1 is shipped" {
		t.Errorf("Expected 'Order o1 is shipped', got %v", result.Body)
	}
}

// TestTranslateNumberResult tests when the result is a number
func TestTranslateNumberResult(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep("body.price * body.quantity * (1 + body.tax_rate)")
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"price":    100.0,
			"quantity": 5.0,
			"tax_rate": 0.1,
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// 100 * 5 * 1.1 = 550
	if result.Body != 550.0 {
		t.Errorf("Expected 550.0, got %v", result.Body)
	}
}

// TestTranslateBooleanResult tests when the result is a boolean
func TestTranslateBooleanResult(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep("body.amount > 1000")
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	testCases := []struct {
		name     string
		amount   float64
		expected bool
	}{
		{"above threshold", 1500.0, true},
		{"below threshold", 500.0, false},
		{"at threshold", 1000.0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := engine.NewMessage(
				map[string]interface{}{"amount": tc.amount},
				"test-route",
				"v1",
			)

			result, err := step.Execute(ctx, msg)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if result == nil {
				t.Fatal("Result message is nil")
			}

			if result.Body != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result.Body)
			}
		})
	}
}

// TestTranslateResultsInDeepCopy tests that modifying result headers doesn't affect original
func TestTranslateResultDoesNotAffectOriginal(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`{ "value": body.value }`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"value": 100},
		"test-route",
		"v1",
	)
	msg.Headers["test"] = "original"

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Modify the result headers
	result.Headers["test"] = "modified"

	// Original message headers should not be affected
	if msg.Headers["test"] != "original" {
		t.Error("Modifying result headers affected the original message")
	}
}

// TestTranslateWithJSONMarshalRoundtrip tests that results can be marshaled/unmarshaled
func TestTranslateWithJSONMarshalRoundtrip(t *testing.T) {
	ctx := context.Background()
	step, err := NewTranslateStep(`{ "order_id": body.id, "total": body.amount / 100 }`)
	if err != nil {
		t.Fatalf("NewTranslateStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"id":     "o1",
			"amount": 1500.0,
		},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should be able to marshal the result
	data, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	// Should be able to unmarshal
	var unmarshaled map[string]interface{}
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if unmarshaled["order_id"] != "o1" {
		t.Errorf("Roundtrip failed: order_id mismatch")
	}

	if unmarshaled["total"] != 15.0 {
		t.Errorf("Roundtrip failed: total mismatch")
	}
}
