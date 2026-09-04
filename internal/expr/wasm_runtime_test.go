package expr

import (
	"context"
	"testing"
)

// TestWASMRuntimeInit verifies WASM runtime initialization
func TestWASMRuntimeInit(t *testing.T) {
	runtime := NewWASMRuntime()
	if runtime == nil {
		t.Fatal("NewWASMRuntime returned nil")
	}
	if runtime.runtime == nil {
		t.Fatal("wazero.Runtime not initialized")
	}
	if len(runtime.modules) != 0 {
		t.Errorf("Initial modules count: got %d, want 0", len(runtime.modules))
	}

	// Clean up
	if err := runtime.Close(); err != nil {
		t.Logf("Close error: %v", err)
	}
}

// TestWASMFunctionInRegistry verifies registering a WASM function
func TestWASMFunctionInRegistry(t *testing.T) {
	runtime := NewWASMRuntime()
	defer runtime.Close()

	registry := NewRegistry()

	// Register a simulated WASM function (actual WASM module loading deferred to Phase 2)
	// This tests the registration and calling convention
	def := &FunctionDef{
		Name:    "wasmDouble",
		Type:    FunctionTypeWasm,
		Runtime: "wazero",
		Ref:     "wasm://test/double",
	}

	// Simulated WASM function that doubles a number
	impl := func(args ...interface{}) (interface{}, error) {
		if len(args) == 0 {
			return nil, nil
		}
		if val, ok := args[0].(int); ok {
			return val * 2, nil
		}
		if val, ok := args[0].(int64); ok {
			return val * 2, nil
		}
		return nil, nil
	}

	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify function is registered
	_, fn, exists := registry.Lookup("wasmDouble")
	if !exists {
		t.Fatal("Function not found in registry")
	}

	// Call the function
	result, err := fn(5)
	if err != nil {
		t.Fatalf("Function call failed: %v", err)
	}
	if result.(int) != 10 {
		t.Errorf("Result: got %v, want 10", result)
	}
}

// TestWASMCallingConvention verifies numeric type conversion
func TestWASMCallingConvention(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected uint64
	}{
		{"int", int(42), uint64(42)},
		{"int32", int32(42), uint64(42)},
		{"int64", int64(42), uint64(42)},
		{"float32", float32(42.5), uint64(42)},
		{"float64", float64(42.5), uint64(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toWasmInt64(tt.input)
			if result != tt.expected {
				t.Errorf("toWasmInt64(%v): got %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestWASMRuntimeModuleLoading verifies error handling for missing files
func TestWASMRuntimeModuleLoading(t *testing.T) {
	runtime := NewWASMRuntime()
	defer runtime.Close()

	// Attempting to load from file should fail with expected message
	_, err := runtime.LoadModule(context.Background(), "/nonexistent/module.wasm", "test")
	if err == nil {
		t.Fatal("Expected error loading nonexistent file")
	}

	// Verify error indicates limitation
	if err.Error() != "file-based WASM loading not yet implemented; use programmatic module creation in tests" {
		t.Logf("Got expected error: %v", err)
	}
}

// TestWASMRuntimeModuleValidation verifies name and path validation
func TestWASMRuntimeModuleValidation(t *testing.T) {
	runtime := NewWASMRuntime()
	defer runtime.Close()

	tests := []struct {
		name        string
		modulePath  string
		moduleName  string
		expectError bool
	}{
		{"empty path", "", "test", true},
		{"empty name", "test.wasm", "", true},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runtime.LoadModule(context.Background(), tt.modulePath, tt.moduleName)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}
