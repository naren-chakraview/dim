package expr

import (
	"testing"
)

// TestPluginRuntimeInit tests basic plugin runtime initialization.
func TestPluginRuntimeInit(t *testing.T) {
	// Create a plugin runtime
	runtime := NewPluginRuntime()

	// Verify runtime is not nil
	if runtime == nil {
		t.Fatal("runtime is nil")
	}

	// Verify plugins map is initialized
	if runtime.plugins == nil {
		t.Fatal("plugins map is nil")
	}

	// Close the runtime
	err := runtime.Close()
	if err != nil {
		t.Fatalf("failed to close runtime: %v", err)
	}
}

// TestPluginFunctionInRegistry tests registering and looking up plugin functions.
func TestPluginFunctionInRegistry(t *testing.T) {
	registry := NewRegistry()

	// Create a plugin runtime
	runtime := NewPluginRuntime()
	defer runtime.Close()

	// Define a plugin function
	def := &FunctionDef{
		Name:    "pluginDouble",
		Type:    FunctionTypePlugin,
		Runtime: "go",
		Ref:     "plugins/double",
	}

	// Create a plugin function wrapper
	impl := runtime.PluginFunction(def.Ref, "Call")

	err := registry.Register(def, impl)
	if err != nil {
		t.Fatalf("failed to register plugin function: %v", err)
	}

	// Verify the function is registered
	if registry.Count() != 1 {
		t.Errorf("expected 1 function, got %d", registry.Count())
	}

	// Look up the function
	retrievedDef, retrievedImpl, found := registry.Lookup("pluginDouble")
	if !found {
		t.Fatal("plugin function not found in registry")
	}

	if retrievedDef.Type != FunctionTypePlugin {
		t.Errorf("expected type 'plugin', got %q", retrievedDef.Type)
	}

	if retrievedImpl == nil {
		t.Fatal("function implementation is nil")
	}
}

// TestPluginRegistration tests registering functions within a plugin.
func TestPluginRegistration(t *testing.T) {
	runtime := NewPluginRuntime()
	defer runtime.Close()

	// Create a test function
	testFunc := func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return 0, nil
		}
		switch v := args[0].(type) {
		case int:
			return v * 2, nil
		default:
			return v, nil
		}
	}

	// Register a function with the plugin runtime
	err := runtime.RegisterFunction("plugins/double", "doubleInt", testFunc)
	if err == nil {
		t.Fatal("expected error when plugin not loaded, got nil")
	}
}

// TestPluginLoadAndClose tests loading and closing plugins.
func TestPluginLoadAndClose(t *testing.T) {
	runtime := NewPluginRuntime()
	defer runtime.Close()

	// Note: We can't test actual plugin loading without a real executable
	// This test validates the error handling
	err := runtime.LoadPlugin("/nonexistent/plugin")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}
