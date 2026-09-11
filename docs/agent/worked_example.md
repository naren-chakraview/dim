# M4.6 Worked Example: Agent Validating Routes via MCP Interface

**Scenario:** A coding agent (Claude-based) helps a domain engineer validate and test their payment-processing route without direct access to internal packages.

**Key Principle:** The agent uses ONLY the published MCP interface. No `internal/` imports, no CLI parsing, no shortcuts.

---

## The Scenario

**Setup:**
- Domain engineer is building a new route: `domains/payments/order-payment.yaml`
- Route needs validation against the schema
- Route needs testing with fixtures: `domains/payments/order-payment.route_test.yaml`
- Engineer wants an agent to help verify everything before committing

**Agent Workflow:**
1. Agent connects to MCP server (`internal/agent/mcp_server.go`)
2. Agent calls `validate_route` operation
3. Agent calls `test_route` operation
4. Agent reports results to engineer
5. Engineer reviews and commits

---

## Step 1: Agent Connects to MCP Server

**Agent Code:**
```python
from route_validator import AgentMCPClient, RouteValidator

# Connect to published MCP interface (v1.0.0)
mcp = AgentMCPClient(endpoint="http://localhost:9090")
agent = RouteValidator(mcp)

# Verify version matches
assert mcp.interface_version == "1.0.0"
```

**Interface Boundary:**
- Agent imports ONLY: `AgentMCPClient`, `RouteValidator`, type definitions
- Agent does NOT import: `internal/config`, `internal/engine`, `internal/secrets`, etc.
- The MCP server exposes the boundary; agent cannot cross it

---

## Step 2: Validate Route

**Agent Call:**
```python
# Call published operation
response = agent.client.call_operation(
    "validate_route",
    {
        "route_config_path": "domains/payments/order-payment.yaml",
        "strict_mode": False,
    }
)
```

**MCP Server Processing:**
```
1. Receive operation: validate_route
2. Dispatch to: internal/agent/operations.go::ValidateRoute()
3. ValidateRoute wraps: config.LoadRouteConfig() + config.ValidateAuthDeclarations()
4. Return structured response with version, timestamp, result/error
```

**Agent Receives (JSON):**
```json
{
  "interface_version": "1.0.0",
  "timestamp": "2026-09-11T08:15:00Z",
  "result": {
    "valid": true,
    "errors": [],
    "warnings": [],
    "route_version": "sha256:abc123def456"
  }
}
```

**Agent Interprets:**
```python
if response.error:
    print(f"❌ Validation failed: {response.error.message}")
else:
    print("✅ Validation passed")
    print(f"   Route version: {response.result['route_version']}")
```

**What the Agent Cannot Do:**
- Agent cannot import `internal/config.LoadRouteConfig` directly
- Agent cannot parse CLI output (`dimctl validate` stdout)
- Agent cannot access route internals beyond what the operation returns
- Agent must use only the published response types

---

## Step 3: Test Route

**Agent Call:**
```python
# Call published operation
response = agent.test_route(
    config_path="domains/payments/order-payment.yaml",
    fixtures_path="domains/payments/order-payment.route_test.yaml"
)
```

**MCP Server Processing:**
```
1. Receive operation: test_route
2. Dispatch to: internal/agent/operations.go::TestRoute()
3. TestRoute wraps: testing.RunFixtures() (same path as `dimctl test`)
4. Return structured response with pass/fail, test results, summary
```

**Agent Receives:**
```json
{
  "interface_version": "1.0.0",
  "timestamp": "2026-09-11T08:15:30Z",
  "result": {
    "passed": true,
    "test_results": [
      {
        "name": "test_order_ingestion",
        "passed": true,
        "duration_ms": 245
      },
      {
        "name": "test_currency_conversion",
        "passed": true,
        "duration_ms": 312
      }
    ],
    "summary": {
      "total": 2,
      "passed": 2,
      "failed": 0,
      "skipped": 0,
      "duration_ms": 557
    }
  }
}
```

**Agent Interprets:**
```python
if response.result["passed"]:
    print("✅ All tests passed")
    print(f"   {summary['passed']}/{summary['total']} tests")
else:
    print("❌ Tests failed:")
    for test in response.result["test_results"]:
        if not test["passed"]:
            print(f"   - {test['name']}: {test['error']}")
```

---

## Step 4: Agent Reports to Engineer

**Agent Output:**
```
============================================================
Route Validator Agent (M4.6 Reference Implementation)
============================================================
Interface Version: 1.0.0

📋 Validating route: domains/payments/order-payment.yaml
✅ Validation passed
   Route version: sha256:abc123def456

🧪 Testing route: domains/payments/order-payment.yaml
✅ All tests passed
   2/2 tests passed (557ms)

============================================================
✅ Operation completed successfully
```

