# dim — Production-Ready Integration Middleware

A declarative, configuration-driven integration middleware for routing, transforming, and auditing messages between systems — without code, configuration-first.

**Status:** ✅ **Phase 0 Production Ready (v0.5.0)** | ✅ **Phase 1 Complete (v0.6.0)** | ✅ **Phase 2 Complete (v0.7.0-beta)** | ✅ **Phase 3 Complete (v0.8.0-beta)**

- Phase 0: Complete. 440+ tests, security audit passed.
- Phase 1 Track A: Complete (R1-R21). Governance, CLI, daemon, observability, schema registry, SFTP, OpenLineage dead-letter.
- Phase 1 Track B: Complete (M1.2-M1.8). PBAC+OPA, OBO, AMQP/Kafka reliability, OpenLineage/Marquez, replay tooling.
- Phase 2 Track B: Complete (M2.1-M2.7). Aggregator, splitter, database adapters, fragment params, obligations/redaction, purge-log auto-export, static contract checking.
- Phase 3: Complete (M3.1-M3.5). Distributed clustering (Postgres dedup, lineage), claim-check pattern (in-memory/S3), plugin SDK (native-Go, WASM), multi-tenant resource quotas, e2e release pipeline.

## What is dim?

**dim** is a **declarative integration middleware** that routes, transforms, and audits messages between systems. You define pipelines in YAML, not code. Every message is traced, authorized, and audited.

**Use dim for:**
- Message routing between APIs, queues, and databases
- Data validation with JSON Schema contracts
- Access control (RBAC, ABAC, PBAC) on every message
- Audit trail and compliance (lineage, retention, purge evidence)
- Observability (OTel tracing, Prometheus metrics)
- Custom business logic (plugins: Go and WASM)
- Large message handling (claim-check pattern)
- Multi-tenant isolation with resource quotas
- Horizontal scaling with distributed clustering

---

## Complete Feature Set

### Message Routing & Transformation
- **Configuration-First:** Routes defined in YAML, not code
- **EIP Steps:** Filter, translate, route (conditional branching), aggregate, split
- **JSONata Expressions:** Powerful transformation language with full function library
- **Plugin Functions:** Custom Go (RPC) and WebAssembly (sandboxed) functions
- **Claim Check Pattern:** Handle large payloads (5MB+ attachments) efficiently
- **Hot Reload:** Config changes take effect without downtime or message loss

### Governance & Security
- **Authentication:** JWT token validation and principal extraction
- **Authorization:** RBAC (role-based), ABAC (attribute-based), PBAC (policy-based with OPA)
- **Data Contracts:** JSON Schema validation at source/sink boundaries
- **Contract Registry:** Inline schemas + registry-backed (Apicurio, Confluent)
- **Obligations:** PDP-driven field redaction, encryption, rate limiting
- **Secrets Management:** `${SECRET:name}` resolved at runtime, never stored plaintext
- **Security Audit:** Authorization decisions tracked in lineage

### Message Processing & Reliability
- **Worker Pools:** Configurable concurrency per route with backpressure
- **Retry Logic:** Exponential backoff for transient failures
- **Dead-Letter Queue:** Automatic routing of unrecoverable messages
- **Error Classifications:** Retryable (5xx), non-retryable (4xx), auth denied, contract violation
- **Ordered Processing:** Guaranteed order per correlation ID
- **Idempotent Processing:** Prevents duplicate message processing

### Observability & Monitoring
- **OpenTelemetry Tracing:** OTLP export to Jaeger, Datadog, etc.
- **Prometheus Metrics:** Message counts, latency histograms, authorization decisions
- **Live Viewer:** Built-in `/debug/routes` showing real-time stats
- **Trace Streaming:** `dimctl trace tail` for live execution visibility
- **Message Provenance:** Reconstruct any message's journey through lineage

### Audit & Compliance
- **SQLite Lineage Store:** Embedded audit trail (no external DB needed)
- **Cross-Cluster Lineage:** Unified queries across distributed instances (PostgreSQL backend)
- **Retention Policies:** Named policies (default, pci, phi, custom) with TTL
- **Automatic Purge:** Background reaper deletes old records per policy
- **Purge Evidence:** Append-only log proving deletion (tamper-proof)
- **Export & Audit:** CSV/NDJSON export for compliance analysis
- **Subject ID Tracking:** GDPR-ready subject-based queries and deletion

