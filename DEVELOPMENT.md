# Development Guide

This document is for developers and coding agents working on dim. It covers the project structure, testing strategies, common tasks, and how to work efficiently with Phase 0 (complete) and Phase 1 (complete) codebase. Phase 2 is next.

## Quick reference

| Task | Command |
|---|---|
| **Build** | `go build ./cmd/dimctl -o midctl` |
| **Test** | `go test ./...` (120/121 passing) |
| **Test with race detector** | `go test -race ./...` |
| **Lint** | `go vet ./...` |
| **Understand architecture** | Read `graphify-out/GRAPH_REPORT.md` or open `graphify-out/graph.html` |
| **Update knowledge graph** | `graphify update .` |
| **Find where X is defined** | `grep -r "type X " internal/` or `graphify query "X"` |
| **Run a route** | `./dimctl run examples/test-route.yaml` |
| **Validate a route** | `./dimctl validate examples/test-route.yaml` |

## For coding agents: understand before coding

Before implementing a subtask, read these in order:

1. **The subtask spec** in `design/phase-0-implementation-plan.md` (§5)
   - What's the exit criterion? What exactly needs to work?
   - What does it depend on? (check the Dependency graph table)

2. **The design context** in `design/eip-middleware-design.md`
   - Find the section(s) the subtask implements (e.g., M0.1.6 → §4 "EIP mapping")
   - Understand the design intent, not just the mechanism

3. **The knowledge graph** via `graphify-out/`
   - Run `/graphify query "how does X relate to Y"` if unsure about connections
   - Check `GRAPH_REPORT.md` for god nodes (core concepts) and surprising connections

4. **Existing code** in `internal/` (the relevant package)
   - Search for similar patterns (grep, or `graphify explain <concept>`)
   - Understand the interfaces you're implementing

5. **Tests** in `test/` (fixtures and integration tests)
   - Read the expected behavior from test cases
   - Run tests to verify understanding

Then code. If you get stuck, query the graph or re-read the design spec — usually the answer is there.

## Architecture overview for Phase 0 (M0.1)

### Implemented Layers ✅

**Engine layer** (`internal/engine/`)
- `message.go` — Message envelope (Headers, Body, Metadata)
- `channel.go` — Bounded message queues with backpressure (configurable buffer)
- `executor.go` — Single-worker processor, step orchestration
- `dlq.go` — Dead-letter envelope wrapping failed messages

**Step layer** (`internal/steps/`)
- `filter.go` — Boolean predicate filtering (drop/pass)
- `translate.go` — JSONata body transformations
- `factory.go` — Step factory from config spec
- Each step implements `Execute(ctx, msg) (*Message, error)`

**Expression layer** (`internal/expr/`)
- `jsonata.go` — JSONata evaluation wrapper around blues/jsonata-go v1.5.4
- Pure Go, 90% JSONata spec compliance

**Adapter layer** (`internal/adapters/`)
- `http/http_source.go` — HTTP webhook listener (port 8080, path /ingest)
- `file/file_sink.go` — JSONL file output with auto-directory creation
- Adapter interfaces: `SourceAdapter`, `SinkAdapter`

**Configuration layer** (`internal/config/`)
- `loader.go` — YAML parsing and loading
- `schema.go` — Config data structures
- `route.schema.json` — JSON Schema validation

**Factory layer** (`internal/factory/`)
- `pipeline.go` — Builds end-to-end pipeline from config

### Deferred to M0.2+ ⏳

**Generation/hot reload** (`internal/route/` — M0.2)
- DAG compilation and versioning
- Hot reload with background draining
- Route versioning

**Reliability enhancements** (M0.2)
- Retry logic with backoff and jitter
- Error classification (retryable vs. non-retryable)
- Multi-sink error routing

**Lineage tracking** (`internal/lineage/` — M0.4)
- Embedded SQLite store
- Retention policy resolution
- Automatic reaper and purge mechanism
- Evidence log

