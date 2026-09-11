# M4.6 Agent-Facing Interface Specification

**Interface Version:** 1.0.0  
**Transport:** Model Context Protocol (MCP)  
**Status:** Specification (ready for implementation)  
**Last Updated:** 2026-09-11

---

## Overview

The Agent Interface is a published, versioned API that external agents can use to:
- **Validate** route configurations
- **Test** routes with fixtures
- **Scaffold** new domains with starter routes
- **Query** lineage and provenance for routes
- **Discover** capabilities and supported adapters/steps (M4.7)

**Core Principle:** Agents use identical operations to those available via `dimctl` CLI or CI pipelines. No shortcuts, no internal access.

---

## Version Strategy

### Semantic Versioning

```go
type InterfaceVersion struct {
    Major int    // Breaking changes (0→1 means agents must update)
    Minor int    // Backward-compatible additions (1.0→1.1, agents don't break)
    Patch int    // Bug fixes (1.0.0→1.0.1, agents don't update)
}

const CurrentInterfaceVersion = "1.0.0"
```

### Versioning Rules

- **Major bump:** Removing operations, changing operation signatures, incompatible error format changes
- **Minor bump:** Adding new operations, adding optional parameters, expanding response types
- **Patch bump:** Bug fixes, performance improvements, documentation clarifications
- **Deprecation:** 6-month notice before removal; agents warned via `deprecation_notice` in responses

### Version Compatibility

Agents MUST check `interface_version` in every response. If `Major` differs from what agent expects, agent should halt and report version mismatch.

---

## Operations

Each operation is exposed as an MCP Tool. Request/response types use JSON marshaling.

### 1. Validate

**Purpose:** Check if a route configuration is valid according to schema and governance rules.

**Operation Definition:**

```
Tool Name: validate_route
Display Name: Validate Route Configuration
Description: Validate a route YAML against JSON schema and governance rules
```

**Request:**

```typescript
{
  "route_config_path": string    // Path to route YAML (relative to workspace root)
  "strict_mode": boolean         // Treat warnings as errors (default: false)
}
```

**Response:**

```typescript
{
  "interface_version": string    // e.g., "1.0.0"
  "valid": boolean
  "errors": ValidationError[]    // Empty if valid
  "warnings": string[]           // Even if valid
  "route_version": string        // SHA256 hash of resolved route (for lineage)
  "timestamp": string            // RFC3339
}

type ValidationError = {
  "code": string                 // e.g., "SCHEMA_VIOLATION", "MISSING_IMPORT"
  "message": string
  "path": string                 // JSONPath to error location in config
  "suggested_fix": string?       // Optional hint for repair
}
```

**Error Handling:**

| Error | Code | HTTP (if REST added) |
|-------|------|-----|
| File not found | FILE_NOT_FOUND | 404 |
| Invalid YAML | YAML_PARSE_ERROR | 400 |
| Schema violation | SCHEMA_VIOLATION | 400 |
| Missing governance import | MISSING_IMPORT | 400 |

---

### 2. Test

**Purpose:** Run fixtures against a route to verify behavior.

**Operation Definition:**

```
Tool Name: test_route
Display Name: Test Route with Fixtures
Description: Run integration tests (fixtures) against a route configuration
```

**Request:**

```typescript
{
  "route_config_path": string   // Path to route YAML
  "fixtures_path": string        // Path to test fixtures file or directory
  "timeout_ms": number?          // Test timeout (default: 30000)
}
```

**Response:**

```typescript
{
  "interface_version": string
  "passed": boolean
  "test_results": TestResult[]
  "summary": {
    "total": number
    "passed": number
    "failed": number
    "skipped": number
    "duration_ms": number
  }
  "timestamp": string
}

type TestResult = {
  "name": string
  "passed": boolean
  "duration_ms": number
  "error"?: string               // If failed
  "expected"?: unknown
  "actual"?: unknown
}
```

**Error Handling:**

| Error | Code |
|-------|------|
| Fixtures not found | FIXTURES_NOT_FOUND |
| Timeout | TEST_TIMEOUT |
| Invalid fixture format | FIXTURE_FORMAT_ERROR |
| Route validation failed | VALIDATION_FAILED |

