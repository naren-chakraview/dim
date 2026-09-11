# M4.6 Transport Decision: Agent-Facing Interface

**Decision Date:** 2026-09-11  
**Status:** DECIDED  
**Version:** 1.0

---

## Executive Summary

**Chosen Transport:** Model Context Protocol (MCP)

**Rationale:** MCP is the first-class integration point for Claude-family agents and aligns with the evolving agent ecosystem. Provides native tool-use semantics, requires minimal scaffolding, and enables future interoperability with other MCP-aware tools.

**Secondary Consideration:** REST/JSON-RPC for non-MCP clients can be added later without breaking changes to the core interface design.

---

## Target Agent Audience

### Primary (MCP-first)
- **Claude API agents** — coding agents, architectural agents, design assistants running within Claude ecosystem
- **MCP-aware clients** — any tool/platform adopting MCP as agent tooling standard (Codeium, future ecosystem partners)
- **Use cases:** Route validation, testing, scaffolding, and lineage queries initiated by Claude or MCP clients

### Secondary (future consideration)
- **CI/CD agents** — local agents running in GitHub Actions, GitLab CI, etc., preferring HTTP
- **Hosted agent platforms** — cloud-based agents needing HTTP/REST API
- **Non-Claude LLM agents** — if/when other LLM platforms adopt agent-tooling standards

---

## Transport Evaluation

### Option 1: MCP Only

**Model Context Protocol — native Claude integration**

| Aspect | Details |
|--------|---------|
| **Adoption** | Native to Claude agents; MCP is becoming ecosystem standard for agent tooling |
| **Developer Experience** | Agents call operations as native tools; no parsing, no auth ceremony |
| **Schema** | JSON Schema for request/response types built-in; generation from Go types straightforward |
| **Streaming** | Server-Sent Events (SSE) support for long-running operations (testing, lineage queries) |
| **Security** | Process-local or HTTP transport; auth handled by parent process |
| **Maintenance** | Single transport implementation to maintain; smaller surface |
| **Dependencies** | Anthropic MCP SDK (Go) — lightweight, purpose-built |
| **Interop Future** | MCP is ecosystem-standard; non-Claude agents adding MCP support already underway |
| **Fit for Phase 4** | Strong — matches Claude agent-first strategy of Phase 4; no need to guess at REST versioning or auth when MCP is already designed for this |

**Verdict:** Strongly preferred for primary use case.

---

### Option 2: REST/JSON-RPC Only

**HTTP-based interface — universal, language-agnostic**

| Aspect | Details |
|--------|---------|
| **Adoption** | Universal; every platform and language speaks HTTP |
| **Developer Experience** | Well-understood; curl, Postman, standard tools work; agents must parse JSON responses |
| **Schema** | OpenAPI/JSON Schema; needs explicit versioning strategy |
| **Streaming** | Chunked HTTP for streaming; more complex for long operations |
| **Security** | Auth strategy must be designed (API key? JWT? OAuth?); non-trivial for a tool interface |
| **Maintenance** | Dual burden: HTTP server + auth + versioning logic; more moving parts |
| **Dependencies** | Standard Go `net/http` or Echo/Gin; ecosystem mature but no purpose-built agent semantics |
| **Interop Future** | Widely supported but no special semantics for agent tooling; agents still parse responses |
| **Fit for Phase 4** | Weaker — adds auth/versioning ceremony not needed for Claude agents; makes sense as *secondary* option, not primary |

**Verdict:** Reserve as secondary option for future (non-Claude) agent platforms, if/when they need it.

---

### Option 3: Both (MCP + REST)

**Dual transport — maximum compatibility, minimum clarity**

| Aspect | Details |
|--------|---------|
| **Adoption** | Serves both Claude agents (MCP) and general platforms (REST) |
| **Developer Experience** | Two different experiences; agents see different tool signatures, response formats, error handling |
| **Maintenance Burden** | Dual implementation; must keep behavior identical across transports; risk of divergence |
| **Complexity** | Higher implementation complexity upfront; "both" often becomes "one works well, one gets stale" |
| **Future Clarity** | Commits too early to REST before knowing if we actually need it |
| **Fit for Phase 4** | Not recommended as starting point — violates "spike first, commit second" principle established in earlier phases |

**Verdict:** Deferred. Start with MCP, add REST support explicitly if/when ecosystem demands it.

---

## Comparison Table

