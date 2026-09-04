package expr

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WASMCallConvention documents the calling convention for WASM functions.
// This spike settles the interface contract between Go and WASM functions.
//
// Calling Convention (WASM ABI Spike):
// - Arguments: Passed as i64/f64 values on the stack (wazero handles marshaling)
// - Return: Single value (i64/f64) or error code
// - Error signaling: Return -1 (or special code) to indicate error
// - Complex types: Passed by reference (memory pointer) for JSON data
// - String data: UTF-8 encoded, written to linear memory with pointer returned
//
// Example: func add(a i64, b i64) -> i64 { return a + b }
// Called as: result := wasmFunc(3, 5) -> returns 8
//
// For simplicity in Phase 1, we support:
// - Numeric arguments (int64, float64)
// - Simple returns (integers, floats)
// - Error handling via return codes or exceptions

// WASMRuntime manages WASM function execution via wazero
type WASMRuntime struct {
	runtime wazero.Runtime
	modules map[string]api.Module // Loaded WASM modules by ref
}

// NewWASMRuntime creates a new WASM runtime
func NewWASMRuntime(ctx context.Context) (*WASMRuntime, error) {
	runtime := wazero.NewRuntime(ctx)

	return &WASMRuntime{
		runtime: runtime,
		modules: make(map[string]api.Module),
	}, nil
}

// LoadModule loads a WASM module from a file
// ref is the path to the .wasm file
func (wr *WASMRuntime) LoadModule(ctx context.Context, ref string) error {
	// Read the WASM binary
	wasmBinary, err := os.ReadFile(ref)
	if err != nil {
		return fmt.Errorf("failed to read WASM file %q: %w", ref, err)
	}

	// Instantiate the module
	module, err := wr.runtime.Instantiate(ctx, wasmBinary)
	if err != nil {
		return fmt.Errorf("failed to instantiate WASM module %q: %w", ref, err)
	}

	wr.modules[ref] = module
	return nil
}

// CallFunction calls a WASM function by name
// Returns the result or an error if the function doesn't exist or fails
func (wr *WASMRuntime) CallFunction(ctx context.Context, ref string, funcName string, args ...interface{}) (interface{}, error) {
	module, exists := wr.modules[ref]
	if !exists {
		return nil, fmt.Errorf("module %q not loaded", ref)
	}

	// Get the function from the module
	fn := module.ExportedFunction(funcName)
	if fn == nil {
		return nil, fmt.Errorf("function %q not found in module %q", funcName, ref)
	}

	// Convert arguments to int64 (simplified - full version handles multiple types)
	wasmArgs := make([]uint64, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case int:
			wasmArgs[i] = uint64(v)
		case int64:
			wasmArgs[i] = uint64(v)
		case float64:
			wasmArgs[i] = uint64(int64(v))
		default:
			return nil, fmt.Errorf("unsupported argument type: %T", v)
		}
	}

	// Call the WASM function
	results, err := fn.Call(ctx, wasmArgs...)
	if err != nil {
		return nil, fmt.Errorf("WASM function call failed: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	// Convert result back (simplified)
	return int64(results[0]), nil
}

// Close closes the WASM runtime and releases resources
func (wr *WASMRuntime) Close(ctx context.Context) error {
	return wr.runtime.Close(ctx)
}

// WASMFunction adapts a WASM function to work with the FunctionImpl interface
// This allows WASM functions to be called through the registry
func (wr *WASMRuntime) WASMFunction(ctx context.Context, ref string, funcName string) FunctionImpl {
	return func(args ...interface{}) (interface{}, error) {
		return wr.CallFunction(ctx, ref, funcName, args...)
	}
}
