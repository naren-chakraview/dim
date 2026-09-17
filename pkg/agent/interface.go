package agent

// Package agent provides the public Agent Interface contract for dim.
//
// This is a published external contract for building AI agents that operate
// on dim routes. The interface is stable and versioned independently from
// the dim engine, following semantic versioning.
//
// External parties can import this package to:
// - Check InterfaceVersion for compatibility
// - Define their own operation request/response types that match the interface
// - Implement agent clients that call dim operations
//
// The agent interface is accessed via HTTP at endpoints like:
//   GET  http://localhost:9090/health      - Server health check
//   GET  http://localhost:9090/operations  - List available operations
//   POST http://localhost:9090/call        - Call an operation
//
// See internal/agent/types.go for operation-specific request/response types.

// InterfaceVersion is the published Agent Interface version.
// External parties building agents should import and check this version
// to ensure compatibility with the dim agent interface.
//
// Semantic versioning applies: MAJOR.MINOR.PATCH
// - MAJOR: Breaking changes to operation signatures or required fields
// - MINOR: New operations or backwards-compatible additions
// - PATCH: Bug fixes or clarifications
const InterfaceVersion = "1.0.0"

// OperationError represents an error response from an agent operation
type OperationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ResponseEnvelope wraps all agent operation responses
type ResponseEnvelope struct {
	InterfaceVersion   string        `json:"interface_version"`
	Timestamp          string        `json:"timestamp"`
	Result             interface{}   `json:"result,omitempty"`
	Error              *OperationError `json:"error,omitempty"`
	DeprecationNotice  string        `json:"deprecation_notice,omitempty"`
}

// OperationMetadata describes an available operation
type OperationMetadata struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}
