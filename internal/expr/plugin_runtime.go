package expr

import (
	"fmt"
	"os"
	"sync"
)

// PluginRuntime manages native Go plugin execution via hashicorp/go-plugin
// Uses subprocess + RPC model for portability across go-plugin versions
type PluginRuntime struct {
	mu      sync.RWMutex
	plugins map[string]PluginInstance
}

// PluginInstance represents a loaded plugin
type PluginInstance struct {
	Ref    string
	Funcs  map[string]FunctionImpl // Functions provided by this plugin
	Closer func() error             // Cleanup function
}

// NewPluginRuntime creates a new plugin runtime
func NewPluginRuntime() *PluginRuntime {
	return &PluginRuntime{
		plugins: make(map[string]PluginInstance),
	}
}

// LoadPlugin loads a plugin executable and establishes communication
// ref is the path to the plugin executable
// For now, this validates the plugin file exists
// Full go-plugin RPC integration happens when actual plugins are implemented
func (pr *PluginRuntime) LoadPlugin(ref string) error {
	// Verify the plugin executable exists
	if _, err := os.Stat(ref); err != nil {
		return fmt.Errorf("plugin file %q not found: %w", ref, err)
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Create a plugin instance entry
	pr.plugins[ref] = PluginInstance{
		Ref:   ref,
		Funcs: make(map[string]FunctionImpl),
		Closer: func() error {
			// Cleanup logic will be implemented with actual go-plugin integration
			return nil
		},
	}

	return nil
}

// RegisterFunction registers a function provided by a plugin
// This is called when a plugin provides a named function
func (pr *PluginRuntime) RegisterFunction(ref string, funcName string, fn FunctionImpl) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plugin, exists := pr.plugins[ref]
	if !exists {
		return fmt.Errorf("plugin %q not loaded", ref)
	}

	if fn == nil {
		return fmt.Errorf("function implementation cannot be nil")
	}

	plugin.Funcs[funcName] = fn
	pr.plugins[ref] = plugin
	return nil
}

// CallFunction calls a function in a loaded plugin
func (pr *PluginRuntime) CallFunction(ref string, funcName string, args ...interface{}) (interface{}, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugin, exists := pr.plugins[ref]
	if !exists {
		return nil, fmt.Errorf("plugin %q not loaded", ref)
	}

	fn, fnExists := plugin.Funcs[funcName]
	if !fnExists {
		return nil, fmt.Errorf("function %q not found in plugin %q", funcName, ref)
	}

	return fn(args...)
}

// UnloadPlugin unloads a plugin and closes the connection
func (pr *PluginRuntime) UnloadPlugin(ref string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plugin, exists := pr.plugins[ref]
	if !exists {
		return fmt.Errorf("plugin %q not loaded", ref)
	}

	if plugin.Closer != nil {
		if err := plugin.Closer(); err != nil {
			return fmt.Errorf("failed to close plugin: %w", err)
		}
	}

	delete(pr.plugins, ref)
	return nil
}

// Close closes all loaded plugins
func (pr *PluginRuntime) Close() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for ref, plugin := range pr.plugins {
		if plugin.Closer != nil {
			plugin.Closer()
		}
		delete(pr.plugins, ref)
	}
	return nil
}

// PluginFunction adapts a plugin function to work with the FunctionImpl interface
// This allows plugin functions to be called through the registry
func (pr *PluginRuntime) PluginFunction(ref string, funcName string) FunctionImpl {
	return func(args ...interface{}) (interface{}, error) {
		return pr.CallFunction(ref, funcName, args...)
	}
}
