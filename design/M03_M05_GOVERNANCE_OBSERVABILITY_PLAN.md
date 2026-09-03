# M0.3 & M0.5 Implementation Plan — Governance & Observability

**Date:** September 1, 2026  
**Status:** Planning Phase  
**Scope:** Remaining Phase 0 features (M0.3 Governance + M0.5 Observability)

---

## Overview

With M0.2.10 (hot reload) and M0.4 (lineage) complete, the remaining Phase 0 work consists of:

1. **M0.3: Governance (7 subtasks)** — Authorization policies and data contracts
2. **M0.5: Observability (5 subtasks)** — OpenTelemetry tracing and dashboards

These features complete the Phase 0 walking skeleton to production-grade compliance and operational visibility.

---

## M0.3: Governance (7 Subtasks)

### Current Status (from prior implementation)

**Already Complete (from Phase 1):**
- ✅ M0.3.2: Authorize step, RBAC mode (`internal/steps/authorize.go`)
- ✅ M0.3.3: Authorize step, ABAC mode (same file)

**NOT YET IMPLEMENTED:**
- ❌ M0.3.1: Principal propagation from HTTP adapter (JWT-based)
- ❌ M0.3.4: Mandatory `auth:` declaration validation
- ❌ M0.3.5: Inline data contract loading + conformance check
- ❌ M0.3.6: `contract_violation` classification + `on_violation` routing
- ❌ M0.3.7: `contract_version` stamping

### M0.3.1: Principal Propagation (JWT-based)

**Purpose:** Extract principal (user identity + claims) from HTTP request headers

**What to Build:**
- JWT token extraction from `Authorization: Bearer <token>` header
- Public key or HMAC secret configuration (from environment or config file)
- Claim extraction: subject, roles, attributes
- Propagation to message metadata for authorization checks

**Implementation:**
- `internal/authz/principal.go` — Principal type (subject, roles, attributes)
- `internal/authz/jwt.go` — JWT parsing and validation
- `internal/adapters/http/http.go` — Extract principal on request ingestion

**Exit Criteria:**
- Unit test: JWT token parsed and claims extracted
- Integration test: principal attached to message metadata
- Configuration: JWT secret configurable via env var or config file

---

### M0.3.4: Mandatory `auth:` Declaration Validation

**Purpose:** Enforce explicit auth configuration on every route

**What to Build:**
- YAML schema validation: `auth:` field required on every route
- CLI flag: `midctl validate --strict` promotes warning to error
- Guidance: log "auth: none" if intentional unauthenticated route

**Implementation:**
- Update `internal/config/schema.go` to make auth required
- Update `cmd/midctl/main.go` validate command with `--strict` flag
- Validation message: "Route <name> missing auth: declaration"

**Exit Criteria:**
- Unit test: validation rejects route missing auth
- CLI test: `--strict` flag works correctly
- Log message: clear guidance on how to fix

---

### M0.3.5-6: Data Contracts (Loading + Violation Classification)

**Purpose:** Validate message payload against JSON Schema contracts

**What to Build:**

**M0.3.5: Contract Loading**
- Inline JSON Schema files (beside route YAML or embedded in config)
- Contract structure: `{id, version, schema, on_violation}`
- Load and cache contracts at startup

**M0.3.6: Conformance Check + Violation Routing**
- Validate message body against contract schema
- Classify violations: `contract_violation` error type
- Route violations per contract's `on_violation` target (success → sink, error → error-sink)

**Implementation:**
- `internal/steps/contract.go` — Contract conformance checking
- `internal/config/schema.go` — Contract definition (id, version, schema, on_violation)
- Contract validation integrated into pipeline execution

**Exit Criteria:**
- Unit test: conforming message passes
- Unit test: non-conforming message classified as contract_violation
- Integration test: violation routed to on_violation sink
- Schema validation library: `santhosh-tekuri/jsonschema/v5`

---

### M0.3.7: Contract Version Stamping

**Purpose:** Track which contract version validated a message

**What to Build:**
- Stamp `contract_version` on message metadata
- Store in lineage records (M0.4 integration)
- Enables compliance audits (which contract version cleared which messages)

**Implementation:**
- Update `internal/engine/message.go` metadata to include `contract_version`
- Update `internal/lineage/store.go` to record contract_version
- Update pipeline to stamp on successful contract check

**Exit Criteria:**
- Message metadata carries contract_version
- Lineage records include contract_version
- Query: can find all messages validated by specific contract version

---

## M0.5: Observability (5 Subtasks)

### Current Status

**Already Complete (from Phase 2):**
- ✅ M0.5.2: Prometheus metrics (`internal/observability/metrics.go`)

**NOT YET IMPLEMENTED:**
- ❌ M0.5.1: OTel span instrumentation
- ❌ M0.5.3: Tier 1 built-in viewer
- ❌ M0.5.4: Example Tier 2 Grafana dashboard
- ❌ M0.5.5: `midctl trace tail` command

### M0.5.1: OTel Span Instrumentation

**Purpose:** Distributed tracing with OpenTelemetry

**What to Build:**
- Span generation for message processing lifecycle
- Attributes: `route_version`, `contract_version`, `principal`
- Span hierarchy: message → steps → output
- OTLP exporter for external collectors (Jaeger, Tempo, Datadog)

**Implementation:**
- Extend `internal/observability/tracing.go` (already exists from Phase 2)
- Add span generation on message receipt, step execution, output
- Integrate into executor run loop

**Exit Criteria:**
- Unit test: spans generated with correct attributes
- Integration test: local Jaeger receiver gets spans
- Attributes verified: route_version, contract_version, principal present

---

### M0.5.3: Tier 1 Built-in Viewer

**Purpose:** In-process live DAG visualization (no external dependency)

