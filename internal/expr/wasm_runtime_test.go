package expr

import (
	"context"
	"testing"
)

// TestWASMRuntimeInit tests basic WASM runtime initialization.
func TestWASMRuntimeInit(t *testing.T) {
	ctx := context.Background()

	// Create a WASM runtime
	runtime, err := NewWASMRuntime(ctx)
	if err != nil {
		t.Fatalf("failed to create WASM runtime: %v", err)
	}

	// Verify runtime is not nil
	if runtime == nil {
		t.Fatal("runtime is nil")
	}

	// Verify modules map is initialized
	if runtime.modules == nil {
		t.Fatal("modules map is nil")
	}

	// Close the runtime
	err = runtime.Close(ctx)
	if err != nil {
		t.Fatalf("failed to close runtime: %v", err)
	}
}

// TestWASMFunctionInRegistry tests registering and looking up WASM functions.
func TestWASMFunctionInRegistry(t *testing.T) {
	ctx := context.Background()

	runtime, err := NewWASMRuntime(ctx)
	if err != nil {
		t.Fatalf("failed to create WASM runtime: %v", err)
	}
	defer runtime.Close(ctx)

	// Simulate registering a WASM function in the registry
	registry := NewRegistry()

	def := &FunctionDef{
		Name:    "wasmAdd",
		Type:    FunctionTypeWasm,
		Runtime: "wasm",
		Ref:     "plugins/add.wasm",
	}

	// Create a WASM function wrapper (simulated - no actual WASM module)
	impl := runtime.WASMFunction(ctx, def.Ref, "add")

	err = registry.Register(def, impl)
	if err != nil {
		t.Fatalf("failed to register WASM function: %v", err)
	}

	// Verify the function is registered
	if registry.Count() != 1 {
		t.Errorf("expected 1 function, got %d", registry.Count())
	}

	// Look up the function
	retrievedDef, retrievedImpl, found := registry.Lookup("wasmAdd")
	if !found {
		t.Fatal("WASM function not found in registry")
	}

	if retrievedDef.Type != FunctionTypeWasm {
		t.Errorf("expected type 'wasm', got %q", retrievedDef.Type)
	}

	if retrievedImpl == nil {
		t.Fatal("function implementation is nil")
	}
}
