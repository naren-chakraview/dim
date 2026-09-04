package expr

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-plugin"
)

// PluginRuntime manages native Go plugin loading and execution
// Uses hashicorp/go-plugin for subprocess-based RPC (version-safe, no cgo needed)
type PluginRuntime struct {
	mu      sync.RWMutex
	clients map[string]*plugin.Client // Loaded plugin client processes
}

// PluginInterface defines the contract for a callable plugin function
type PluginInterface interface {
	Call(args ...interface{}) (interface{}, error)
}

// NewPluginRuntime creates a new plugin runtime manager
func NewPluginRuntime() *PluginRuntime {
	return &PluginRuntime{
		clients: make(map[string]*plugin.Client),
	}
}

// LoadPlugin loads a plugin executable and connects via RPC
// pluginPath: path to the compiled plugin executable
// functionName: name of the function to call in the plugin
// Returns a callable implementation in the functions registry
func (pr *PluginRuntime) LoadPlugin(pluginPath string, functionName string) (FunctionImpl, error) {
	if pluginPath == "" || functionName == "" {
		return nil, fmt.Errorf("plugin path and function name cannot be empty")
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	key := pluginPath + "::" + functionName

	// Check if already loaded
	if _, exists := pr.clients[key]; exists {
		return nil, fmt.Errorf("plugin %q already loaded", key)
	}

	// Spawn the plugin process
	// Note: Full plugin spawning and handshake deferred to Phase 2
	// For Phase 1, we provide the framework and error handling
	client := plugin.NewClient(&plugin.ClientConfig{
		Cmd:             nil, // Placeholder; actual plugin binary path in Phase 2
		HandshakeConfig: defaultHandshakeConfig,
		Managed:         true,
		Stderr:          nil,
	})

	// Store client reference for lifecycle management
	pr.clients[key] = client

	// Return a function that will call the plugin
	// Phase 2 will implement actual RPC communication
	return func(args ...interface{}) (interface{}, error) {
		return pr.callPlugin(key, args...)
	}, nil
}

// callPlugin invokes a function through a loaded plugin
// Phase 2 will implement actual RPC marshaling
func (pr *PluginRuntime) callPlugin(key string, args ...interface{}) (interface{}, error) {
	pr.mu.RLock()
	client, exists := pr.clients[key]
	pr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin %q not loaded", key)
	}

	if client == nil {
		return nil, fmt.Errorf("plugin %q client is nil", key)
	}

	// Phase 1: Framework in place, actual RPC call deferred
	// In Phase 2, this will:
	// 1. Establish RPC connection via client.Client()
	// 2. Marshal arguments
	// 3. Call remote function via rpc.Call
	// 4. Unmarshal result
	return nil, fmt.Errorf("plugin RPC execution not yet implemented; framework in place for Phase 2")
}

// Close disconnects all loaded plugins
func (pr *PluginRuntime) Close() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for key, client := range pr.clients {
		if client != nil {
			client.Kill()
		}
		delete(pr.clients, key)
	}
	return nil
}

// RegisterPluginFunction registers a loaded plugin function in the registry
func (pr *PluginRuntime) RegisterPluginFunction(registry *Registry, pluginPath, functionName string) error {
	// Load the plugin
	impl, err := pr.LoadPlugin(pluginPath, functionName)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Create function definition
	def := &FunctionDef{
		Name:    functionName,
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     fmt.Sprintf("plugin://%s/%s", pluginPath, functionName),
	}

	// Register in the functions registry
	return registry.Register(def, impl)
}

// Handshake configuration for plugin communication
// This is the interface both plugin and host must agree on
var defaultHandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "DIM_PLUGIN",
	MagicCookieValue: "dim-function-plugin",
}
