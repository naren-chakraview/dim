package sdk

// Registry manages function plugins available to dim routes
type Registry interface {
	// Register adds a function definition and its implementation to the registry
	// Returns an error if a function with that name already exists or if inputs are invalid
	Register(def *FunctionDef, impl Function) error

	// Get retrieves a registered function by name
	// Returns the function and true if found, nil and false otherwise
	Get(name string) (Function, bool)

	// GetDef retrieves the function definition (metadata) by name
	// Returns the definition and true if found, nil and false otherwise
	GetDef(name string) (*FunctionDef, bool)

	// List returns all registered function names
	List() []string

	// Count returns the number of registered functions
	Count() int

	// Unregister removes a function from the registry
	// Returns an error if the function doesn't exist
	Unregister(name string) error
}

// PluginLoader loads plugins from external sources
type PluginLoader interface {
	// Load loads a plugin from a reference (path, URL, etc.)
	// Returns a Function ready to call, or an error
	Load(def *FunctionDef) (Function, error)

	// Unload releases resources held by a loaded plugin
	// Returns an error if the plugin wasn't loaded or unloading failed
	Unload(name string) error

	// Close shuts down the plugin loader and releases all resources
	// Should be called when no more plugins will be loaded
	Close() error
}
