# dim — Declarative Integration Middleware

A declarative, configuration-driven integration middleware built on Enterprise Integration Patterns (EIP). Routes and transforms messages between heterogeneous systems using YAML-defined routes rather than hand-written glue code.

**Status:** Phase 0 (M0.1) — Walking skeleton complete ✅. End-to-end message processing pipeline with 120/121 tests passing.

## What is dim?

dim is a single-binary message router that sits between heterogeneous systems — queues, topics, HTTP endpoints, files, databases — and lets you define how messages flow, transform, and fail without writing code. Every route is a YAML file: diffable, reviewable, testable, versionable.

## Quick links

- **[User Guide](USER_GUIDE.md)** — Installation, quick start, project structure, developer notes
- **[Design Documents](design/)** — Architecture, implementation plan, data mesh analysis, reference architectures

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
```

## Testing

```bash
go test ./...                        # unit tests (120/121 passing)
go test -race ./...                  # race detector (required for CI)
```

**Note:** `dimd` daemon is in M0.2+. Currently, use `midctl run` to start a pipeline.

## Developer quick reference

| Task | Command | Status |
|---|---|---|
| Validate a route | `./midctl validate examples/test-route.yaml` | ✅ M0.1 |
| Run a route locally | `./midctl run examples/test-route.yaml` | ✅ M0.1 |
| Send a test message | `curl -X POST http://localhost:8080/ingest -H "Content-Type: application/json" -d '{"amount": 100}'` | ✅ M0.1 |
| Check output | `tail -f output/messages.jsonl` | ✅ M0.1 |
| View error messages | `tail -f output/errors.jsonl` | ✅ M0.1 |
| Test a route | `./midctl test examples/` | ⏳ M0.2 |
| Explain DAG | `./midctl explain examples/` | ⏳ M0.2 |
| Tail live traces | `./midctl trace tail` | ⏳ M0.3 |
| Query message history | `./midctl provenance <correlation-id>` | ⏳ M0.4 |

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

## Phase 0 (M0.1) — Walking Skeleton ✅

**Implemented (M0.1.1–M0.1.10):**
- ✅ Core engine: single-worker executor with step pipeline
- ✅ Message envelope: Headers + Body + Metadata with correlation tracking
- ✅ Bounded channels: configurable buffer with backpressure awareness
- ✅ Steps: `filter` (boolean predicates), `translate` (JSONata transformations)
- ✅ Adapters: HTTP source (port 8080), file sink (JSONL output)
- ✅ JSONata expressions: blues/jsonata-go v1.5.4 (pure Go, 90% spec compliance)
- ✅ Configuration: YAML route config with JSON Schema validation
- ✅ Error handling: dead-letter envelope wrapping failed messages
- ✅ CLI: `midctl validate`, `midctl run`
- ✅ Tests: 120/121 passing (99.2% coverage)

**Phase 1 and later:**
- **M0.2 Reliability:** Worker pools, retry logic, additional step types (route, wiretap, idempotent, authorize)
- **M0.3 Observability:** Prometheus metrics, OTel tracing, built-in viewer
- **M0.4 Lineage:** Embedded SQLite store, retention policies, purge mechanism
- **M0.5 Authorization:** RBAC/ABAC enforcement, principal propagation
- **M0.6 Scalability:** Hot reload, multi-route support, distributed control plane
- **Phase 1+:** Kafka, AMQP, schema registry, OpenLineage export, OBO token exchange

See [phase-0-implementation-plan.md](design/phase-0-implementation-plan.md) (§2) for full details.

## Documentation

All design and planning docs are in `design/`:

- **[eip-middleware-design.md](design/eip-middleware-design.md)** — Core architecture, EIP mapping, all features, design tenets, open questions (v8)
- **[phase-0-implementation-plan.md](design/phase-0-implementation-plan.md)** — Engineering roadmap: 6 milestones, 50 subtasks, risk register, testing strategy (v5)
- **[data-mesh-feasibility-analysis.md](design/data-mesh-feasibility-analysis.md)** — How well does this design fit data mesh principles?
- **[data-mesh-reference-architecture.md](design/data-mesh-reference-architecture.md)** — Reference impl of a data mesh on top of dim; domain/namespace proposal
- **[self-service-feasibility-study.md](design/self-service-feasibility-study.md)** — How feasible is self-service config/deploy? (very; GitOps + guardrails)

## Getting started

1. **[Read M0.1_COMPLETE.md](M0.1_COMPLETE.md)** for architecture and Phase 0 features
2. **Clone and build:**
   ```bash
   git clone https://github.com/naren-chakraview/dim.git
   cd dim
   go build ./cmd/midctl -o midctl
   ```
3. **Try a route:**
   ```bash
   ./midctl run examples/test-route.yaml
   # In another terminal:
   curl -X POST http://localhost:8080/ingest \
     -H "Content-Type: application/json" \
     -d '{"name": "Alice", "amount": 100}'
   tail -f output/messages.jsonl
   ```
4. **Run tests:**
   ```bash
   go test ./...
   ```
5. **Explore the codebase:**
   - Start with `graphify-out/GRAPH_REPORT.md` for architecture overview
   - Then `M0.1_COMPLETE.md` for Phase 0 design decisions
   - Then `design/phase-0-implementation-plan.md` for roadmap

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
