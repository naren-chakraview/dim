package sdk

// SDKVersion is the current version of the plugin SDK
// This version is independent from the dim engine version
// and follows semantic versioning: MAJOR.MINOR.PATCH
//
// Version 1.0:
// - Native plugins via hashicorp/go-plugin (RPC)
// - WASM plugins via wazero (linear memory ABI)
// - Deterministic function execution
// - JSON serialization for I/O
const SDKVersion = "1.0"

// Version returns the current SDK version
func Version() string {
	return SDKVersion
}

// IsCompatibleVersion checks if a plugin's target version is compatible with current SDK
// A plugin built against SDK v1.0 is compatible with engine shipping SDK v1.x
// Breaking changes only occur at major version boundaries (v1 → v2)
func IsCompatibleVersion(targetVersion string) bool {
	// Simple check: major version must match
	// targetVersion format: "1.0", "1.1", etc.
	if len(targetVersion) < 1 {
		return false
	}

	currentMajor := SDKVersion[0]
	targetMajor := targetVersion[0]

	return currentMajor == targetMajor
}
