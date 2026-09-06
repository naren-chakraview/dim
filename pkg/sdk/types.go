package sdk

import "time"

// FunctionType represents the kind of function (native plugin or WASM)
type FunctionType string

const (
	FunctionTypeNative FunctionType = "native"
	FunctionTypeWasm   FunctionType = "wasm"
)

// Function is the core interface all plugins must implement
// A plugin is a reusable unit of computation that accepts input arguments,
// performs domain-specific logic, and returns a result or error.
type Function interface {
	// Call executes the plugin with the given arguments
	// ctx: cancellation context (respects timeout)
	// args: JSON-serializable input arguments (may be empty)
	// Returns: JSON-serializable result, or error if call failed
	//
	// Implementations must be deterministic (same inputs → same outputs)
	// and have no side effects (stateless).
	Call(args ...interface{}) (interface{}, error)
}

// FunctionDef describes a plugin function for registration and discovery
type FunctionDef struct {
	// Name is the function identifier (e.g., "validate_email", "enrich_customer")
	Name string

	// Type indicates the plugin runtime (native or wasm)
	Type FunctionType

	// Runtime is the specific runtime identifier
	// For native: "go" or the language (e.g., "rust")
	// For wasm: "wazero" or the implementation
	Runtime string

	// Ref is the reference to the plugin (path to binary or WASM file)
	Ref string

	// Version is the SDK version this plugin targets (e.g., "1.0")
	Version string

	// Timeout is the maximum execution time for a call (default: 5s)
	Timeout time.Duration
}

// CallResult wraps a function call result with metadata
type CallResult struct {
	// Result is the return value from the function
	Result interface{}

	// Error is non-nil if the call failed
	Error error

	// Duration is how long the call took
	Duration time.Duration
}
