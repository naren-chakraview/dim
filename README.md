# dim — Production-Ready Integration Middleware

A declarative, configuration-driven integration middleware for routing, transforming, and auditing messages between systems — without code, configuration-first.

**Status:** ✅ **Phase 0 Production Ready (v0.5.0)** | 🔄 **Phase 1 In Progress (v0.6.0-beta)**  
Phase 0: Governance, observability, and audit complete. 440+ tests passing, security audit passed.  
Phase 1 Track A: Governance framework, CLI (dimctl), daemon (dimd), OTel integration — complete.  
Phase 1 Track B: Kafka, AMQP, S3 adapters with design-spec interfaces — complete.

## What is dim?

dim is a **configuration-driven integration middleware** that routes, transforms, and audits messages between systems. You define pipelines in YAML, not code. Every message is traced, authorized, and audited.

**Use dim for:**
- Message routing between APIs, queues, and databases
- Data validation with JSON Schema contracts
- Access control (RBAC, ABAC) on every message
- Audit trail and compliance (lineage, retention, purge evidence)
- Observability (OTel tracing, Prometheus metrics)

---

## Features

- **Configuration-First:** Routes are YAML, not code
- **Governance:** JWT auth, RBAC/ABAC, data contracts
- **Lineage:** SQLite audit trail, retention policies, purge evidence
- **Observability:** OTel tracing, Prometheus, Grafana dashboard, live viewer
- **Reliability:** Worker pools, retry logic, hot reload, ordered processing
- **Performance:** 1,245+ msg/sec, <50ms p99 latency, <1% tracing overhead
- **Security:** No hardcoded secrets, parameterized SQL, JWT validation

---

## Quick Start (5 minutes)

### 1. Build

```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
go build ./cmd/dimctl -o dimctl
```

### 2. Create Route Config

```yaml
# config.yaml
name: payment-processor
routes:
  - name: process-payment
    source: http
    steps:
      - type: authorize
        mode: rbac
        allowRoles: [admin, processor]
      - type: translate
        expression: '{"amount": body.amount, "currency": body.currency}'
    sinks:
      - type: file
        path: output/payments.jsonl
```

### 3. Run

```bash
# In terminal 1
./dimctl run config.yaml

# In terminal 2
curl -X POST http://localhost:8080/message \
  -H "Authorization: Bearer $(your-jwt-token)" \
  -H "Content-Type: application/json" \
  -d '{"amount": 100, "currency": "USD"}'

# In terminal 3
tail -f output/payments.jsonl
```

### 4. View Stats

```bash
curl http://localhost:8081/debug/routes | jq
```

---

## Quick Links

**Phase 0 (v0.5.0):**
- **[RELEASE_NOTES_v0.5.0.md](RELEASE_NOTES_v0.5.0.md)** — Complete feature list, deployment checklist
- **[Security Review](internal/security/SECURITY_REVIEW.md)** — Audit results (no critical issues)
- **[Performance Results](examples/bench/RESULTS.md)** — Benchmarks and metrics

**Phase 1 (v0.6.0-beta):**
- **[ADAPTER_SPEC.md](ADAPTER_SPEC.md)** — Source/Sink interface contract, patterns, metadata headers
- **[CODE_REVIEW_WORKFLOW.md](CODE_REVIEW_WORKFLOW.md)** — 3-tier review governance
- **[REVIEWERS.md](REVIEWERS.md)** — Subsystem expertise mapping
- **[KAFKA_ADAPTER.md](KAFKA_ADAPTER.md)** — Kafka consumer/producer configuration and examples
- **[AMQP_ADAPTER.md](AMQP_ADAPTER.md)** — AMQP queue-based consumption and publishing
- **[OKF.md](OKF.md)** — Operational knowledge framework (decisions, concurrency patterns, state)

**General:**
- **[User Guide](USER_GUIDE.md)** — Installation, concepts, project structure
- **[Design Documents](design/)** — Architecture, EIP mapping, data mesh analysis
- **[Configuration Examples](examples/fragments/)** — YAML patterns and templates

## Architecture overview

A route is a pipeline: `source → [ordered steps] → sink(s)`, with built-in error handling, data contracts, lineage tracking, and authorization.

