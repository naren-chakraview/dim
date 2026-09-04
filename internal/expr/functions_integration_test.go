package expr

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestFunctionsRegistryIntegration demonstrates how a translate step would use
// the functions registry to call a registered custom function.
// This is a simplified version without full JSONata function binding (that's part of R4/R5).
func TestFunctionsRegistryIntegration(t *testing.T) {
	// Create a registry with a custom function
	registry := NewRegistry()

	def := &FunctionDef{
		Name:    "formatPhone",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugins/formatPhone",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return "", nil
		}
		if phone, ok := args[0].(string); ok {
			// Simple formatting: add hyphens
			if len(phone) == 10 {
				return phone[0:3] + "-" + phone[3:6] + "-" + phone[6:10], nil
			}
		}
		return args[0], nil
	}

	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("failed to register function: %v", err)
	}

	// Create a translate step that would want to use the custom function
	// For now, we can directly call the function from the registry
	evaluator, err := CompileExpressionWithRegistry("body.phone", registry)
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// Verify the registry is accessible
	reg := evaluator.GetRegistry()
	if reg == nil {
		t.Fatal("registry not accessible")
	}

	// Get the custom function and call it directly
	_, formatPhoneImpl, found := reg.Lookup("formatPhone")
	if !found {
		t.Fatal("formatPhone function not found")
	}

	// Simulate what a step would do: extract the value, pass it to the custom function
	input := map[string]interface{}{
		"body": map[string]interface{}{
			"phone": "5551234567",
		},
	}

	// Evaluate to get the phone number
	phoneValue, err := evaluator.Eval(input)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Call the custom function with the extracted value
	formatted, err := formatPhoneImpl(phoneValue)
	if err != nil {
		t.Fatalf("function call failed: %v", err)
	}

	expectedFormatted := "555-123-4567"
	if formatted != expectedFormatted {
		t.Errorf("expected %q, got %q", expectedFormatted, formatted)
	}
}

// TestCustomFunctionInRoute demonstrates registering and using custom functions
// in the context of a full message processing pipeline.
func TestCustomFunctionWithMessage(t *testing.T) {
	registry := NewRegistry()

	// Register a function that doubles a number
	def := &FunctionDef{
		Name:    "doubleAmount",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugins/doubleAmount",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return 0, nil
		}
		switch v := args[0].(type) {
		case float64:
			return v * 2, nil
		case int:
			return v * 2, nil
		default:
			return v, nil
		}
	}

	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("failed to register function: %v", err)
	}

	// Verify the function can be called for message processing
	ctx := context.Background()

	// Create a message
	msg := &engine.Message{
		Body: map[string]interface{}{
			"amount": 100,
		},
		Headers: map[string]interface{}{},
		Metadata: engine.Metadata{
			CorrelationID: "msg-123",
		},
	}

	// Get the function from the registry
	_, doubleImpl, found := registry.Lookup("doubleAmount")
	if !found {
		t.Fatal("doubleAmount function not found")
	}

	// Extract the amount and process it through the custom function
	amount := msg.Body.(map[string]interface{})["amount"]
	doubled, err := doubleImpl(amount)
	if err != nil {
		t.Fatalf("function call failed: %v", err)
	}

	expectedDoubled := 200
	if doubled != expectedDoubled {
		t.Errorf("expected %v, got %v", expectedDoubled, doubled)
	}

	_ = ctx // silence unused variable warning
}