---

### 3. Scaffold

**Purpose:** Generate a domain directory with starter route template.

**Operation Definition:**

```
Tool Name: scaffold_domain
Display Name: Scaffold New Domain
Description: Generate a domain directory with governance imports and starter route
```

**Request:**

```typescript
{
  "domain": string               // Domain name (required)
  "source_type": string          // e.g., "http", "kafka", "file" (default: "http")
  "sink_type": string            // e.g., "file", "kafka", "http" (default: "file")
  "with_contract": boolean       // Generate contract template (default: false)
  "template": string             // "passthrough" | "transform" | "contract" (default: "passthrough")
}
```

**Response:**

```typescript
{
  "interface_version": string
  "success": boolean
  "domain_path": string          // Path where domain was created
  "files_created": string[]       // List of generated files
  "next_steps": string[]         // Actionable guidance
  "timestamp": string
}
```

**Error Handling:**

| Error | Code |
|-------|------|
| Domain already exists | DOMAIN_EXISTS |
| Invalid domain name | INVALID_DOMAIN_NAME |
| Permission denied | PERMISSION_DENIED |

---

### 4. GetLineage

**Purpose:** Retrieve the lineage chain for a message (what happened to it).

**Operation Definition:**

```
Tool Name: get_lineage
Display Name: Query Message Lineage
Description: Get the lineage chain for a message by ID
```

**Request:**

```typescript
{
  "message_id": string           // Message correlation ID
}
```

**Response:**

```typescript
{
  "interface_version": string
  "found": boolean
  "lineage_records": LineageRecord[]
  "timestamp": string
}

type LineageRecord = {
  "id": string
  "message_id": string
  "route_name": string
  "domain": string
  "created_at": string           // RFC3339
  "step": string                 // Which step processed it
  "status": "success" | "error"
}
```

**Error Handling:**

| Error | Code |
|-------|------|
| Message not found | LINEAGE_NOT_FOUND |
| Lineage store unavailable | STORE_UNAVAILABLE |

---

### 5. GetProvenance

**Purpose:** Retrieve the complete provenance chain for a subject (all related messages).

**Operation Definition:**

```
Tool Name: get_provenance
Display Name: Query Subject Provenance
Description: Get the complete provenance chain for a subject ID
```

**Request:**

```typescript
{
  "subject_id": string           // Subject identifier
}
```

**Response:**

```typescript
{
  "interface_version": string
  "found": boolean
  "provenance_chain": ProvenanceRecord[]
  "timestamp": string
}

type ProvenanceRecord = {
  "id": string
  "subject_id": string
  "message_id": string
  "route_name": string
  "created_at": string
  "status": "success" | "error"
}
```

**Error Handling:**

| Error | Code |
|-------|------|
| Subject not found | PROVENANCE_NOT_FOUND |
| Store unavailable | STORE_UNAVAILABLE |

---

### 6. GetCapabilities (M4.7 integration)

**Purpose:** Enumerate all supported adapters, steps, EIPs with their schemas.

**Operation Definition:**

```
Tool Name: get_capabilities
Display Name: Query Supported Capabilities
Description: Get enumerable catalog of all supported adapters, steps, and schemas
```

**Request:**

```typescript
{
  "filter_type": string?         // Optional filter: "adapter" | "step" | "eip"
}
```

**Response:**

```typescript
{
  "interface_version": string
  "capabilities": Capability[]
  "schema_version": string       // Version of schemas/route.schema.json used
  "timestamp": string
}

type Capability = {
  "name": string                 // e.g., "kafka-source", "translate-step"
  "type": "adapter" | "step" | "eip"
  "config_schema": object        // JSON Schema for this capability
  "description": string
  "example": object?             // Optional example config
}
```

**Error Handling:**

| Error | Code |
|-------|------|
| Schema not found | SCHEMA_NOT_FOUND |

---

## Shared Response Format

Every operation returns a response envelope:

```typescript
{
  "interface_version": string        // Always present; agents check this first
  "timestamp": string                // RFC3339; when operation executed
  "result": object                   // Operation-specific result
  "error": OperationError?           // Present only on error
  "deprecation_notice": string?      // If operation is being deprecated
}

type OperationError = {
  "code": string                     // Machine-readable error code
  "message": string                  // Human-readable message
  "details": object?                 // Operation-specific error details
}
```