**Authorization** (`internal/authz/` — M0.5)
- Principal extraction and propagation
- RBAC/ABAC enforcement
- Policy-based access control (PBAC)

**Observability** (`internal/observability/` — M0.3)
- Prometheus metrics
- OpenTelemetry tracing
- Built-in Tier 1 viewer
- RBAC and ABAC evaluation
- Obligation handling (redaction, etc.)

**Observability layer** (`internal/observability/`)
- OTel span emission (trace context + attributes)
- Prometheus metric counters
- Built-in Tier 1 viewer (in-memory ring buffer + web UI)

### Data flow

```
HTTP webhook / Kafka consumer / file poller
    ↓
Source adapter (opens connection, gets messages, attaches principal)
    ↓
Route registry (loads config, resolves imports, compiles DAG, computes route_version)
    ↓
Executor (runs DAG stages in order; each stage has a worker pool)
    ├─ Step: filter (JSONata predicate)
    ├─ Step: authorize (RBAC/ABAC, routes to on_deny on rejection)
    ├─ Step: translate (JSONata transformation)
    ├─ Step: route (conditional branching, fan-out to multiple sinks)
    └─ On error at any stage:
       └─ Error classifier (retryable? non-retryable? authorization? contract?)
          ├─ Retryable → retry with backoff, then dead-letter
          ├─ Non-retryable → dead-letter immediately
          ├─ Authorization denied → on_deny sink
          └─ Contract violation → on_violation sink
    ↓
Sink adapter (writes to Kafka, HTTP, file, etc.)
    ↓
Lineage store (records route_version, contract_version, principal, outcome)
    ↓
Metrics + tracing (OTel spans + Prometheus counters)
```

### Key patterns

**Channels and backpressure:**
- Messages flow through bounded channels between stages
- When a downstream stage is slow, the channel backs up → upstream worker blocks → natural flow control

**Generations for hot reload:**
- Old DAG keeps running as a "Draining" generation
- New DAG starts as "Active" immediately
- Old gen stops accepting new messages, finishes in-flight, tears down
- Messages stamped with which generation processed them

**Lineage as a first-class concern:**
- Every message processed writes a lineage event (not optional)
- Lineage events carry: route_version, contract_version, function versions, authorization decision
- Retention policies manage how long events are kept
- Evidence log proves deletions happened

**Authorization as structure:**
- Every route declares `auth: none` or carry `authorize` steps
- Denials route to a distinct sink (`on_deny`), not silent discard
- Auditable: denial events are lineage events

**Contracts as independent versioning:**
- `route_version` tracks logic changes
- `contract_version` tracks data schema changes
- They're independent: logic can change without schema changing, and vice versa

## Testing strategy

### Unit tests
```bash
go test ./internal/steps/translate_test.go
```
Test individual step logic, expression evaluation, retry classification. Fast, deterministic, colocated in `*_test.go` files.

### Route fixture tests
```bash
./dimctl test test/fixtures/
```
Test whole routes against golden input/output pairs. Fixtures are YAML files defining: input message, expected output per sink, expected error handling. This is the primary regression net for JSONata and step interactions.

### Integration tests
```bash
go test -race ./test/integration/
```
End-to-end routes against in-process adapters (httptest, temp files). Always use `-race` to catch concurrency bugs. Hot reload is integration-tested here.

### CI gate
- `go build ./...` (does it compile?)
- `go vet ./...` (obvious mistakes?)
- `golangci-lint run` (style/best practices?)
- `go test ./...` (unit tests)
- `go test -race ./...` (race detector on integration tests)

All required to pass before merge. No CI-green-only merges; human review is also required.

## Common development tasks

### Adding a new step type

1. Create `internal/steps/<name>.go` with a struct implementing the step interface
2. Add to the route schema in `schemas/route.schema.json`
3. Write fixture tests in `test/fixtures/<name>_test.yaml`
4. Register in `internal/engine/executor.go` (instantiate the step from the config)
5. Update `design/phase-0-implementation-plan.md` if it's a new subtask

