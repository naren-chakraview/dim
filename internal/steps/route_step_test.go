package steps

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestNewRouteStep verifies that a valid route step is created successfully
func TestNewRouteStep(t *testing.T) {
	cases := map[string]string{
		"premium":  "sink-premium",
		"standard": "sink-standard",
	}

	step, err := NewRouteStep("body.type", cases, "sink-default")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	if step == nil {
		t.Error("NewRouteStep returned nil step")
	}
}

// TestNewRouteStepEmptyExpression verifies that empty expression is rejected
func TestNewRouteStepEmptyExpression(t *testing.T) {
	cases := map[string]string{
		"premium": "sink-premium",
	}

	_, err := NewRouteStep("", cases, "sink-default")
	if err == nil {
		t.Error("Expected error for empty expression, got nil")
	}
}

// TestNewRouteStepEmptyCases verifies that empty cases map is rejected
func TestNewRouteStepEmptyCases(t *testing.T) {
	_, err := NewRouteStep("body.type", map[string]string{}, "sink-default")
	if err == nil {
		t.Error("Expected error for empty cases map, got nil")
	}
}

// TestNewRouteStepInvalidExpression verifies that invalid expressions fail at compile time
func TestNewRouteStepInvalidExpression(t *testing.T) {
	cases := map[string]string{
		"test": "sink-test",
	}

	_, err := NewRouteStep("{ invalid json ] }", cases, "sink-default")
	if err == nil {
		t.Error("Expected compile error for invalid expression, got nil")
	}
}

// TestRouteStepCaseMatch verifies that expressions matching a case route to the correct target
func TestRouteStepCaseMatch(t *testing.T) {
	ctx := context.Background()
	cases := map[string]string{
		"premium":  "sink-premium",
		"standard": "sink-standard",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "premium"},
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

	if result.Metadata.Route != "sink-premium" {
		t.Errorf("Expected Route 'sink-premium', got %s", result.Metadata.Route)
	}
}

// TestRouteStepComplexExpression verifies routing with complex JSONata expressions
func TestRouteStepComplexExpression(t *testing.T) {
	ctx := context.Background()

	// Map results to different sinks based on conditions
	cases := map[string]string{
		"high":   "sink-high",
		"medium": "sink-medium",
		"low":    "sink-low",
	}

	// Expression: evaluate amount and return priority level
	step, err := NewRouteStep(
		`body.amount > 1000 ? "high" : body.amount > 100 ? "medium" : "low"`,
		cases,
		"",
	)
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	tests := []struct {
		name      string
		amount    float64
		expected  string
	}{
		{"high amount", 2000.0, "sink-high"},
		{"medium amount", 500.0, "sink-medium"},
		{"low amount", 50.0, "sink-low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := engine.NewMessage(
				map[string]interface{}{"amount": tt.amount},
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

			if result.Metadata.Route != tt.expected {
				t.Errorf("Expected Route %q, got %q", tt.expected, result.Metadata.Route)
			}
		})
	}
}

// TestRouteStepNestedFieldAccess verifies routing with nested field access
func TestRouteStepNestedFieldAccess(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"us":   "sink-us",
		"eu":   "sink-eu",
		"asia": "sink-asia",
	}

	step, err := NewRouteStep("body.customer.region", cases, "sink-default")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{
			"customer": map[string]interface{}{
				"region": "eu",
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

	if result.Metadata.Route != "sink-eu" {
		t.Errorf("Expected Route 'sink-eu', got %s", result.Metadata.Route)
	}
}

// TestRouteStepMissingCaseUsesDefault verifies that missing cases use the default target
func TestRouteStepMissingCaseUsesDefault(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"premium": "sink-premium",
	}

	step, err := NewRouteStep("body.type", cases, "sink-default")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "unknown"},
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

	if result.Metadata.Route != "sink-default" {
		t.Errorf("Expected Route 'sink-default', got %s", result.Metadata.Route)
	}
}

// TestRouteStepMissingCaseNoDefault verifies error when no case matches and no default
func TestRouteStepMissingCaseNoDefault(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"premium": "sink-premium",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "unknown"},
		"test-route",
		"v1",
	)

	result, err := step.Execute(ctx, msg)
	if err == nil {
		t.Fatal("Expected error for no matching case and no default, got nil")
	}

	if result != nil {
		t.Error("Expected result to be nil when routing fails")
	}

	// Verify error message contains case name
	if err != nil && len(err.Error()) == 0 {
		t.Error("Expected non-empty error message")
	}
}

// TestRouteStepMetadataRouteUpdated verifies that metadata.Route is updated
func TestRouteStepMetadataRouteUpdated(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"premium":  "sink-premium",
		"standard": "sink-standard",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "premium"},
		"original-route",
		"v1",
	)

	// Verify original route
	if msg.Metadata.Route != "original-route" {
		t.Errorf("Original route should be 'original-route', got %s", msg.Metadata.Route)
	}

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// Verify route is updated
	if result.Metadata.Route != "sink-premium" {
		t.Errorf("Route not updated: got %s, want sink-premium", result.Metadata.Route)
	}

	// Verify original message is unchanged (immutability)
	if msg.Metadata.Route != "original-route" {
		t.Error("Original message should not be modified")
	}
}

// TestRouteStepHeadersPreserved verifies that headers are preserved
func TestRouteStepHeadersPreserved(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"test": "sink-test",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "test"},
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
		t.Error("Expected Content-Type header to be preserved")
	}

	if result.Headers["X-Custom-Header"] != "custom-value" {
		t.Error("Expected X-Custom-Header to be preserved")
	}
}

