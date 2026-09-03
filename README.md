# dim — Production-Ready Integration Middleware

A declarative, configuration-driven integration middleware for routing, transforming, and auditing messages between systems — without code, configuration-first.

**Status:** ✅ **Phase 0 Production Ready (v0.5.0)**  
Governance, observability, and audit complete. 440+ tests passing, security audit passed, performance benchmarked.

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
go build ./cmd/midctl -o midctl
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
./midctl run config.yaml

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

- **[RELEASE_NOTES_v0.5.0.md](RELEASE_NOTES_v0.5.0.md)** — Complete feature list, deployment checklist
- **[User Guide](USER_GUIDE.md)** — Installation, concepts, project structure
- **[Security Review](internal/security/SECURITY_REVIEW.md)** — Audit results (no critical issues)
- **[Performance Results](examples/bench/RESULTS.md)** — Benchmarks and metrics
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
go build ./cmd/midctl -o midctl  # CLI tool
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
| Validate a route | `./midctl validate config.yaml` | ✅ M0.1 |
| Run a route | `./midctl run config.yaml` | ✅ M0.1 |
| Send a test message | `curl -X POST http://localhost:8080/message -H "Authorization: Bearer <token>" -d '{...}'` | ✅ M0.1 |
| View live stats | `curl http://localhost:8081/debug/routes` | ✅ M0.5 |
| Stream traces | `./midctl trace tail --route example` | ✅ M0.5 |
| Query lineage | `./midctl lineage query --message-id msg-123` | ✅ M0.4 |
| Export audit trail | `./midctl lineage export --format csv --since 2026-09-01` | ✅ M0.4 |
| Purge old records | `./midctl lineage purge --before 2026-06-01 --reason "compliance"` | ✅ M0.4 |

## Project structure

```
dim/
├── cmd/
│   └── midctl/            # CLI tool (validate, run)
├── internal/
│   ├── config/            # YAML + schema validation
│   ├── engine/            # executor, channels, messages, DLQ
│   ├── steps/             # EIP steps (filter, translate)
│   ├── expr/              # JSONata expression evaluation
│   ├── adapters/          # sources and sinks (HTTP, file)
│   └── factory/           # pipeline builder
├── pkg/sdk/               # public SPI (placeholder)
├── schemas/               # JSON Schema for routes
├── examples/              # sample configurations
├── test/fixtures/         # test fixtures
├── design/                # architecture docs
├── graphify-out/          # knowledge graph (architecture AST)
├── M0.1_COMPLETE.md       # M0.1 walking skeleton documentation
├── M0.1_PROGRESS.md       # completion status
├── M0.1_IMPLEMENTATION_ROADMAP.md
├── USER_GUIDE.md          # developer guide
├── DEVELOPMENT.md         # development patterns
├── OKF.md                 # operational knowledge framework
└── README.md              # this file
```

**M0.2+ directories (deferred):**
- `cmd/dimd/` — engine daemon
- `internal/route/` — route model, DAG compilation
- `internal/lineage/` — lineage store, retention, purge
- `internal/authz/` — authorization (RBAC/ABAC)
- `internal/observability/` — metrics, tracing, viewer

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

**Phase 1 and later:**
- **Kafka, AMQP, S3 adapters**
- **Schema registry integration**
- **Persistent idempotent store**
- **Built-in TLS listener**
- **Distributed consensus for ordering**
- **WASM expression sandbox**
- **GraphQL API over lineage**
- **Kubernetes operator**

See [RELEASE_NOTES_v0.5.0.md](RELEASE_NOTES_v0.5.0.md) for complete feature list.

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
go build ./cmd/midctl -o midctl

# Try example route
./midctl run examples/fragments/simple.yaml
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
./midctl trace tail
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