Example: `filter.go` is a ~20-line step that takes a JSONata predicate and drops non-matching messages.

### Adding a new adapter (source or sink)

1. Implement `adapters.Source` or `adapters.Sink` interface in `internal/adapters/<type>/`
2. Handle connection lifecycle (open, close, retry, circuit breaker)
3. For sources: emit messages on a channel; for sinks: accept messages and write
4. Write integration tests in `test/integration/`
5. Register in the adapter factory

Example: HTTP adapter is in `internal/adapters/http/`. It listens on a port, accepts POST requests, converts them to messages.

### Debugging a route

Use `midctl` tooling:
```bash
./dimctl explain examples/hello.yaml          # See the compiled DAG
./dimctl test examples/hello_test.yaml        # Run fixtures
./dimctl validate examples/hello.yaml         # Check schema
./dimctl trace tail hello                     # Live span stream (Phase 1)
./dimctl provenance <correlation-id>         # Reconstruct journey (Phase 1)
```

Or inspect in code: add print statements or use a debugger. Go test `t.Logf()` is your friend.

### Understanding the codebase

First, check the knowledge graph:
```bash
/graphify query "how do routes get compiled"
/graphify explain "hot_reload"
/graphify path "http_source" "kafka_sink"
```

The graph shows relationships you might not find by grepping. It's especially useful for understanding:
- Which packages depend on which (build order matters)
- How data flows through the pipeline
- Where configuration is resolved
- How error paths connect to sinks

If the graph is stale, rebuild it:
```bash
/graphify --update
```

## Performance notes

- **Channels:** Use bounded queues (prevents unbounded memory growth). Default buffer size is per-route config.
- **Worker pools:** Default pool size is per-stage config. Tune based on load (more workers = more goroutines but more concurrency).
- **Lineage store:** Single writer goroutine serializes writes; readers are concurrent (SQLite WAL mode). At ~1000 msg/sec, writes should not be a bottleneck.
- **Hot reload:** Draining generations occupy memory until in-flight count reaches zero. Cap is default 3 concurrent draining gens per route to prevent memory leak if reloads happen too frequently.
- **JSONata:** Expression evaluation is the most likely CPU bottleneck. For high-throughput routes, keep expressions simple and push complex logic to registered functions (native Go or WASM).

## Security checklist

Before shipping code:
- [ ] No hardcoded credentials or secrets (use `${SECRET:name}` references)
- [ ] All user input validated (routes, config, expressions)
- [ ] Authorization gates checked (does the route declare `auth:`?)
- [ ] Error messages don't leak sensitive data (secrets redacted automatically)
- [ ] Race detector passes (`go test -race ./...`)
- [ ] No unchecked error returns (at least log them)

## Deployment

Phase 0 produces two binaries:
- `dimd` — the engine daemon
- `dimctl` — the CLI tool

Both are compiled with `go build` and produce static executables. No runtime dependencies (SQLite and WASM runtime are built in).

Ship as:
- Tarball with both binaries + example config
- Or Docker image with `FROM scratch` (minimal attack surface)
- Or in your existing Go binary build pipeline

See `design/phase-0-implementation-plan.md` (§6) for CI strategy.

## Phase progression

Current: Phase 0 (walking skeleton → hardening & release)
- M0.1: Core executor, filter, translate, HTTP/file, dead-letter
- M0.2: Reliability (retry, hot reload, route step, fragments)
- M0.3: Governance (authorization, contracts)
- M0.4: Lineage (store, retention, purge)
- M0.5: Observability (tracing, metrics, viewer)
- M0.6: Hardening & release

Phase 1 (planned):
- Kafka/AMQP adapters
- PBAC (policy-based access control) via OPA
- Schema registry integration (Apicurio)
- OpenLineage export
- OBO (on-behalf-of) token exchange

See `design/phase-0-implementation-plan.md` (§19) for full roadmap.

---

**Last updated:** 2026-09-01  
**For questions:** Read the design docs first; check the knowledge graph second; grep third.