// TestRouteStepBodyPreserved verifies that body is preserved
func TestRouteStepBodyPreserved(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"test": "sink-test",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	originalBody := map[string]interface{}{
		"type": "test",
		"data": "important",
	}

	msg := engine.NewMessage(originalBody, "test-route", "v1")

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	// Verify body is preserved
	resultBody := result.Body.(map[string]interface{})
	if resultBody["type"] != "test" {
		t.Errorf("Body.type not preserved: got %v", resultBody["type"])
	}

	if resultBody["data"] != "important" {
		t.Errorf("Body.data not preserved: got %v", resultBody["data"])
	}
}

// TestRouteStepNumericCaseNames verifies routing with numeric case names
func TestRouteStepNumericCaseNames(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"1": "sink-one",
		"2": "sink-two",
		"3": "sink-three",
	}

	step, err := NewRouteStep("body.priority", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"priority": 2.0},
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

	if result.Metadata.Route != "sink-two" {
		t.Errorf("Expected Route 'sink-two', got %s", result.Metadata.Route)
	}
}

// TestRouteStepHeadersInExpression verifies routing with headers in expression
func TestRouteStepHeadersInExpression(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"api": "sink-api",
		"web": "sink-web",
	}

	step, err := NewRouteStep("headers.client_type", cases, "sink-default")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"data": "test"},
		"test-route",
		"v1",
	)
	msg.Headers["client_type"] = "api"

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	if result.Metadata.Route != "sink-api" {
		t.Errorf("Expected Route 'sink-api', got %s", result.Metadata.Route)
	}
}

// TestRouteStepNilMessage verifies error handling for nil message
func TestRouteStepNilMessage(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"test": "sink-test",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	result, err := step.Execute(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil message")
	}

	if result != nil {
		t.Error("Expected result to be nil for nil message")
	}
}

// TestRouteStepEvaluationError verifies error handling for expression evaluation failures
func TestRouteStepEvaluationError(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"test": "sink-test",
	}

	// Use a function that will fail during evaluation
	step, err := NewRouteStep("$nonexistent_function(body)", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
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

// TestRouteStepCaseSensitivity verifies that case matching is case-sensitive
func TestRouteStepCaseSensitivity(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"Premium": "sink-premium",
	}

	step, err := NewRouteStep("body.type", cases, "sink-default")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "premium"},
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

	// Should use default since "premium" != "Premium"
	if result.Metadata.Route != "sink-default" {
		t.Errorf("Expected Route 'sink-default' (case-sensitive match), got %s", result.Metadata.Route)
	}
}

// TestRouteStepBooleanCaseNames verifies routing with boolean case names
func TestRouteStepBooleanCaseNames(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"true":  "sink-active",
		"false": "sink-inactive",
	}

	step, err := NewRouteStep("body.active", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"active": true},
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

	if result.Metadata.Route != "sink-active" {
		t.Errorf("Expected Route 'sink-active', got %s", result.Metadata.Route)
	}
}

// TestRouteStepMultipleCases verifies routing with many cases
func TestRouteStepMultipleCases(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"category_a": "sink-a",
		"category_b": "sink-b",
		"category_c": "sink-c",
		"category_d": "sink-d",
		"category_e": "sink-e",
	}

	step, err := NewRouteStep("body.category", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"category": "category_d"},
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

	if result.Metadata.Route != "sink-d" {
		t.Errorf("Expected Route 'sink-d', got %s", result.Metadata.Route)
	}
}

// TestRouteStepExpressionWithFunctions verifies routing with JSONata functions
func TestRouteStepExpressionWithFunctions(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"short": "sink-short",
		"long":  "sink-long",
	}

	// Expression: check length of a string field
	step, err := NewRouteStep(
		`$length(body.name) > 10 ? "long" : "short"`,
		cases,
		"",
	)
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"short name", "John", "sink-short"},
		{"long name", "Alexander Hamilton", "sink-long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := engine.NewMessage(
				map[string]interface{}{"name": tt.value},
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

			if result.Metadata.Route != tt.expected {
				t.Errorf("Expected Route %q, got %q", tt.expected, result.Metadata.Route)
			}
		})
	}
}

// TestRouteStepCorrelationIDPreserved verifies that correlation ID is preserved
func TestRouteStepCorrelationIDPreserved(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"test": "sink-test",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "test"},
		"test-route",
		"v1",
	)
	originalCorrID := msg.Metadata.CorrelationID

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	if result.Metadata.CorrelationID != originalCorrID {
		t.Error("CorrelationID not preserved through routing")
	}
}

// TestRouteStepIngestedAtPreserved verifies that ingest timestamp is preserved
func TestRouteStepIngestedAtPreserved(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"test": "sink-test",
	}

	step, err := NewRouteStep("body.type", cases, "")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"type": "test"},
		"test-route",
		"v1",
	)
	originalIngestedAt := msg.Metadata.IngestedAt

	result, err := step.Execute(ctx, msg)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result message is nil")
	}

	if result.Metadata.IngestedAt != originalIngestedAt {
		t.Error("IngestedAt not preserved through routing")
	}
}

// TestRouteStepDefaultWithUnmatchedValue verifies default routing when expression result doesn't match any case
func TestRouteStepDefaultWithUnmatchedValue(t *testing.T) {
	ctx := context.Background()

	cases := map[string]string{
		"premium": "sink-premium",
	}

	// This expression will return "standard" which isn't in cases
	step, err := NewRouteStep("body.tier", cases, "sink-default")
	if err != nil {
		t.Fatalf("NewRouteStep failed: %v", err)
	}

	msg := engine.NewMessage(
		map[string]interface{}{"tier": "standard"},
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

	// "standard" case doesn't exist, should use default
	if result.Metadata.Route != "sink-default" {
		t.Errorf("Expected Route 'sink-default', got %s", result.Metadata.Route)
	}
}
