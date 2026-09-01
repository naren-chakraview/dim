# dim — Declarative Integration Middleware

A declarative, configuration-driven integration middleware built on Enterprise Integration Patterns (EIP). Routes and transforms messages between heterogeneous systems using YAML-defined routes rather than hand-written glue code.

**Status:** Phase 0 — core engine scaffold, architecture complete, implementation roadmap finalized.

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
go build ./cmd/dimd -o dimd    # engine daemon
go build ./cmd/midctl -o midctl  # CLI tool
```

## Testing

```bash
go test ./...                        # unit tests
go test -race ./...                  # race detector (required for CI)
./midctl test examples/              # route fixture tests
```

## Developer quick reference

| Task | Command |
|---|---|
| Validate a route | `./midctl validate examples/hello.yaml` |
| Test a route | `./midctl test examples/` |
| Run a route locally | `./dimd examples/hello.yaml` |
| Explain a route's DAG | `./midctl explain examples/hello.yaml` |
| Tail live traces | `./midctl trace tail <route-name>` |
| Query message history | `./midctl provenance <correlation-id>` |
| Export lineage for audit | `./midctl lineage export --route <name> --since <date> --format csv` |

## Project structure

```
dim/
├── cmd/
│   ├── dimd/              # engine daemon
│   └── midctl/            # CLI tool
├── internal/
│   ├── config/            # YAML + schema validation
│   ├── route/             # route model, DAG compilation
│   ├── engine/            # executor, channels, hot reload
│   ├── steps/             # EIP steps (filter, translate, route, etc.)
│   ├── expr/              # JSONata, functions, WASM/plugins
│   ├── adapters/          # sources and sinks (HTTP, file, SFTP)
│   ├── lineage/           # lineage store, retention, purge
│   ├── authz/             # authorization (RBAC/ABAC)
│   ├── observability/     # metrics, tracing, viewer
│   └── secrets/           # ${SECRET:name} resolution
├── pkg/sdk/               # public SPI
├── schemas/               # JSON Schema for routes
├── examples/              # worked examples
├── test/                  # fixtures and integration tests
├── deploy/                # Grafana dashboard-as-code
└── design/                # architecture docs
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

## Phase 0 scope

**Shipped:**
- Core engine and DAG executor with hot reload
- Steps: `filter`, `translate`, `route` (content-based router), `wiretap`, `idempotent`, `authorize` (RBAC/ABAC)
- Adapters: HTTP, file, SFTP
- JSONata expressions with plugin (Go) and WASM functions
- Data contracts: inline JSON Schema with enforcement and violation tracking
- Lineage store with retention policies, automatic reaper, and purge evidence log
- Error handling: retry, backoff, dead-letter, distinction of error classifications
- Observability: Prometheus metrics, OTel tracing, built-in Tier 1 viewer
- CLI: `validate`, `test`, `explain`, `trace tail`, `provenance`, `lineage export/purge`

**Phase 1 and later:**
- Kafka and AMQP adapters
- PBAC (policy-based access control) via formal PDP contract
- Schema-registry-backed contracts (Apicurio reference, Confluent/AWS/Azure support)
- OpenLineage export to catalogs (Marquez, Collibra, Atlan, Purview)
- OBO (on-behalf-of) token exchange
- Aggregator/splitter/claim-check steps
- Domain/namespace model
- Distributed control plane and multi-tenant isolation

See [phase-0-implementation-plan.md](design/phase-0-implementation-plan.md) (§2) for full details.

## Documentation

All design and planning docs are in `design/`:

- **[eip-middleware-design.md](design/eip-middleware-design.md)** — Core architecture, EIP mapping, all features, design tenets, open questions (v8)
- **[phase-0-implementation-plan.md](design/phase-0-implementation-plan.md)** — Engineering roadmap: 6 milestones, 50 subtasks, risk register, testing strategy (v5)
- **[data-mesh-feasibility-analysis.md](design/data-mesh-feasibility-analysis.md)** — How well does this design fit data mesh principles?
- **[data-mesh-reference-architecture.md](design/data-mesh-reference-architecture.md)** — Reference impl of a data mesh on top of dim; domain/namespace proposal
- **[self-service-feasibility-study.md](design/self-service-feasibility-study.md)** — How feasible is self-service config/deploy? (very; GitOps + guardrails)

## Getting started

1. **[Read the User Guide](USER_GUIDE.md)** for quick start and project structure
2. **Clone and build:**
   ```bash
   git clone https://github.com/naren-chakraview/dim.git
   cd dim
   go build ./cmd/dimd -o dimd
   go build ./cmd/midctl -o midctl
   ```
3. **Try an example:**
   ```bash
   ./dimd examples/order-processing.yaml &
   # (examples in Phase 0 scope; see USER_GUIDE.md)
   ```
4. **Run tests:**
   ```bash
   go test -race ./...
   ```
5. **Explore the design:**
   - Start with `design/eip-middleware-design.md` (§1–§3)
   - Then `design/phase-0-implementation-plan.md` for the roadmap
   - Then the reference architectures for data mesh and self-service use cases

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
