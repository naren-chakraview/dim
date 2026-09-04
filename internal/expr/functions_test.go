package expr

import (
	"testing"
)

// TestRegistryBasics verifies registration and lookup of custom functions
func TestRegistryBasics(t *testing.T) {
	registry := NewRegistry()

	// Register a simple function
	def := &FunctionDef{
		Name:    "double",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugin://builtin/double",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		if len(args) == 0 {
			return nil, nil
		}
		// Simple doubler for testing
		if val, ok := args[0].(float64); ok {
			return val * 2, nil
		}
		return nil, nil
	}

	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Lookup should find it
	foundDef, foundImpl, exists := registry.Lookup("double")
	if !exists {
		t.Fatal("Lookup failed: function not found")
	}
	if foundDef.Name != "double" {
		t.Errorf("Definition mismatch: got %s, want double", foundDef.Name)
	}
	if foundImpl == nil {
		t.Fatal("Implementation is nil")
	}

	// Call the function
	result, err := foundImpl(5.0)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
	if result.(float64) != 10.0 {
		t.Errorf("Result mismatch: got %v, want 10.0", result)
	}
}

// TestRegistryDuplicates verifies duplicate registration is rejected
func TestRegistryDuplicates(t *testing.T) {
	registry := NewRegistry()

	def := &FunctionDef{
		Name:    "test",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugin://test",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		return nil, nil
	}

	// First registration should succeed
	if err := registry.Register(def, impl); err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	// Duplicate should fail
	if err := registry.Register(def, impl); err == nil {
		t.Fatal("Duplicate register should have failed")
	}
}

// TestRegistryGetAll verifies enumeration of registered functions
func TestRegistryGetAll(t *testing.T) {
	registry := NewRegistry()

	// Register multiple functions
	funcs := map[string]FunctionImpl{
		"add": func(args ...interface{}) (interface{}, error) {
			return nil, nil
		},
		"subtract": func(args ...interface{}) (interface{}, error) {
			return nil, nil
		},
	}

	for name, impl := range funcs {
		def := &FunctionDef{
			Name:    name,
			Type:    FunctionTypePlugin,
			Runtime: "go",
			Ref:     "plugin://builtin/" + name,
		}
		if err := registry.Register(def, impl); err != nil {
			t.Fatalf("Register %s failed: %v", name, err)
		}
	}

	// GetAll should return both
	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll returned %d functions, want 2", len(all))
	}

	for name := range funcs {
		if _, exists := all[name]; !exists {
			t.Errorf("Function %s not found in GetAll", name)
		}
	}
}

// TestRegistryCount verifies function counting
func TestRegistryCount(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Errorf("Initial count: got %d, want 0", registry.Count())
	}

	// Register one
	def := &FunctionDef{
		Name:    "test1",
		Type:    FunctionTypeWasm,
		Runtime: "wasm",
		Ref:     "module://test.wasm",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		return nil, nil
	}
	registry.Register(def, impl)

	if registry.Count() != 1 {
		t.Errorf("After register: got %d, want 1", registry.Count())
	}
}

// TestEvaluatorWithRegistry verifies Evaluator can access registry
func TestEvaluatorWithRegistry(t *testing.T) {
	registry := NewRegistry()

	// Register a function
	def := &FunctionDef{
		Name:    "myFunc",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugin://custom",
	}
	impl := func(args ...interface{}) (interface{}, error) {
		return "called", nil
	}
	registry.Register(def, impl)

	// Create evaluator with registry
	eval, err := CompileExpressionWithRegistry("1 + 1", registry)
	if err != nil {
		t.Fatalf("CompileExpressionWithRegistry failed: %v", err)
	}

	// Verify registry is accessible
	if eval.GetRegistry() == nil {
		t.Fatal("Registry is nil after CompileExpressionWithRegistry")
	}

	// Verify we can look up the function through the evaluator's registry
	_,foundImpl,exists:=eval.GetRegistry().Lookup("myFunc")
	if !exists {
		t.Fatal("Function not found through evaluator registry")
	}

	result, err := foundImpl()
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
	if result != "called" {
		t.Errorf("Result mismatch: got %v, want called", result)
	}

	// The JSONata expression itself still works (no direct function integration yet)
	evalResult, err := eval.Eval(nil)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if evalResult != float64(2) {
		t.Errorf("JSONata eval result: got %v, want 2", evalResult)
	}
}