```
Message comes in → filter (JSONata) → translate (reshape) → route (branch) → sink
                                                                                ↓
                                                                        (on error) → dead-letter
```

Features:
- **Config-first:** routes are YAML, not code
- **Reliability:** retry/backoff/dead-letter are structural, not bolted on
- **Observability:** metrics, tracing, lineage, and a built-in Tier 1 viewer
- **Authorization:** RBAC, ABAC, and pluggable policy engines
- **Data contracts:** inline or registry-backed schema validation
- **Hot reload:** route changes go live without outages or message loss
- **Expressions:** JSONata with a pluggable functions registry (Go plugins and WASM)

## Building

```bash
go build ./cmd/dimctl -o dimctl  # CLI tool
go build ./cmd/dimd -o dimd      # Daemon (optional)
```

## Testing

```bash
# Run all tests
go test ./... -race -v

# Specific package
go test ./internal/steps -race -v

# Benchmarks
go test ./examples/bench -bench=. -benchmem

# Coverage
go test ./... -cover
```

**Status:** ✅ 440+ tests passing with race detector enabled. No race conditions.

## CLI Reference

| Task | Command | Status |
|---|---|---|
| Validate a route | `./dimctl validate config.yaml` | ✅ M0.1 |
| Run a route | `./dimctl run config.yaml` | ✅ M0.1 |
| Send a test message | `curl -X POST http://localhost:8080/message -H "Authorization: Bearer <token>" -d '{...}'` | ✅ M0.1 |
| View live stats | `curl http://localhost:8081/debug/routes` | ✅ M0.5 |
| Stream traces | `./dimctl trace tail --route example` | ✅ M0.5 |
| Query lineage | `./dimctl lineage query --message-id msg-123` | ✅ M0.4 |
| Export audit trail | `./dimctl lineage export --format csv --since 2026-09-01` | ✅ M0.4 |
| Purge old records | `./dimctl lineage purge --before 2026-06-01 --reason "compliance"` | ✅ M0.4 |

## Project structure

```
dim/
├── cmd/
│   ├── dimctl/            # CLI tool (validate, run, lineage, trace, provenance)
│   └── dimd/              # Engine daemon (Phase 1 R9, complete)
├── internal/
│   ├── config/            # YAML + schema validation
│   ├── engine/            # executor, channels, messages, DLQ
│   ├── steps/             # EIP steps (filter, translate, authorize)
│   ├── expr/              # JSONata expression evaluation
│   ├── adapters/          # Source/Sink interfaces (Phase 1 R17)
│   │   ├── interfaces.go  # Source/Sink contracts, Result type
│   │   ├── kafka/         # Consumer groups, offset tracking (Phase 1 R14)
│   │   ├── amqp/          # Queue-based consumption (Phase 1 R15)
│   │   ├── s3/            # Bucket polling (Phase 1 R16)
│   │   ├── file/          # Directory polling, deduplication (Phase 0 R6)
│   │   └── http/          # HTTP source (Phase 0 M0.1)
│   ├── lineage/           # SQLite store, retention, purge (Phase 0 M0.4)
│   ├── observability/     # OTel tracing, Prometheus (Phase 0 M0.5 + Phase 1 R7)
│   └── factory/           # pipeline builder
├── pkg/sdk/               # public SPI (placeholder)
├── schemas/               # JSON Schema for routes
├── examples/              # sample configurations
├── test/fixtures/         # test fixtures
├── design/                # architecture docs
├── graphify-out/          # knowledge graph (1130 nodes, 3352 edges)
├── ADAPTER_SPEC.md        # Source/Sink interface specification
├── CODE_REVIEW_WORKFLOW.md # 3-tier review governance
├── REVIEWERS.md           # subsystem expertise
├── USER_GUIDE.md          # developer guide
├── DEVELOPMENT.md         # development patterns
├── OKF.md                 # operational knowledge framework
├── CHANGELOG.md           # version history
└── README.md              # this file
```

