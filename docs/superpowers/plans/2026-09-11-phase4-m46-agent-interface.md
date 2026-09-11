# Phase 4, M4.6 — Agent-Facing Operation Interface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a published, versioned interface that external agents (coding agents, architectural agents, design assistants) can use to validate, test, scaffold, and query routes — without accessing internal packages or parsing CLI output. Agents use the same operations humans do, with identical validation and enforcement.

**Architecture:** Published agent-facing interface wrapping existing `dimctl` capabilities:
1. Spike: Determine transport (MCP vs REST/JSON-RPC vs both)
2. Interface design: Operations for validate, test, scaffold, lineage queries
3. Version independently (like `pkg/sdk`'s `SDKVersion`)
4. Conformance: Real agent accessing only the published interface

**Tech Stack:** Go, existing dimctl infrastructure, published API boundary, optional MCP or REST transport.

**Spec:** `design/phase-4-implementation-plan.md` (M4.6, §6.3), `eip-middleware-design.md` v11 (§1.1 — no shortcut for agents).

---

## Global Constraints

- **No agent shortcuts:** Agents use the same `dimctl validate`/`test` paths humans do — no separate, less-governed code path
- **Generated, not duplicated:** Interface schema and version must not drift from source of truth (internal operation definitions)
- **Stable versioning:** Once published, breaking changes are expensive — version independently, communicate deprecation path
- **Deterministic feedback:** Operations return structured, parseable results — no CLI output parsing from agents
- **No internal access:** Agents bind against published interface only, same discipline as `pkg/sdk` in Phase 3

---

## Task Breakdown

### Task 1: Transport Spike & Decision

**Deliverable:** Short decision doc recommending transport + implementation roadmap

**Steps:**

- [ ] **Step 1: Research current agent ecosystem**

Investigate which agent frameworks/products are likely consumers:
- Claude API agents (native tool use)
- MCP-connected agents (Anthropic/Codeium/other MCP clients)
- REST-first workflows (local CI agents, hosted platforms)
- Determine which 2-3 are actual target audience

- [ ] **Step 2: Evaluate transport options**

For each option, document:
- **MCP (Model Context Protocol):**
  - Pros: Native Claude integration, no separate auth/routing layer, SSE transport built-in
  - Cons: Limited to MCP-aware clients, requires MCP SDK
  - Fit for: Claude agents, MCP ecosystem
  
- **REST/JSON-RPC:**
  - Pros: Universal, language-agnostic, easy to test with curl
  - Cons: Requires auth/versioning strategy, HTTP overhead
  - Fit for: Polyglot agents, hosted platforms, CI integration
  
- **Both (MCP + REST):**
  - Pros: Maximum compatibility
  - Cons: Dual maintenance burden, must keep in sync
  - Fit for: If both audiences are clearly identified

- [ ] **Step 3: Document recommendation**

Write decision doc with:
- Target agent audience (who we're actually building for)
- Chosen transport(s) and rationale
- API design principles (naming, versioning, error handling)
- Implementation phasing (MCP first? REST first? Parallel?)

- [ ] **Step 4: Commit decision doc**

```bash
git add docs/design/m46-transport-decision.md
git commit -m "docs: M4.6 transport decision and API design principles"
```

---

### Task 2: Define Agent-Facing Interface

**Deliverable:** Versioned interface spec with operations, types, and error handling

**Steps:**

- [ ] **Step 1: Define operation set**

Based on decision doc, define operations:

```
Validate(routeConfigPath: string, strictMode: bool) -> ValidationResult
Test(routeConfigPath: string, fixtureDir: string) -> TestResults
Scaffold(domain: string, source: string, sink: string, withContract: bool) -> RouteYAML
GetLineage(messageID: string) -> LineageChain
GetProvenance(subjectID: string) -> ProvenanceChain
GetCapabilities() -> CapabilityManifest  (for M4.7)
```

Each returns structured types, never raw CLI output.

- [ ] **Step 2: Define versioning strategy**

```go
type InterfaceVersion struct {
  Major int    // Breaking changes
  Minor int    // Backward-compatible additions
  Patch int    // Bug fixes
}

// Published constant
const AgentInterfaceVersion = "1.0.0"
```

Include version in every response.

- [ ] **Step 3: Define error handling**

```go
type OperationError struct {
  Code    string      // e.g., "VALIDATION_FAILED", "TEST_TIMEOUT"
  Message string      // Human-readable
  Details interface{} // Structured error details (validation errors, etc.)
}
```

- [ ] **Step 4: Write interface spec document**

```bash
git add docs/design/m46-agent-interface-spec.md
git commit -m "docs: M4.6 agent operation interface specification"
```

---

### Task 3: Implement Interface (Transport Layer)

**Deliverable:** Working interface accessible via chosen transport(s)

**Steps:**

- [ ] **Step 1: Implement core operation handlers**

Create `internal/agent/operations.go`:
- Each operation wraps existing dimctl/engine capability
- No duplicate logic — reuse `config.LoadRouteConfig`, `testing.RunFixtures`, etc.
- Return structured types, never side effects

- [ ] **Step 2: Implement transport layer**

**If MCP:**
- Create `internal/agent/mcp_server.go` using Anthropic's MCP SDK
- Expose operations as MCP tools
- Handle lifecycle (init, shutdown, etc.)

**If REST:**
- Create `cmd/dimctl/agent_server.go` with HTTP handlers
- JSON request/response marshaling
- Auth strategy (API key? JWT? Token from env?)

- [ ] **Step 3: Versioning middleware**

Add version to every response:
```go
type OperationResponse struct {
  InterfaceVersion string
  Timestamp        time.Time
  Result           interface{}
  Error            *OperationError
}
```

- [ ] **Step 4: Test transport layer**

Write tests verifying:
- Operations accessible via transport
- Versioning included in responses
- Error handling working
- No internal package exposure

- [ ] **Step 5: Commit**

```bash
git add internal/agent/operations.go [transport implementation]
git commit -m "feat: implement M4.6 agent-facing operation interface"
```

---

### Task 4: Conformance & Worked Example

**Deliverable:** Real agent (code example) using only published interface

**Steps:**

- [ ] **Step 1: Implement reference agent**

Build `examples/agent/route_validator.py` (or similar) that:
- Connects to published interface (MCP/REST)
- Calls Validate and Test operations
- Never imports `internal/`
- Demonstrates real agent use case

- [ ] **Step 2: Worked example scenario**

Document an end-to-end flow:
1. Agent loads unvalidated route config
2. Agent calls Validate operation
3. Agent calls Test operation
4. Agent queries Lineage for related routes
5. Agent reports results to user/CI

- [ ] **Step 3: Verify no internal access**

Audit reference agent:
- No `import "github.com/naren-chakraview/dim/internal/..."`
- No CLI parsing
- Only public types from published interface

- [ ] **Step 4: Integration test**

Test that reference agent works end-to-end:
```bash
# Start agent server
dimctl agent-server &

# Run reference agent against it
python examples/agent/route_validator.py --config domains/payments/order-payment.yaml

# Verify output matches expected structured format
```

- [ ] **Step 5: Commit**

```bash
git add examples/agent/route_validator.py docs/agent/worked_example.md
git commit -m "example: M4.6 reference agent and worked example"
```

---

### Task 5: Documentation & Exit Verification

**Deliverable:** Complete documentation of interface, versioning, and deprecation path

**Steps:**

- [ ] **Step 1: Write agent integration guide**

`docs/agent/INTEGRATION_GUIDE.md` covering:
- Connecting to the interface (MCP setup, REST endpoint)
- Calling operations
- Parsing responses
- Handling errors and versioning

- [ ] **Step 2: Write versioning & deprecation policy**

`docs/agent/VERSIONING.md` covering:
- How version numbers work
- What constitutes a breaking change
- Deprecation timeline (e.g., 6 months notice)
- How agents discover available versions

- [ ] **Step 3: Verify exit criteria**

Checklist:
- [ ] External agent can Validate via published interface only
- [ ] External agent can Test via published interface only
- [ ] External agent can Scaffold via published interface only
- [ ] External agent can query Lineage/Provenance via published interface only
- [ ] No `internal/` imports in agent code or interface
- [ ] Every agent operation runs identical `dimctl validate`/`test` to CLI equivalent
- [ ] Operations return structured types, not CLI output
- [ ] Interface versioned independently
- [ ] Worked example uses only published interface

- [ ] **Step 4: Final test suite**

```bash
go test ./internal/agent -v
# Verify: all operations accessible, no internal access leaks, versioning consistent
```

- [ ] **Step 5: Commit documentation**

```bash
git add docs/agent/INTEGRATION_GUIDE.md docs/agent/VERSIONING.md
git commit -m "docs: M4.6 agent integration guide and versioning policy"
```

---

## Exit Criteria

✅ An external agent can Validate a route using only published interface
✅ An external agent can Test a route using only published interface
✅ An external agent can Scaffold a domain using only published interface
✅ An external agent can query Lineage/Provenance using only published interface
✅ No `internal/` packages exposed to agents
✅ Every operation an agent invokes runs identical validation to CLI equivalent
✅ Interface versioned independently (e.g., 1.0.0)
✅ Worked example (reference agent) uses published interface exclusively
✅ Transport decision documented (MCP, REST, or both) with rationale
✅ Integration guide and versioning policy documented

---

## Summary

**M4.6 establishes the foundation** for all AI-agent consumability work (M4.7-M4.9). By publishing a stable, versioned interface wrapping existing `dimctl` operations, agents gain access to the same deterministic feedback loops humans do — validation, testing, scaffolding, and lineage queries — without shortcuts or internal access. Agents produce output indistinguishable from human-authored routes, subject to identical review and deployment gates.

**Next:** M4.7 (Capability Manifest) — enumerate every step/adapter/EIP with its config schema, generated from schema, CI-verified to never drift
