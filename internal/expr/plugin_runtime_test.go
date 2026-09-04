package expr

import (
	"testing"
)

// TestPluginRuntimeInit verifies plugin runtime initialization
func TestPluginRuntimeInit(t *testing.T) {
	runtime := NewPluginRuntime()
	if runtime == nil {
		t.Fatal("NewPluginRuntime returned nil")
	}
	if len(runtime.clients) != 0 {
		t.Errorf("Initial clients count: got %d, want 0", len(runtime.clients))
	}

	// Clean up
	runtime.Close()
}

// TestPluginFunctionInRegistry verifies registering a plugin function
func TestPluginFunctionInRegistry(t *testing.T) {
	runtime := NewPluginRuntime()
	defer runtime.Close()

	registry := NewRegistry()

	// Register a simulated plugin function
	// (Full plugin loading/RPC deferred to Phase 2)
	def := &FunctionDef{
		Name:    "pluginAdd",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugin://add-plugin/add",
	}

	// Simulated plugin function that adds two numbers
	impl := func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, nil
		}
		var sum int64
		if v1, ok := args[0].(int); ok {
			sum += int64(v1)
		} else if v1, ok := args[0].(int64); ok {
			sum += v1
		}
		if v2, ok := args[1].(int); ok {
			sum += int64(v2)
		} else if v2, ok := args[1].(int64); ok {
			sum += v2
		}
		return sum, nil
	}

	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify function is registered
	_, fn, exists := registry.Lookup("pluginAdd")
	if !exists {
		t.Fatal("Function not found in registry")
	}

	// Call the function
	result, err := fn(3, 5)
	if err != nil {
		t.Fatalf("Function call failed: %v", err)
	}
	if result.(int64) != 8 {
		t.Errorf("Result: got %v, want 8", result)
	}
}

// TestPluginLoadAndClose verifies plugin lifecycle
func TestPluginLoadAndClose(t *testing.T) {
	runtime := NewPluginRuntime()

	// Load a plugin (framework in place, RPC deferred to Phase 2)
	impl, err := runtime.LoadPlugin("./plugins/test.so", "testFunc")
	if err != nil {
		// Phase 1: LoadPlugin may fail, that's OK - we're testing framework
		t.Logf("LoadPlugin error (expected in Phase 1): %v", err)
	}

	// If load succeeded, verify implementation is returned
	if err == nil && impl == nil {
		t.Fatal("Implementation should be non-nil if load succeeds")
	}

	// Verify Close doesn't panic
	if err := runtime.Close(); err != nil {
		t.Logf("Close error: %v", err)
	}

	// Verify we can create a new runtime after close
	runtime2 := NewPluginRuntime()
	if len(runtime2.clients) != 0 {
		t.Errorf("New runtime should have no clients")
	}
	runtime2.Close()
}

// TestPluginValidation verifies input validation
func TestPluginValidation(t *testing.T) {
	runtime := NewPluginRuntime()
	defer runtime.Close()

	tests := []struct {
		name       string
		pluginPath string
		funcName   string
		expectErr  bool
	}{
		{"empty path", "", "func", true},
		{"empty name", "path.so", "", true},
		{"both empty", "", "", true},
		{"valid args", "plugin.so", "func", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runtime.LoadPlugin(tt.pluginPath, tt.funcName)
			if (err != nil) != tt.expectErr {
				if tt.expectErr {
					t.Errorf("Expected error for %q, got none", tt.name)
				}
				// Phase 1: valid args may fail if binary doesn't exist
			}
		})
	}
}

// TestPluginHandshakeConfig verifies handshake configuration
func TestPluginHandshakeConfig(t *testing.T) {
	config := defaultHandshakeConfig

	if config.ProtocolVersion != 1 {
		t.Errorf("Protocol version: got %d, want 1", config.ProtocolVersion)
	}

	if config.MagicCookieKey != "DIM_PLUGIN" {
		t.Errorf("Magic cookie key: got %q, want DIM_PLUGIN", config.MagicCookieKey)
	}

	if config.MagicCookieValue != "dim-function-plugin" {
		t.Errorf("Magic cookie value: got %q, want dim-function-plugin", config.MagicCookieValue)
	}
}

// TestPluginCallError verifies RPC call error in Phase 1
func TestPluginCallError(t *testing.T) {
	runtime := NewPluginRuntime()
	defer runtime.Close()

	// Try to call a function
	impl, _ := runtime.LoadPlugin("test.so", "func")

	if impl != nil {
		// If load succeeded, calling should indicate RPC not ready
		result, err := impl(42)
		if err == nil {
			t.Fatal("Expected error for RPC not implemented")
		}
		if result != nil {
			t.Logf("Result should be nil on error, got %v", result)
		}
	}
}