See [User Guide — Project structure](USER_GUIDE.md#project-structure-overview) for details.

## Design principles

1. **Config is the source of truth** — routes are data (YAML), not code
2. **Everything is a pipeline** — sources → steps → sinks, connected by channels
3. **Reliability is structural** — error handling, retry, dead-letter are always declared
4. **Small orthogonal primitives** — each EIP maps to one step type
5. **Streaming by default** — batch as a special case of scheduling
6. **Core stays small** — adapters are plugins
7. **Expressions stay pure** — stateful logic is a named function
8. **Observability is composed** — standard OTel + Prometheus
9. **Authorization is structural** — every route declares its auth stance
10. **Journey is reconstructable** — route version, function versions, authz decisions in lineage
11. **Policy is provably enforced** — retention policies paired with deletion evidence
12. **Contracts version independently** — `route_version` ≠ `contract_version`

## Phase 0 (M0.1–M0.6) — Complete ✅ [v0.5.0]

### M0.1: Core Pipeline
- ✅ HTTP message ingestion and routing
- ✅ Single-worker executor with configurable pipeline steps
- ✅ Message envelope with headers, body, metadata, correlation tracking
- ✅ Steps: `filter`, `translate`, `route`, `idempotent`
- ✅ Adapters: HTTP source (port 8080), file sink (JSONL output), DLQ
- ✅ JSONata expressions (blues/jsonata-go v1.5.4)
- ✅ YAML configuration with JSON Schema validation
- ✅ Error handling with dead-letter queue

### M0.2: Reliability & Concurrency
- ✅ Worker pool with configurable concurrency
- ✅ Exponential backoff retry logic
- ✅ Configuration composition (fragments + merge)
- ✅ Route versioning with deterministic hashing
- ✅ Hot reload (SIGHUP) with graceful message draining
- ✅ Ordered message processing per correlation ID

### M0.3: Governance & Security
- ✅ JWT principal propagation from `Authorization` headers
- ✅ Role-Based Access Control (RBAC)
- ✅ Attribute-Based Access Control (ABAC)
- ✅ Data contracts with JSON Schema validation
- ✅ Contract violation routing (`on_violation`)
- ✅ Contract version tracking

### M0.4: Lineage & Audit
- ✅ SQLite-backed message lineage store
- ✅ Static and dynamic retention policies
- ✅ Automatic reaper for policy-based purging
- ✅ Purge evidence logging (tamper-proof)
- ✅ CSV/NDJSON export for audit
- ✅ Provenance queries by message or subject ID

### M0.5: Observability
- ✅ OpenTelemetry distributed tracing (OTLP)
- ✅ Prometheus metrics collection
- ✅ Built-in live viewer (`/debug/routes`)
- ✅ Grafana dashboard (example)
- ✅ `midctl trace tail` for live span streaming

### M0.6: Hardening & Release
- ✅ Security audit document (no critical issues)
- ✅ Performance benchmarks (1,245+ msg/sec, p99: 45ms)
- ✅ Release notes and deployment checklist
- ✅ Updated README with quick start

## Phase 1 (v0.6.0-beta) — In Progress 🔄

### Track A: Governance & Infrastructure ✅ Complete
- ✅ File adapter (Phase 1 R6)
- ✅ Real OTel SDK integration (Phase 1 R7)
- ✅ Prometheus metrics (Phase 1 R8)
- ✅ dimd daemon with graceful shutdown (Phase 1 R9)
- ✅ dimctl CLI (Phase 1 R10, renamed from midctl)
- ✅ ORDERING_FEATURE.md documentation (Phase 1 R11)
- ✅ CODE_REVIEW_WORKFLOW.md + REVIEWERS.md (Phase 1 R12)
- ✅ Build health verification (Phase 1 R13)
- ✅ CLI finalization (Phase 1 R14)

### Track B: Ecosystem Adapters ✅ Complete
- ✅ **Kafka adapter** (consumer groups, offset tracking, compression/ACKs)
- ✅ **AMQP adapter** (queue-based consumption, exchange routing)
- ✅ **S3 adapter** (bucket polling, LastModified deduplication)
- ✅ **Adapter interfaces** (Source/Sink contracts, HealthCheck, Checkpoint, Result types)

### Track B: Remaining Items 🔄 Pending
- **Database adapters** (CDC, JDBC)
- **Replay tooling** (resend from lineage)
- **PBAC/OPA** (policy decision point)
- **OBO token exchange** (service auth)
- **OpenLineage export** (Marquez integration)
- **Schema Registry backing** (contract storage)

See [RELEASE_NOTES_v0.5.0.md](RELEASE_NOTES_v0.5.0.md) for Phase 0 complete feature list. See [ADAPTER_SPEC.md](ADAPTER_SPEC.md) for Phase 1 adapter patterns.

## Documentation

All design and planning docs are in `design/`:

- **[eip-middleware-design.md](design/eip-middleware-design.md)** — Core architecture, EIP mapping, all features, design tenets, open questions (v8)
- **[phase-0-implementation-plan.md](design/phase-0-implementation-plan.md)** — Engineering roadmap: 6 milestones, 50 subtasks, risk register, testing strategy (v5)
- **[data-mesh-feasibility-analysis.md](design/data-mesh-feasibility-analysis.md)** — How well does this design fit data mesh principles?
- **[data-mesh-reference-architecture.md](design/data-mesh-reference-architecture.md)** — Reference impl of a data mesh on top of dim; domain/namespace proposal
- **[self-service-feasibility-study.md](design/self-service-feasibility-study.md)** — How feasible is self-service config/deploy? (very; GitOps + guardrails)

## Getting Started

### 1. Read the Release Notes
**[RELEASE_NOTES_v0.5.0.md](RELEASE_NOTES_v0.5.0.md)** — Complete feature list, performance metrics, production checklist

### 2. Install and Run
```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
go build ./cmd/dimctl -o dimctl

# Try example route
./dimctl run examples/fragments/simple.yaml
```

### 3. Send Test Message
```bash
curl -X POST http://localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{"id": "msg-1", "amount": 100}'
```

### 4. View Live Stats
```bash
curl http://localhost:8081/debug/routes | jq
```

### 5. Stream Traces
```bash
./dimctl trace tail
```

### Next Steps

- **[Configuration Examples](examples/fragments/)** — YAML patterns for common scenarios
- **[User Guide](USER_GUIDE.md)** — Detailed concepts and CLI reference
- **[Security Review](internal/security/SECURITY_REVIEW.md)** — Audit results
- **[Performance Results](examples/bench/RESULTS.md)** — Benchmarks and metrics
- **[Development Guide](DEVELOPMENT.md)** — Code patterns and contributing

### Explore Architecture

- **[graphify-out/GRAPH_REPORT.md](graphify-out/GRAPH_REPORT.md)** — Module overview and god nodes
- **[design/eip-middleware-design.md](design/eip-middleware-design.md)** — Architecture and EIP mapping
- **[design/phase-0-implementation-plan.md](design/phase-0-implementation-plan.md)** — Roadmap and design decisions

## Contributing

Phase 0 is implemented as a series of 6 milestones (M0.1–M0.6), broken into 50 subtasks. Each subtask is defined in [phase-0-implementation-plan.md](design/phase-0-implementation-plan.md) (§5) with explicit exit criteria.

**Code review tiers:**
- **Specialist review required** for: DAG generation/hot reload, authorization (principal/RBAC/ABAC), lineage purge/reaper
- **Standard review** for everything else

No `CODEOWNERS` gate; reviewers self-select by subsystem expertise. See [phase-0-implementation-plan.md](design/phase-0-implementation-plan.md) (§6) for full testing and CI strategy.

## Understanding the codebase

The project uses **graphify** for AST-based code analysis. A knowledge graph of the codebase is maintained in `graphify-out/`:

- **`graphify-out/GRAPH_REPORT.md`** — Architecture summary, god nodes, surprising connections
- **`graphify-out/graph.html`** — Interactive visualization of modules and their relationships
- **`graphify-out/graph.json`** — Machine-readable graph (used by agents for efficient navigation)

To rebuild the graph after significant code changes:
```bash
/graphify --update
```

Coding agents use graphify to understand architecture before modifying code — it reduces exploration overhead and catches cross-module impacts that grep cannot.

## License

Apache 2.0