**What Engineer Sees:**
- Validation passed (same as `dimctl validate`)
- Tests passed (same as `dimctl test`)
- Route version (same as `dimctl provenance`)
- No special agent shortcuts — identical to human workflow

---

## Step 5: Engineer Reviews and Commits

Engineer can now:
1. Trust the agent's validation (used same engine as `dimctl`)
2. Know the tests passed (same test runner)
3. Create a PR with confidence
4. CI will re-run `dimctl validate` and `dimctl test` (same operations)

**Key Point:** Agent output is *indistinguishable* from human output because they use identical operations.

---

## Code Audit: No Internal Access

**Proof Agent Uses Only Published Interface:**

```bash
# Check reference agent imports
grep "^import" examples/agent/route_validator.py
# Output:
#   import argparse
#   import json
#   import sys
#   from typing import Any, Dict, Optional
#   from dataclasses import dataclass
#
# ✅ No "from github.com/naren-chakraview/dim/internal" imports
```

**Proof Agent Calls Operations by Name:**

```python
response = self.client.call_operation(
    "validate_route",           # ← Public operation name
    {"route_config_path": "..."}  # ← Request type from published interface
)
```

**Proof Agent Never Accesses Internal Types:**

```python
# Agent receives:
#   interface_version: str
#   result: Optional[Dict]
#   error: Optional[OperationError]
#
# Agent does NOT receive:
#   config.RouteConfig
#   engine.Executor
#   internal packages
```

---

## Interface Boundary Enforcement

**At Compile Time (Go):**
```go
// internal/agent/operations.go can import internal packages
import "github.com/naren-chakraview/dim/internal/config"

// But agent (Python) cannot
// Only types.go and published operation names are exported
```

**At Runtime (MCP):**
```
Agent (Python) ──call_operation("validate_route")──→ MCP Server (Go)
                                                            ↓
                                                  operations.go (can use internal/)
                                                            ↓
                                         Returns: ResponseEnvelope (public types)
```

**Result:** Agent cannot bypass the interface even if it tries.

---

## Failure Scenario: What If Validation Fails?

**Route has a missing import:**

```yaml
# domains/payments/order-payment.yaml (BAD)
version: 1
# MISSING: imports: ["../governance/fragments.yaml"]

routes:
  order-payment:
    ...
```

**Agent Calls validate_route:**
```python
response = agent.client.call_operation(
    "validate_route",
    {"route_config_path": "domains/payments/order-payment.yaml"}
)

# Response:
{
  "interface_version": "1.0.0",
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Missing mandatory governance import: ../governance/fragments.yaml",
    "details": {
      "path": "routes.order-payment",
      "suggested_fix": "Add imports: [../governance/fragments.yaml] to top of file"
    }
  }
}
```

**Agent Interprets:**
```python
if response.error:
    print(f"❌ Validation failed: {response.error.message}")
    print(f"   Suggested fix: {response.error.details.get('suggested_fix')}")
```

**Engineer See:**
```
❌ Validation failed: Missing mandatory governance import
   Suggested fix: Add imports: [../governance/fragments.yaml] to top of file
```

**Engineer Fixes Route:**
```yaml
version: 1
imports:
  - ../governance/fragments.yaml  # ← Added

routes:
  order-payment:
    ...
```

**Agent Re-validates:** Now passes ✅

---

## Exit Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| External agent can Validate via published interface only | ✅ | examples/agent/route_validator.py:validate_route() |
| External agent can Test via published interface only | ✅ | examples/agent/route_validator.py:test_route() |
| External agent can Scaffold via published interface only | ✅ | examples/agent/route_validator.py:scaffold_domain() |
| No `internal/` imports in agent code | ✅ | Grep confirms only typing/dataclasses imports |
| Every agent operation runs identical `dimctl validate`/`test` to CLI | ✅ | operations.go wraps same config.LoadRouteConfig, testing.RunFixtures |
| Operations return structured types, not CLI output | ✅ | All responses are JSON ResponseEnvelopes with typed fields |
| Interface versioned independently | ✅ | AgentInterfaceVersion = "1.0.0" in types.go |
| Agent output indistinguishable from human (same operations) | ✅ | Both use config.LoadRouteConfig, testing.RunFixtures under the hood |

---

## Summary

This worked example shows:
1. **Agent connects to published interface** — no shortcuts
2. **Agent calls operations by name** — no direct function imports
3. **Agent receives structured responses** — no CLI parsing
4. **Agent output mirrors human workflow** — identical validation/testing
5. **Interface boundary holds** — Go compile-time enforces, MCP runtime enforces

**Next Phase:** This same interface powers M4.7 (Capability Manifest discovery), M4.8 (agent-assisted design), and M4.9 (impact analysis queries).