---

## Error Codes (Standard)

| Code | Meaning | Recovery |
|------|---------|----------|
| INVALID_REQUEST | Malformed request | Check request format against spec |
| VERSION_MISMATCH | Interface version incompatible | Agent must update |
| OPERATION_TIMEOUT | Operation took too long | Increase timeout or reduce scope |
| INTERNAL_ERROR | Unexpected server error | Retry; if persistent, file issue |
| NOT_IMPLEMENTED | Operation not yet available | Check interface version |

---

## MCP Tool Binding

All operations are exposed as MCP Tools with this schema:

```json
{
  "name": "<tool_name>",
  "displayName": "<display_name>",
  "description": "<description>",
  "inputSchema": {
    "type": "object",
    "properties": { ... },
    "required": [ ... ]
  }
}
```

Example (Validate):

```json
{
  "name": "validate_route",
  "displayName": "Validate Route Configuration",
  "description": "Validate a route YAML against JSON schema and governance rules",
  "inputSchema": {
    "type": "object",
    "properties": {
      "route_config_path": { "type": "string", "description": "Path to route YAML" },
      "strict_mode": { "type": "boolean", "description": "Treat warnings as errors", "default": false }
    },
    "required": ["route_config_path"]
  }
}
```

---

## Boundary: Exposed vs. Internal

### Exposed (Safe for Agents)

- Operations defined above (Validate, Test, Scaffold, etc.)
- Public types: `ValidationError`, `TestResult`, `Capability`, etc.
- `AgentInterfaceVersion` constant
- Configuration file paths (relative to workspace root)

### Internal (Not Exposed)

- `internal/config`, `internal/engine`, `internal/steps` packages
- Executor implementation details
- Internal metrics/observability structures
- Undocumented operation modes

**Enforcement:** Interface layer has compile-time boundary; agents cannot import `internal/` packages.

---

## Usage Example (Pseudocode)

```go
// Agent connects to MCP server
client := mcp.Connect("dimctl-agent-server")
defer client.Close()

// Call validate operation
result := client.CallTool("validate_route", map[string]interface{}{
  "route_config_path": "domains/payments/order-payment.yaml",
  "strict_mode": true,
})

// Check version
if result.InterfaceVersion != "1.0.0" {
  log.Fatal("Version mismatch")
}

// Check result
if !result.Valid {
  for _, err := range result.Errors {
    log.Printf("Error: %s at %s", err.Message, err.Path)
  }
  return errors.New("validation failed")
}

log.Printf("✓ Route validated (version=%s)", result.RouteVersion)
```

---

## Testing the Interface

### Unit Test Coverage

- Each operation callable with valid/invalid inputs
- Error codes returned correctly
- Versioning included in every response
- No internal package exposure

### Conformance Test

- Reference agent (examples/agent/route_validator.py) calls all operations via MCP
- Agent uses only published interface (no internal imports)
- All responses parsed successfully
- Errors handled gracefully

---

## Deprecation Timeline

When an operation or parameter is deprecated:

1. Add `"deprecation_notice"` to response: `"validate_route.strict_mode will be removed in v2.0.0 (2027-09-11)"`
2. Operation continues to work (backward compatible)
3. 6-month notice period before removal
4. On removal date, update to v2.0.0 (Major version bump)

---

## Future Extensions

This spec is designed to be extended without breaking v1 agents:

- **v1.1:** Add new optional operation parameter → backward compatible
- **v1.2:** Add new operation → backward compatible (agents ignore unknown tools)
- **v2.0:** Remove deprecated operation → requires agent update

---

## Next Steps

1. ✅ **Transport Decision** (Task 1) — MCP chosen
2. ✅ **Interface Spec** (Task 2, this document) — Operations defined
3. **Implementation** (Task 3) — Build MCP server wrapping operations
4. **Conformance** (Task 4) — Reference agent using this spec only
5. **Documentation** (Task 5) — Integration guide and versioning policy

---

**Specification Approved:** 2026-09-11  
**Ready for Implementation:** Yes