| Criterion | MCP Only | REST Only | Both |
|-----------|----------|-----------|------|
| Claude Agent Fit | ✅ Native | ⚠️ Parse output | ✅ Native (via MCP only) |
| Security/Auth | ✅ Process-local | ⚠️ Design needed | ⚠️ Design needed for REST |
| Maintenance | ✅ Single path | ✅ Single path | ❌ Dual paths |
| Ecosystem Alignment | ✅ MCP standard | ⚠️ Generic HTTP | ⚠️ Hedging bet |
| Phase 4 Timing | ✅ Ship fast | ❌ Design delay | ❌ Over-engineer |
| Future Extensibility | ✅ MCP → REST later | ❌ Hard to remove HTTP | ✅ Can add REST later |

---

## Decision

**Primary Transport:** Model Context Protocol (MCP)

**Rationale:**
1. **Claude agents are the primary audience** — MCP is native integration point
2. **No premature commitment to REST** — MCP is sufficient for Phase 4; if REST is needed later, it can be added as secondary without breaking MCP interface
3. **Simpler implementation** — MCP already handles versioning, schema, streaming; REST would add auth/versioning design burden with unclear payoff
4. **Ecosystem alignment** — MCP is becoming standard for agent tooling; starting there positions dim as interoperable with future agents
5. **Phase 4 principle** — "Spike first, commit second"; we've spiked (this doc), now commit to MCP

---

## Implementation Roadmap

### Phase 1 (M4.6): MCP Interface
- Implement core operations (Validate, Test, Scaffold, Lineage queries) as MCP tools
- Versioning: `AgentInterfaceVersion = "1.0.0"` (independent from dimctl CLI version)
- Reference agent: Claude-based example validating routes via MCP
- Exit: External agent can call all operations via MCP only

### Phase 2 (M4.7-M4.9): MCP-only Track C
- Capability Manifest (M4.7): Exposed via MCP as queryable tool
- Agent-Assisted Design (M4.8): Claude agent scaffolding routes via MCP
- Impact Analysis (M4.9): Query service for cross-route references via MCP

### Phase 3 (Future, if needed): REST as Secondary
- **Only if:** Non-Claude agent platforms explicitly request HTTP API  
- **Implementation:** REST server wrapping same MCP operations (no code duplication)
- **Versioning:** Same `AgentInterfaceVersion`; REST clients query via same mechanism

---

## Design Principles (for all transports)

These principles apply to MCP now; if REST is added later, they must hold there too:

1. **No Internal Access** — Agents bind against published interface only; no `internal/` imports
2. **Deterministic Feedback** — Operations return structured types, never CLI output
3. **Version Independence** — Interface versioned separately from dimctl CLI (can update dimctl without breaking agents)
4. **Identical Enforcement** — Every agent operation runs the same `dimctl validate`/`test` path a human would
5. **Generated, Not Duplicated** — Interface schema derived from source of truth (operation definitions), CI-verified to stay in sync

---

## Dependencies

### MCP SDK for Go
- Package: `github.com/anthropics/mcp-go` (or equivalent published SDK)
- Version: Latest stable
- Scope: MCP server implementation, schema marshaling, versioning utilities

### No external REST framework required for Phase 1
- Implementation uses Go `net/http` only if REST added later

---

## Open Questions Resolved

| Question | Answer |
|----------|--------|
| Which agent frameworks? | Claude API agents (primary); MCP-aware clients (secondary); reserve REST for non-Claude platforms |
| MCP vs REST? | MCP as primary; REST deferred to if/when ecosystem demands it |
| How to version? | Independent `AgentInterfaceVersion` (e.g., 1.0.0); separate from dimctl CLI version |
| How to handle breaking changes? | Deprecation policy: 6-month notice before removal; document in VERSIONING.md |
| Single or dual transport? | Single (MCP) for Phase 1; dual only if/when data shows REST necessity |

---

## Next Steps

1. ✅ **Decision Doc (this)** — Accepted by team  
2. **Interface Spec (Task 2)** — Define operations, types, error handling based on MCP transport  
3. **MCP Implementation (Task 3)** — Build `internal/agent/mcp_server.go` and operation handlers  
4. **Conformance (Task 4)** — Reference agent (Claude-based) using MCP only  
5. **Documentation (Task 5)** — Integration guide, versioning policy, worked examples

---

**Approved:** 2026-09-11  
**Prepared By:** Claude (Haiku 4.5)  
**Review:** Ready for implementation
