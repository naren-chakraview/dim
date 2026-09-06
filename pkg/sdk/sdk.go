// Package sdk provides the public interface for building dim function plugins.
//
// This package defines the versioned contract that third-party developers use
// to create plugins without accessing dim's internal packages.
//
// Two plugin runtimes are supported:
//
//   - Native plugins: Subprocess-based via hashicorp/go-plugin, JSON-RPC communication
//   - WASM plugins: In-process sandboxed modules via wazero, linear-memory I/O
//
// Both conform to the same logical Function interface.
//
// Example native plugin (Go):
//
//	func (p *MyPlugin) Call(args ...interface{}) (interface{}, error) {
//	    // Process input arguments
//	    // Return result or error
//	}
//
// Example WASM plugin (Rust):
//
//	#[no_mangle]
//	pub fn my_function(args_ptr: i32, args_len: i32) -> u64 {
//	    // Read JSON from memory at args_ptr
//	    // Process and write result back
//	    // Return (result_ptr, result_len)
//	}
//
// For detailed specification, see design/M3.3.1_PLUGIN_SDK_SPEC.md
package sdk
