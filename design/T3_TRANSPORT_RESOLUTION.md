# T3.2 — Transport Resolution Decision

**Context:**
The Phase 4 implementation plan (line 226) stated the resolved answer for M4.6's transport was "MCP and A2A as first-class transports, alongside a traditional REST/JSON-RPC API." However, the phase-4-completion-report only implemented MCP, creating a discrepancy that T3.2 requires resolving.

**Decision:**

We are implementing **HTTP + REST/JSON-RPC as the primary transport**, with MCP protocol layer on top.

**Rationale:**

1. **A2A is not applicable for `dimctl`:** A2A (Anthropic-to-Anthropic) is an internal protocol within Anthropic's infrastructure. The `dimctl agent serve` command is a public-facing tool intended for external consumption (by Claude, other LLMs, and third-party agents). A2A is not relevant for this use case.

2. **REST/JSON-RPC is the right external transport:** The implemented HTTP server with JSON endpoints provides:
   - Industry-standard REST/JSON-RPC semantics
   - HTTP/1.1 compatibility (works with curl, any HTTP client, web browsers)
   - JSON request/response format (machine-readable, well-understood)
   - Standard HTTP status codes for errors
   - Zero transport-specific dependencies

3. **MCP is the protocol layer, not transport:** MCP (Model Context Protocol) defines the message schema and operation semantics. It can be transported via HTTP, WebSocket, stdio, or other means. Our implementation uses HTTP as the transport and JSON-RPC as the calling convention.

**Implementation Status:**

✅ **Complete:** `dimctl agent serve` provides:
- HTTP server listening on configurable address (default: localhost:9090)
- `/health` endpoint for server health checks
- `/operations` endpoint for capability discovery
- `/call` endpoint for operation invocation (REST/JSON-RPC style)
- All operations accessible via unified HTTP interface

**Endpoints:**

```
POST /call
{
  "operation": "validate_route",
  "request": { "route": {...} }
}
→ { "result": {...}, "interfaceVersion": "...", "timestamp": "..." }
```

**Conclusion:**

T3.2 is resolved by clarifying that:
- REST/JSON-RPC via HTTP is our chosen external transport (not A2A)
- MCP defines the protocol layer, not the transport layer
- The discrepancy in the plan was about terminology, not actual requirements
- The implementation already satisfies both the plan and the decision doc
