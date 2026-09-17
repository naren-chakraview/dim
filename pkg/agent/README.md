# pkg/agent — Published Agent Interface Contract

This package defines the stable, public contract for AI agents building on dim.

## Why pkg/agent?

Following Phase 3's pattern (`pkg/sdk`), the agent interface is published at the package level so:

1. **External parties can depend on it** — The import path `github.com/naren-chakraview/dim/pkg/agent` is stable
2. **Versioning is independent** — `InterfaceVersion` evolves separately from the engine
3. **No internal coupling** — Agents never import `internal/`; they only import this public contract
4. **Documented contract** — Types and constants are public, with clear semver semantics

## Using the Interface

### Check Compatibility

```go
import "github.com/naren-chakraview/dim/pkg/agent"

// Ensure the agent interface supports your client
if agent.InterfaceVersion != "1.0.0" {
    log.Fatalf("unsupported agent interface version: %s", agent.InterfaceVersion)
}
```

### Call Operations

The agent interface is accessed via HTTP:

```bash
# Start the server
dimctl agent serve --addr localhost:9090

# Call an operation (in another terminal)
curl -X POST http://localhost:9090/call \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "validate_route",
    "request": {
      "route_config_path": "domains/payments/order-payment.yaml"
    }
  }'
```

### Implement a Client

See `examples/agent/route_validator.py` for a reference Python client that:
- Imports only the public interface version
- Calls operations via HTTP/JSON-RPC
- Never touches internal packages
- Is entirely external to the dim codebase

## Stability Guarantees

- **InterfaceVersion** is the canonical version number. Check it at runtime.
- **Operation names and request/response shapes** are backwards-compatible within a major version
- **New operations** increment the minor version (non-breaking addition)
- **Breaking changes** increment the major version

## Extending the Interface

To add a new operation:

1. Define its request/response types in `internal/agent/types.go`
2. Register it in `internal/agent/mcp_server.go:registerOperations()`
3. Implement the handler
4. Update the documentation
5. If backwards-compatible: bump minor version
6. If breaking: bump major version (rarely done)