**What to Build:**
- HTTP endpoint: `/debug/routes` returns live route states
- In-memory ring buffer: recent messages (100-500 most recent)
- JSON response: route name, in-flight count, recent message IDs, latencies

**Implementation:**
- `internal/observability/viewer/viewer.go` — Ring buffer + HTTP handler
- Register endpoint in `cmd/midctl/main.go`
- Response: `{route: string, inFlight: int, recentMessages: [], avgLatency: ms}`

**Exit Criteria:**
- HTTP endpoint live after startup
- Ring buffer captures recent messages
- Response is valid JSON with expected fields

---

### M0.5.4: Example Grafana Dashboard

**Purpose:** Pre-built Grafana dashboard for operators

**What to Build:**
- Dashboard JSON (Grafana API format)
- Panels: message count, latency (p50/p90/p99), error rate, worker utilization
- Data source: Prometheus (metrics from M0.5.2)
- Panels for multiple routes (dropdown selector)

**Implementation:**
- `deploy/grafana/dashboard.json` — Dashboard-as-code
- Queries: standard Prometheus PromQL for dim metrics
- Example setup: docker-compose with Prometheus + Grafana

**Exit Criteria:**
- Dashboard JSON valid (can import into Grafana)
- Panels render against sample Prometheus data
- README: how to import and configure

---

### M0.5.5: `midctl trace tail` Command

**Purpose:** Stream live spans to terminal for debugging

**What to Build:**
- CLI command: `midctl trace tail [--route <name>] [--service <name>]`
- Connect to OTel collector (via gRPC) or read from Jaeger API
- Stream spans in real-time with color coding
- Format: timestamp | route | span_name | duration_ms | attributes

**Implementation:**
- `cmd/midctl/main.go` — Add `trace tail` command
- `internal/observability/tail.go` — Span streaming logic
- Configuration: OTel collector endpoint (from env or config)

**Exit Criteria:**
- CLI command parses arguments
- Connects to OTel collector
- Streams spans in expected format
- Test: local Jaeger server integration

---

## Dependency Analysis

```
M0.3.1: Principal Propagation
    └─ Requires: JWT library (stdlib or external)

M0.3.2-3: Authorize (RBAC/ABAC)
    └─ Already done ✅

M0.3.4: Mandatory auth: validation
    └─ Requires: M0.3.1 (principal type definition)

M0.3.5-7: Contracts + versioning
    └─ Requires: M0.4 (lineage integration)

M0.5.1: OTel Tracing
    └─ Requires: go.opentelemetry.io/* (already in go.mod from Phase 2)

M0.5.3: Built-in Viewer
    └─ No dependencies (in-process only)

M0.5.4: Grafana Dashboard
    └─ No dependencies (static JSON)

M0.5.5: trace tail
    └─ Requires: M0.5.1 (OTel integration)
```

---

## Execution Strategy

### Option 1: Sequential (8-10 hours)
```
M0.3.1 → M0.3.4 → M0.3.5-7 → M0.5.1 → M0.5.3 → M0.5.4 → M0.5.5
```

### Option 2: Parallel (5-6 hours wall clock) ⭐ Recommended
```
Phase A: M0.3 Governance (Serial: 1 → 4 → 5-7)
    └─ 5-6 hours
    
Phase B: M0.5 Observability (Parallel: 1, 3, 4, 5)
    ├─ M0.5.1 (OTel tracing) — 1-2 hours
    ├─ M0.5.3 (Built-in viewer) — 1 hour
    ├─ M0.5.4 (Grafana dashboard) — 1 hour
    └─ M0.5.5 (trace tail) — 1.5 hours (depends on M0.5.1)
    
Total: ~6 hours wall clock
```

**Recommendation:** Launch M0.3 sequential (dependencies) + M0.5.1/3/4 in parallel, then M0.5.5 after M0.5.1 done.

---

## Quality Gates

✅ All tests pass with `-race` flag  
✅ M0.3 governance complete (7/7 subtasks)  
✅ M0.5 observability complete (5/5 subtasks)  
✅ Backward compatible with Phase 0-1-2-4 features  
✅ End-to-end integration: principal → authorize → contract → trace → metrics  

---

## Phase 0 Completion Criteria

After M0.3 + M0.5:
- ✅ M0.1: Walking skeleton (DONE in v0.1)
- ✅ M0.2: Reliability & composition (DONE in v0.2-0.4)
- ✅ M0.3: Governance (TODO)
- ✅ M0.4: Lineage & purge (DONE in v0.4)
- ✅ M0.5: Observability (TODO)
- ✅ M0.6: Hardening & release (TODO after M0.5)

**Phase 0 will be production-complete after M0.3 + M0.5 + M0.6.**

---

## Success Criteria

**M0.3: Governance**
- ✅ JWT principal extracted from Authorization header
- ✅ Principal attached to message metadata
- ✅ RBAC/ABAC authorization working end-to-end
- ✅ Auth declaration mandatory on all routes (validated)
- ✅ Contract schema loaded and validated
- ✅ Contract violations classified and routed correctly
- ✅ Contract version stamped and recorded in lineage

**M0.5: Observability**
- ✅ OTel spans generated with route_version, contract_version, principal
- ✅ Built-in viewer showing live route state
- ✅ Grafana dashboard renders with Prometheus metrics
- ✅ `midctl trace tail` streams spans to terminal
- ✅ All observability integrates end-to-end

---

## Next Steps

1. Confirm execution strategy (sequential M0.3, parallel M0.5)
2. Launch Phase A agent (M0.3 governance)
3. Launch Phase B agents (M0.5 observability parallel)
4. After M0.3+M0.5 complete: Plan M0.6 (hardening & final release)