### Data Connectivity
- **Sources:** HTTP, file polling, Kafka (consumer groups), AMQP, S3, SFTP, database (CDC)
- **Sinks:** HTTP, file, Kafka (producer), AMQP, S3, SFTP, database (UPSERT/INSERT)
- **CDC:** Log-based (Debezium/Maxwell) and trigger-based (Postgres timestamp watermark)
- **Schema Evolution:** Contract versioning independent from route versioning

### Multi-Tenancy & Scalability
- **Resource Quotas:** Per-tenant message rate limits, worker slot allocation, lineage storage caps
- **Distributed Clustering:** Active-active mode with shared state (PostgreSQL/Redis backend)
- **No Duplicate Processing:** Shared idempotent dedup store across cluster
- **Coordinated Hot-Reload:** Config changes reach all instances together
- **Geographic Distribution:** Run instances across regions with unified lineage

### Configuration & Composition
- **Fragments:** Reusable YAML blocks with late binding and parameter substitution
- **Imports:** Recursive configuration composition with cycle detection
- **Parameter Substitution:** `${PARAM:name}` for template values
- **Build Tags:** Conditional compilation for environment-specific code
- **Fragment Defaults:** Merged from global, domain, and route levels

### Performance & Efficiency
- **Single Static Binary:** No runtime dependencies (SQLite embedded, WASM runtime pure Go)
- **High Throughput:** 1,245+ messages/sec on single instance
- **Low Latency:** <50ms p99, <1% tracing overhead
- **Horizontal Scaling:** Add instances for linear throughput increase
- **Resource Accounting:** Per-tenant quotas prevent noisy neighbor issues

### Extensibility
- **Native Plugins:** Go binaries via RPC (fast, trusted logic)
- **WASM Plugins:** Sandboxed functions (any language, secure)
- **Pluggable Functions Registry:** Custom functions in expressions
- **Adapter Interfaces:** Implement Source/Sink for custom adapters
- **Policy Decision Point:** Pluggable authorization engine (OPA reference)

### Developer Experience
- **Single Binary:** `go build ./cmd/dimctl` produces ready-to-use CLI
- **No Code Required:** Everything is configuration
- **Rapid Development:** Hot reload for instant testing
- **Debugging Tools:** `dimctl explain`, `dimctl trace tail`, `dimctl provenance`
- **Testing Framework:** Fixture-based route validation
- **Race Detector Ready:** Full concurrency testing coverage

---

## Quick Start (5 minutes)

### 1. Install

**Option A: Pre-built binaries (recommended)**
```bash
# Automatic install (detects OS/arch, downloads from GitHub Releases)
curl -sSL https://raw.githubusercontent.com/naren-chakraview/dim/master/scripts/install.sh | sh

# Or download manually from GitHub Releases
# https://github.com/naren-chakraview/dim/releases/latest
```

**Option B: Go install (requires Go 1.26.7+)**
```bash
go install github.com/naren-chakraview/dim/cmd/dimd@latest
go install github.com/naren-chakraview/dim/cmd/dimctl@latest
```

