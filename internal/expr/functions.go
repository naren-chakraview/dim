package expr

import (
	"fmt"
	"sync"
)

// FunctionType represents the kind of custom function (plugin or WASM)
type FunctionType string

const (
	FunctionTypePlugin FunctionType = "plugin"
	FunctionTypeWasm   FunctionType = "wasm"
)

// FunctionDef describes a registered custom function
type FunctionDef struct {
	Name    string       `json:"name"`
	Type    FunctionType `json:"type"`
	Runtime string       `json:"runtime"` // "go" for plugin, "wasm" or language for WASM
	Ref     string       `json:"ref"`     // Path to plugin executable or WASM file
}

// FunctionImpl is the actual callable function signature for custom functions
// Takes input arguments and returns a result or error
type FunctionImpl func(args ...interface{}) (interface{}, error)

// Registry manages custom functions available to JSONata expressions
type Registry struct {
	mu        sync.RWMutex
	functions map[string]*FunctionDef
	impls     map[string]FunctionImpl
}

// NewRegistry creates an empty functions registry
func NewRegistry() *Registry {
	return &Registry{
		functions: make(map[string]*FunctionDef),
		impls:     make(map[string]FunctionImpl),
	}
}

// Register adds a function definition and its implementation to the registry.
// Returns an error if a function with that name already exists.
func (r *Registry) Register(def *FunctionDef, impl FunctionImpl) error {
	if def == nil || impl == nil {
		return fmt.Errorf("function definition and implementation cannot be nil")
	}
	if def.Name == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.functions[def.Name]; exists {
		return fmt.Errorf("function %q already registered", def.Name)
	}

	r.functions[def.Name] = def
	r.impls[def.Name] = impl
	return nil
}

// Lookup retrieves a function by name
// Returns the function definition, implementation, and a boolean indicating if it was found
func (r *Registry) Lookup(name string) (*FunctionDef, FunctionImpl, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, defExists := r.functions[name]
	impl, implExists := r.impls[name]

	return def, impl, defExists && implExists
}

// GetAll returns a copy of all registered function definitions
func (r *Registry) GetAll() map[string]*FunctionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*FunctionDef, len(r.functions))
	for name, def := range r.functions {
		result[name] = def
	}
	return result
}

// Count returns the number of registered functions
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.functions)
}
