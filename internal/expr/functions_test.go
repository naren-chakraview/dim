package expr

import (
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	// Define a simple test function
	def := &FunctionDef{
		Name:    "double",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugins/double",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return nil, nil
		}
		switch v := args[0].(type) {
		case float64:
			return v * 2, nil
		case int:
			return v * 2, nil
		default:
			return nil, nil
		}
	}

	// Register the function
	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("failed to register function: %v", err)
	}

	// Verify it was registered
	if registry.Count() != 1 {
		t.Errorf("expected 1 function, got %d", registry.Count())
	}

	// Lookup the function
	retrievedDef, retrievedImpl, found := registry.Lookup("double")
	if !found {
		t.Fatal("function not found in registry")
	}
	if retrievedDef.Name != "double" {
		t.Errorf("expected function name 'double', got %q", retrievedDef.Name)
	}
	if retrievedImpl == nil {
		t.Fatal("function implementation is nil")
	}

	// Test function execution
	result, err := retrievedImpl(5)
	if err != nil {
		t.Fatalf("function execution failed: %v", err)
	}
	if result != 10 {
		t.Errorf("expected 10, got %v", result)
	}
}

func TestRegistry_Duplicate(t *testing.T) {
	registry := NewRegistry()

	def := &FunctionDef{
		Name:    "test",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugins/test",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		return "ok", nil
	}

	// First registration should succeed
	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration with same name should fail
	err = registry.Register(def, impl)
	if err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestEvaluatorWithRegistry(t *testing.T) {
	// Create a registry and register a custom function
	registry := NewRegistry()

	def := &FunctionDef{
		Name:    "addTen",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugins/addTen",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return 0, nil
		}
		switch v := args[0].(type) {
		case float64:
			return v + 10, nil
		case int:
			return v + 10, nil
		default:
			return nil, nil
		}
	}

	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("failed to register function: %v", err)
	}

	// Compile an expression with the registry
	exprStr := `body.value`
	evaluator, err := CompileExpressionWithRegistry(exprStr, registry)
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// Verify the registry is accessible from the evaluator
	retrievedRegistry := evaluator.GetRegistry()
	if retrievedRegistry == nil {
		t.Fatal("registry not accessible from evaluator")
	}

	// Verify the function is in the registry
	retrievedDef, retrievedImpl, found := retrievedRegistry.Lookup("addTen")
	if !found {
		t.Fatal("function not found in registry via evaluator")
	}
	if retrievedDef.Name != "addTen" {
		t.Errorf("expected function name 'addTen', got %q", retrievedDef.Name)
	}

	// Test that the function can be called directly from the registry
	result, err := retrievedImpl(5)
	if err != nil {
		t.Fatalf("function call failed: %v", err)
	}
	if result != 15 {
		t.Errorf("expected 15, got %v", result)
	}

	// Verify the evaluator still works for standard expressions
	input := map[string]interface{}{
		"body": map[string]interface{}{
			"value": 42,
		},
	}
	evalResult, err := evaluator.Eval(input)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}
	// JSONata may return int or float depending on context
	if evalResult != 42.0 && evalResult != 42 {
		t.Errorf("expected 42 or 42.0, got %v", evalResult)
	}
}