**Option C: Build from source**
```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
go build ./cmd/dimd -o dimd
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

**For Users — Getting Started:**
- **[docs/GETTING_STARTED.md](docs/GETTING_STARTED.md)** — Quick start (5-minute tutorial)
- **[docs/LANGUAGE_REFERENCE.md](docs/LANGUAGE_REFERENCE.md)** — Complete configuration reference
- **[docs/USE_CASES.md](docs/USE_CASES.md)** — Real-world examples and patterns
- **[docs/CLI_REFERENCE.md](docs/CLI_REFERENCE.md)** — `dimctl` command reference

**For Developers:**
- **[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)** — Project structure, testing, adding features
- **[OKF.md](OKF.md)** — Operational knowledge framework (architecture decisions, patterns)
- **[Design Documents](design/)** — EIP mapping, implementation roadmaps, feasibility studies

**Release & Operations:**
- **[RELEASE.md](RELEASE.md)** — Release process, semantic versioning, creating releases
- **[RELEASE_NOTES_v0.5.0.md](RELEASE_NOTES_v0.5.0.md)** — Phase 0 features and deployment checklist
- **[docs/E2E_TESTING.md](docs/E2E_TESTING.md)** — Integration testing, local e2e setup, release gates
- **[Security Review](internal/security/SECURITY_REVIEW.md)** — Audit results (no critical issues)
- **[Performance Results](examples/bench/RESULTS.md)** — Benchmarks and metrics
- **[ADAPTER_SPEC.md](ADAPTER_SPEC.md)** — Source/Sink interface contract, patterns
- **[CODE_REVIEW_WORKFLOW.md](CODE_REVIEW_WORKFLOW.md)** — 3-tier review governance
- **[REVIEWERS.md](REVIEWERS.md)** — Subsystem expertise mapping

**Specifications:**
- **[KAFKA_ADAPTER.md](KAFKA_ADAPTER.md)** — Kafka consumer/producer configuration
- **[AMQP_ADAPTER.md](AMQP_ADAPTER.md)** — AMQP queue-based consumption and publishing

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
├── deploy/                # Docker Compose for e2e testing & local development
│   ├── docker-compose.e2e.yml # Full integration environment (Kafka, PostgreSQL, etc.)
│   └── prometheus.e2e.yml     # Prometheus config for local monitoring
├── scripts/               # utility scripts
│   ├── e2e-test.sh        # Run full e2e test suite locally
│   └── check-release-readiness.sh # Pre-release validation
├── .github/workflows/     # CI/CD automation
│   ├── ci.yml             # Unit tests, vet, lint on every PR
│   ├── e2e.yml            # Integration test gates on every push
│   └── release.yml        # Release workflow (tag → build → publish)
├── graphify-out/          # knowledge graph (1130 nodes, 3352 edges)
├── ADAPTER_SPEC.md        # Source/Sink interface specification
├── CODE_REVIEW_WORKFLOW.md # 3-tier review governance
├── REVIEWERS.md           # subsystem expertise
├── RELEASE.md             # Release process and semantic versioning
├── USER_GUIDE.md          # user guide (configuration reference)
├── DEVELOPMENT.md         # development patterns
├── DEVELOPER_GUIDE.md     # developer guide (project structure, building, testing)
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

**User-Facing Guides (in `docs/`):**
- **[docs/GETTING_STARTED.md](docs/GETTING_STARTED.md)** — 5-minute quick start
- **[docs/LANGUAGE_REFERENCE.md](docs/LANGUAGE_REFERENCE.md)** — Complete config reference (sources, sinks, steps, expressions)
- **[docs/USE_CASES.md](docs/USE_CASES.md)** — 10 real-world patterns (e-commerce, compliance, CDC, notifications, etc.)
- **[docs/CLI_REFERENCE.md](docs/CLI_REFERENCE.md)** — `dimctl` and `dimd` command reference

**Design & Architecture (in `design/`):**
- **[design/eip-middleware-design.md](design/eip-middleware-design.md)** — Core architecture, EIP mapping, all features, design tenets
- **[design/phase-0-implementation-plan.md](design/phase-0-implementation-plan.md)** — Engineering roadmap: 6 milestones, 50 subtasks
- **[design/phase-*-implementation-plan.md](design/)** — Implementation plans for Phases 1 and 2
- **[design/data-mesh-feasibility-analysis.md](design/data-mesh-feasibility-analysis.md)** — Data mesh principles fit
- **[design/data-mesh-reference-architecture.md](design/data-mesh-reference-architecture.md)** — Reference data mesh on dim
- **[design/self-service-feasibility-study.md](design/self-service-feasibility-study.md)** — Self-service deployment patterns

## Getting Started (5 Minutes)

### 1. Quick Start
```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
go build ./cmd/dimctl -o dimctl
```

### 2. Create Your First Route
See **[docs/GETTING_STARTED.md](docs/GETTING_STARTED.md)** for a complete 5-minute tutorial.

### 3. Explore
- **[docs/LANGUAGE_REFERENCE.md](docs/LANGUAGE_REFERENCE.md)** — All configuration options explained
- **[docs/USE_CASES.md](docs/USE_CASES.md)** — Copy-paste real-world patterns (payments, CDC, multi-source ingestion, etc.)
- **[docs/CLI_REFERENCE.md](docs/CLI_REFERENCE.md)** — `dimctl` command reference

### Next Steps

- **[Configuration Examples](examples/fragments/)** — Working YAML examples in the repository
- **[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)** — Contributing, testing, project structure
- **[OKF.md](OKF.md)** — Architecture decisions and design patterns
- **[Security Review](internal/security/SECURITY_REVIEW.md)** — Audit results
- **[Performance Results](examples/bench/RESULTS.md)** — Benchmarks and metrics

### Understanding the Codebase

- **[graphify-out/GRAPH_REPORT.md](graphify-out/GRAPH_REPORT.md)** — Module overview and god nodes
- **[design/eip-middleware-design.md](design/eip-middleware-design.md)** — Architecture and EIP mapping
- **[design/phase-0-implementation-plan.md](design/phase-0-implementation-plan.md)** — Implementation roadmap

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
