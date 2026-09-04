package expr

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WASMRuntime manages WASM module loading and function execution
type WASMRuntime struct {
	runtime wazero.Runtime
	modules map[string]api.Module
}

// NewWASMRuntime creates a new WASM runtime
func NewWASMRuntime() *WASMRuntime {
	return &WASMRuntime{
		runtime: wazero.NewRuntime(context.Background()),
		modules: make(map[string]api.Module),
	}
}

// LoadModule loads a WASM module from a file path
// Returns the module name for reference in function calls
func (wr *WASMRuntime) LoadModule(ctx context.Context, modulePath string, moduleName string) (string, error) {
	if modulePath == "" || moduleName == "" {
		return "", fmt.Errorf("module path and name cannot be empty")
	}

	if _, exists := wr.modules[moduleName]; exists {
		return "", fmt.Errorf("module %q already loaded", moduleName)
	}

	// Read WASM binary from file
	wasmBinary, err := readWasmBinary(modulePath)
	if err != nil {
		return "", fmt.Errorf("failed to read WASM binary: %w", err)
	}

	// Instantiate the module
	// Note: full module instantiation (with imports) is deferred to Phase 2
	// For Phase 1, we support read-only module introspection and basic calls
	module, err := wr.runtime.Instantiate(ctx, wasmBinary)
	if err != nil {
		return "", fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	wr.modules[moduleName] = module
	return moduleName, nil
}

// CallFunction invokes a WASM-exported function
// Supports numeric types (i32, i64, f32, f64) and returns numeric results
// Complex type marshaling deferred to Phase 2
func (wr *WASMRuntime) CallFunction(moduleName string, functionName string, args ...interface{}) (interface{}, error) {
	module, exists := wr.modules[moduleName]
	if !exists {
		return nil, fmt.Errorf("module %q not loaded", moduleName)
	}

	// Get the exported function
	fn := module.ExportedFunction(functionName)
	if fn == nil {
		return nil, fmt.Errorf("function %q not exported from module %q", functionName, moduleName)
	}

	// Convert input arguments to uint64 (WASM calling convention)
	wasmArgs := make([]uint64, len(args))
	for i, arg := range args {
		wasmArgs[i] = toWasmInt64(arg)
	}

	// Call the function
	results, err := fn.Call(context.Background(), wasmArgs...)
	if err != nil {
		return nil, fmt.Errorf("function call failed: %w", err)
	}

	// Convert result back to Go value
	if len(results) == 0 {
		return nil, nil
	}

	return fromWasmInt64(results[0]), nil
}

// Close releases WASM runtime resources
func (wr *WASMRuntime) Close() error {
	if wr.runtime != nil {
		return wr.runtime.Close(context.Background())
	}
	return nil
}

// toWasmInt64 converts a Go value to WASM i64 calling convention
// Supports: int, int32, int64, float32, float64
func toWasmInt64(v interface{}) uint64 {
	switch val := v.(type) {
	case int:
		return uint64(val)
	case int32:
		return uint64(val)
	case int64:
		return uint64(val)
	case float32:
		return uint64(val)
	case float64:
		return uint64(val)
	default:
		return 0
	}
}

// fromWasmInt64 converts a WASM i64 result to Go value
// Returns as int64 (caller responsible for interpretation)
func fromWasmInt64(v uint64) interface{} {
	return int64(v)
}

// readWasmBinary reads a WASM binary file (implementation in util file)
func readWasmBinary(path string) ([]byte, error) {
	// Placeholder: actual implementation reads from file
	// In tests, binaries are created programmatically
	return nil, fmt.Errorf("file-based WASM loading not yet implemented; use programmatic module creation in tests")
}

// RegisterWASMFunction registers a loaded WASM function in the functions registry
func (wr *WASMRuntime) RegisterWASMFunction(registry *Registry, moduleName, functionName string) error {
	// Verify function exists in module
	module, exists := wr.modules[moduleName]
	if !exists {
		return fmt.Errorf("module %q not loaded", moduleName)
	}

	fn := module.ExportedFunction(functionName)
	if fn == nil {
		return fmt.Errorf("function %q not exported from module %q", functionName, moduleName)
	}

	// Create function definition
	def := &FunctionDef{
		Name:    functionName,
		Type:    FunctionTypeWasm,
		Runtime: "wazero",
		Ref:     fmt.Sprintf("wasm://%s/%s", moduleName, functionName),
	}

	// Create implementation that calls the WASM function
	impl := func(args ...interface{}) (interface{}, error) {
		return wr.CallFunction(moduleName, functionName, args...)
	}

	// Register in the functions registry
	return registry.Register(def, impl)
}
