# Phase 4 Features — Agent-Assisted Route Design & Impact Analysis

**Status:** ✅ Complete (v0.9.0-beta)  
**Released:** 2026-09-14  
**Milestones:** M4.1–M4.9 (all implemented and tested)

## Overview

Phase 4 introduces **agent-assisted design and governance** capabilities. Agents can now design routes from natural language, critique them for best practices, analyze cross-route impact, and discover capabilities—all through a standard interface (MCP).

Every agent-produced route passes the same validation as human-authored routes. Humans remain in the review/merge loop; no automatic deployment.

---

## What's New

### M4.1: GitOps Deployment Pipeline ✅
Routes flow from Git to running instances through a declarative CI/CD pipeline.

### M4.2: dimctl Scaffold Command ✅
```bash
./dimctl scaffold --template passthrough --domain payments
```
Generates route templates for three shapes: passthrough, transform, contract-enforced.

### M4.3: Pre-Deployment Discovery ✅
```bash
./dimctl discover routes --output json
./dimctl discover adapters --type sink
./dimctl discover contracts --filter "payment*"
```
Query routes, adapters, contracts, sources, and sinks without grep.

### M4.4: Domain-Scoped Secrets ✅
Secrets stored per-domain, rotated independently:
```yaml
routes:
  - name: process-payment
    sources:
      - type: kafka
        brokers: ${SECRET:kafka-brokers}
        saslPassword: ${SECRET:kafka-sasl-password}
```

### M4.5: Visual Route-Authoring (Studio) ✅
Browser-based drag-drop pipeline builder with real-time validation and YAML export.

### M4.6: Agent-Facing MCP Interface ✅
Seven operations for agent integration:
- `validate_route` — validate YAML against schema
- `test_route` — run route through test fixtures
- `scaffold_from_intent` — intent → YAML template
- `critique_route` — analyze route for best practices
- `query_impact` — what routes/sinks reference this contract/sink?
- `get_capabilities` — enumerate all adapters, steps, and models
- `query_lineage` — cross-route lineage queries

### M4.7: Machine-Readable Capability Manifest ✅
Auto-generated from schemas, CI-verified to never drift:
```bash
./dimctl get-capabilities --format json > capability-manifest.json
```
Includes all adapters (source/sink), steps, EIPs, and their configuration schemas.

### M4.8: Agent-Assisted Route Design ✅

#### Scaffolding
Agent writes intent:
```
"HTTP source, validate payment with payment-schema contract, translate 
amount to cents, route to Kafka payments topic or DLQ on error"
```

Output: Full route YAML with placeholders for human review.

#### Critique
Agent analyzes routes for patterns:
- Redundant translate steps
- Missing error paths
- Contract enforcement gaps
- Unused imports
- Authorization declaration omissions

### M4.9: Impact Analysis Query Surface ✅

**Query:** What routes are affected if we bump the payment contract?
```bash
./dimctl query-impact contract payment-contract --version 2.0
```

**Response:**
```json
{
  "change_type": "contract",
  "change_name": "payment-contract",
  "affected_routes": ["process-payment", "validate-refund"],
  "affected_sinks": ["kafka-payments", "dlq"],
  "impact_level": "high",
  "confidence": 1.0,
  "references": [...]
}
```

**Features:**
- Static cross-route reference index (no runtime overhead)
- Confidence metrics (0.0–1.0) for analysis completeness
- Uncertainty surfacing (dynamic/JSONata references flagged)
- Impact levels: low (0 affected), medium (1–3), high (4–10), critical (11+)

---

## Use Cases

### 1. Route Design from Scratch
**Agent:** "Design a route that reads from an SFTP server, validates each file against payment-schema, transforms currency to cents, and writes to Kafka"

**dim:** Scaffolds full route YAML with all fields filled in. Human reviews, adjusts, merges.

### 2. Impact Analysis Before Changes
**Team:** "We're deprecating the old payment-contract version"

**Agent:** Queries `query_impact` to find all affected routes and sinks. Reports risks:
- 2 routes need testing
- 3 sinks need validation
- 1 route uses dynamic contract references (uncertain)

**Result:** Risk mitigation plan before making changes.

### 3. Route Critique & Optimization
**Agent:** Reads proposed route YAML, runs critique:
- "Redundant translate step 2 and step 4 can be combined"
- "Missing error_path will cause messages to fail silently"
- "Auth step uses 'rbac' but never declares allowRoles"

**Human:** Reviews suggestions, accepts/rejects, commits.

### 4. Capability Discovery
**Agent:** Needs to know what adapters are available before designing.

```bash
./dimctl get-capabilities
```

Returns complete enumeration of sources, sinks, steps, contracts, and schemas. Agent uses this to design within known constraints.

---

## Architecture

### MCP Server Integration
```
Agent (Claude)
    ↓
MCP Server (Agent Interface)
    ├→ validate_route
    ├→ test_route
    ├→ scaffold_from_intent
    ├→ critique_route
    ├→ query_impact
    ├→ get_capabilities
    └→ query_lineage
    ↓
dim SDK (existing CLI)
    ├→ Config loader
    ├→ Schema validator
    ├→ Impact index builder
    └→ Capability enumerator
```

### Impact Index
```
Route Configs (YAML)
    ↓
[Import resolution]
    ↓
Static Index
    ├── ContractReferences (which sinks enforce which contracts)
    ├── SinkReferences (which routes use which sinks)
    ├── SourceReferences (which routes read from which sources)
    └── DynamicReferences (flagged as uncertain)
    ↓
Query Surface
    ├── "What routes reference contract X?" → [route1, route2]
    └── Confidence: X% (some refs are dynamic)
```

---

## Testing

**Phase 4 Test Suite: 150+ new tests**
- MCP server operations (12 tests)
- Capability manifest generation (8 tests)
- Route scaffolding (40+ tests)
- Route critique (25+ tests)
- Impact index building (30+ tests)
- Impact queries (35+ tests)

**All tests pass:**
```
✅ go test ./internal/agent ./internal/lineage -v
✅ go test -race ./... (no race conditions)
✅ CI: build ✅, vet ✅, tests ✅, validation ✅
```

---

## Documentation

- **[PHASE-4-COMPLETION-REPORT.md](../design/PHASE-4-COMPLETION-REPORT.md)** — Complete technical summary
- **[docs/examples/m48-route-design-examples.md](examples/m48-route-design-examples.md)** — Worked examples (scaffolding, critique)
- **[docs/examples/m49-impact-analysis-examples.md](examples/m49-impact-analysis-examples.md)** — Impact query examples
- **[OKF.md](OKF.md)** — Operational patterns for agent-assisted workflows

---

## Key Principles

1. ✅ **Agents propose, humans review** — No automatic deployment
2. ✅ **Complete knowledge** — Capability manifest auto-generated, always in sync
3. ✅ **Explicit about limits** — Uncertainty in impact queries clearly flagged
4. ✅ **Standard interface** — MCP for seamless agent integration
5. ✅ **Configuration-first** — Routes remain YAML, validated before execution

---

## Next Steps

- **Gather agent feedback** — Insights from real agent usage
- **Performance optimization** — Profile impact index for large route sets
- **Advanced lineage queries** — Cross-cluster impact analysis
- **Phase 5** (if greenlit) — Additional capabilities based on user needs

---

## Backward Compatibility

✅ All existing routes continue to validate identically  
✅ CLI commands unchanged (MCP is additive)  
✅ No breaking changes to configuration format
