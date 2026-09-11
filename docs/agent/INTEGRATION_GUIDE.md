# M4.6 Agent Integration Guide

**Version:** 1.0.0  
**Last Updated:** 2026-09-11

---

## Overview

The Agent-Facing Interface (M4.6) provides external agents with a stable, versioned API to:
- **Validate** route configurations
- **Test** routes with fixtures
- **Scaffold** new domains
- **Query** lineage and provenance
- **Discover** capabilities and supported adapters

This guide explains how to integrate an agent with the published interface.

---

## Prerequisites

**For the Agent:**
- Go 1.19+ (if building on top of dim)
- Python 3.9+ (if using reference agent)
- MCP library for your language/platform
- Access to dim workspace (local or remote)

**For the Server:**
- Running `dimctl` agent server (see [Starting the Server](#starting-the-server))
- Port 9090 available (configurable)
- dim repository with routes to validate

---

## Starting the Server

The MCP server is built into `dimctl`. To start it:

```bash
cd /path/to/dim
dimctl agent-server --port 9090
```

**Output:**
```
2026-09-11T08:20:00Z [INFO] Agent Interface Server v1.0.0 ready
2026-09-11T08:20:00Z [INFO] Listening on http://localhost:9090
2026-09-11T08:20:00Z [INFO] Operations: 6
  - validate_route
  - test_route
  - scaffold_domain
  - get_lineage
  - get_provenance
  - get_capabilities
```

---

## Connecting to the Interface

### Python (Reference Implementation)

```python
from examples.agent.route_validator import AgentMCPClient, RouteValidator

# Create client
client = AgentMCPClient(endpoint="http://localhost:9090")
agent = RouteValidator(client)

# Verify version compatibility
assert client.interface_version == "1.0.0", "Interface version mismatch"

# Use agent
agent.validate_route("domains/payments/order-payment.yaml")
agent.test_route("domains/payments/order-payment.yaml", "fixtures.yaml")
```

### Go

```go
import "github.com/naren-chakraview/dim/internal/agent"

// Create server (for hosted agents)
server := agent.NewMCPServer()

// Call operation
ctx := context.Background()
result := server.CallOperation(ctx, "validate_route", rawRequest)

// Check interface version
if result.InterfaceVersion != agent.AgentInterfaceVersion {
    log.Fatal("Version mismatch")
}
```

### Pseudocode (Other Languages)

```python
# 1. Send JSON-RPC request to server
request = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "validate_route",
    "params": {
        "route_config_path": "domains/payments/order-payment.yaml",
        "strict_mode": false
    }
}

# 2. Parse response envelope
response = {
    "interface_version": "1.0.0",
    "timestamp": "2026-09-11T08:20:30Z",
    "result": {
        "valid": true,
        "errors": [],
        "warnings": [],
        "route_version": "sha256:abc123"
    }
}

# 3. Check version first
assert response["interface_version"] == "1.0.0"

# 4. Check for errors
if response.get("error"):
    handle_error(response["error"])
else:
    handle_result(response["result"])
```

---

## Operations Reference

### 1. validate_route

**Purpose:** Validate a route configuration against schema and governance rules.

**Request:**
```json
{
  "route_config_path": "domains/payments/order-payment.yaml",
  "strict_mode": false
}
```

**Success Response:**
```json
{
  "interface_version": "1.0.0",
  "result": {
    "valid": true,
    "errors": [],
    "warnings": [],
    "route_version": "sha256:abc123def456"
  }
}
```

**Error Response:**
```json
{
  "interface_version": "1.0.0",
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Schema validation failed",
    "details": {
      "errors": [
        {
          "code": "REQUIRED_FIELD",
          "path": "routes.order-payment.from",
          "message": "'from' field is required"
        }
      ]
    }
  }
}
```

---

### 2. test_route

**Purpose:** Run fixtures against a route.

**Request:**
```json
{
  "route_config_path": "domains/payments/order-payment.yaml",
  "fixtures_path": "domains/payments/fixtures.yaml",
  "timeout_ms": 30000
}
```

**Success Response:**
```json
{
  "interface_version": "1.0.0",
  "result": {
    "passed": true,
    "test_results": [
      {
        "name": "test_order_ingestion",
        "passed": true,
        "duration_ms": 245
      }
    ],
    "summary": {
      "total": 1,
      "passed": 1,
      "failed": 0,
      "skipped": 0,
      "duration_ms": 245
    }
  }
}
```

---

### 3. scaffold_domain

**Purpose:** Generate a domain directory with starter routes.

**Request:**
```json
{
  "domain": "payments",
  "source_type": "http",
  "sink_type": "file",
  "with_contract": false,
  "template": "passthrough"
}
```

**Response:**
```json
{
  "interface_version": "1.0.0",
  "result": {
    "success": true,
    "domain_path": "domains/payments",
    "files_created": [
      "domains/payments/DOMAIN.yaml",
      "domains/payments/payments-route.yaml"
    ],
    "next_steps": [
      "Review the generated files",
      "Run 'dimctl validate' to verify",
      "Commit to git"
    ]
  }
}
```

---

### 4. get_lineage

**Purpose:** Query message lineage by message ID.

**Request:**
```json
{
  "message_id": "msg-12345-abcde"
}
```

**Response:**
```json
{
  "interface_version": "1.0.0",
  "result": {
    "found": true,
    "lineage_records": [
      {
        "id": "lineage-001",
        "message_id": "msg-12345-abcde",
        "route_name": "order-payment",
        "domain": "payments",
        "created_at": "2026-09-11T08:20:00Z",
        "step": "translate-step",
        "status": "success"
      }
    ]
  }
}
```

---

### 5. get_provenance

**Purpose:** Query provenance chain by subject ID.

**Request:**
```json
{
  "subject_id": "order-12345"
}
```

**Response:**
```json
{
  "interface_version": "1.0.0",
  "result": {
    "found": true,
    "provenance_chain": [
      {
        "id": "prov-001",
        "subject_id": "order-12345",
        "message_id": "msg-001",
        "route_name": "order-payment",
        "created_at": "2026-09-11T08:20:00Z",
        "status": "success"
      }
    ]
  }
}
```

---

### 6. get_capabilities (M4.7)

**Purpose:** Discover all supported adapters, steps, and EIPs.

**Request:**
```json
{
  "filter_type": "adapter"
}
```

**Response:**
```json
{
  "interface_version": "1.0.0",
  "result": {
    "capabilities": [
      {
        "name": "kafka-source",
        "type": "adapter",
        "config_schema": { ... },
        "description": "Kafka message source",
        "example": { ... }
      }
    ],
    "schema_version": "1.0.0"
  }
}
```

---

## Error Handling

Every operation can return an error. Check `error` field first:

```python
def call_operation(operation: str, request: dict) -> dict:
    response = client.call_operation(operation, request)
    
    # Always check version first
    if response["interface_version"] != "1.0.0":
        raise VersionMismatch(f"Got {response['interface_version']}")
    
    # Then check for error
    if response.get("error"):
        error = response["error"]
        code = error["code"]
        message = error["message"]
        
        # Handle error based on code
        if code == "VALIDATION_FAILED":
            # Route config is invalid
            pass
        elif code == "FILE_NOT_FOUND":
            # Route config file doesn't exist
            pass
        elif code == "VERSION_MISMATCH":
            # Interface version incompatible
            raise IncompatibleInterface(message)
        else:
            # Generic error
            raise OperationError(f"{code}: {message}")
    
    # Success: use result
    return response["result"]
```

---

## Best Practices

### 1. Always Check Version First

```python
if response["interface_version"] != agent.EXPECTED_VERSION:
    raise IncompatibleInterface()
```

### 2. Handle Errors Gracefully

```python
if response.get("error"):
    log_error(response["error"])
    return None
```

### 3. Parse Structured Responses

```python
# Good: use typed fields
valid = response["result"]["valid"]
errors = response["result"]["errors"]

# Bad: parse error messages with regex
```

### 4. Don't Parse CLI Output

```python
# Bad: what agents CANNOT do
output = subprocess.run(["dimctl", "validate", ...]).stdout
lines = output.split("\n")  # Fragile!

# Good: what agents SHOULD do
response = client.call_operation("validate_route", {...})
valid = response["result"]["valid"]
```

### 5. Respect Timeout Parameters

```python
# Default timeout is 30s; increase for large test suites
response = client.call_operation("test_route", {
    "route_config_path": "...",
    "fixtures_path": "...",
    "timeout_ms": 60000  # 60 seconds for large tests
})
```

---

## Deprecation Handling

Agents should gracefully handle deprecation notices:

```python
if response.get("deprecation_notice"):
    log_warning(response["deprecation_notice"])
    # Continue using operation, but plan for migration
```

When an operation is deprecated:
1. You'll see `deprecation_notice` in responses
2. Operation continues to work (backward compatible)
3. 6-month notice period before removal
4. When removed, `interface_version` bumps to `2.0.0`

See [VERSIONING.md](VERSIONING.md) for details.

---

## Example: Complete Agent Workflow

```python
#!/usr/bin/env python3
from route_validator import AgentMCPClient, RouteValidator

def main():
    # Connect
    client = AgentMCPClient()
    agent = RouteValidator(client)
    
    config_path = "domains/payments/order-payment.yaml"
    fixtures_path = "domains/payments/fixtures.yaml"
    
    # Step 1: Validate
    if not agent.validate_route(config_path, strict=True):
        print("❌ Validation failed")
        return 1
    
    # Step 2: Test
    if not agent.test_route(config_path, fixtures_path):
        print("❌ Tests failed")
        return 1
    
    # Step 3: Report success
    print("✅ Route is valid and all tests passed")
    print("   Engineer can now create a PR")
    return 0

if __name__ == "__main__":
    exit(main())
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Connection refused | Check server is running: `dimctl agent-server --port 9090` |
| Version mismatch | Agent expects v1.0.0; server might be newer. See VERSIONING.md |
| Timeout on test_route | Increase `timeout_ms` parameter; large test suites need more time |
| File not found | Ensure `route_config_path` is relative to repository root |
| Schema validation error | Route config doesn't match JSON schema; check YAML syntax and required fields |

---

## Next Steps

- Read [VERSIONING.md](VERSIONING.md) for version compatibility strategy
- Review [worked_example.md](worked_example.md) for real-world usage
- Check [M4.6 specification](../design/m46-agent-interface-spec.md) for complete API details
- See examples/agent/route_validator.py for reference implementation
