# Development Guide

This document is for developers and coding agents working on dim. It covers the project structure, testing strategies, common tasks, and how to work efficiently with the codebase.

## Quick reference

| Task | Command |
|---|---|
| **Build** | `go build ./cmd/dimd -o dimd` |
| **Test** | `go test -race ./...` |
| **Lint** | `go vet ./...` && `golangci-lint run` |
| **Understand architecture** | Read `graphify-out/GRAPH_REPORT.md` or open `graphify-out/graph.html` |
| **Update knowledge graph** | `/graphify --update` (or `graphify --update .` if CLI) |
| **Find where X is defined** | `grep -r "type X " internal/` or query the graph: `graphify query "where is X defined"` |

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

## Architecture overview for agents

### Core layers

**Engine layer** (`internal/engine/`)
- `channel.go` — bounded message queues with backpressure
- `executor.go` — worker pool, stage routing, message processing
- `generation.go` — DAG generations, hot reload, draining lifecycle

**Step layer** (`internal/steps/`)
- Each step type (filter, translate, route, authorize, etc.) is a separate file
- All steps receive a message envelope, optionally transform it, pass to output channel(s)
- Steps are pure functions (no I/O, no state mutations beyond the message)

**Expression layer** (`internal/expr/`)
- `jsonata.go` — JSONata expression evaluation
- `functions.go` — function registry (both native plugins and WASM)
- `wasm_runtime.go`, `plugin_runtime.go` — language-specific runtimes

**Adapter layer** (`internal/adapters/`)
- `source.go`, `sink.go` — interfaces for message producers and consumers
- `http/`, `file/` — concrete implementations
- Adapters handle connection lifecycle, retry, circuit breaking

**Reliability layer** (`internal/errorpath/`)
- Retry classification (retryable vs. non-retryable)
- Backoff scheduling and jitter
- Dead-letter envelope construction
- Error routing (to dead-letter, denial, or violation sinks)

**Lineage layer** (`internal/lineage/`)
- Embedded SQLite store (one writer, multiple readers via WAL mode)
- Retention policy resolution (static or dynamic per message)
- Automatic reaper background job
- Manual purge with evidence log
- CSV/NDJSON export

**Authorization layer** (`internal/authz/`)
- Principal extraction from HTTP headers (JWT validation)
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
./midctl test test/fixtures/
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
./midctl explain examples/hello.yaml          # See the compiled DAG
./midctl test examples/hello_test.yaml        # Run fixtures
./midctl validate examples/hello.yaml         # Check schema
./midctl trace tail hello                     # Live span stream (Phase 1)
./midctl provenance <correlation-id>         # Reconstruct journey (Phase 1)
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
- `midctl` — the CLI tool

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
