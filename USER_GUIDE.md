# dim — User Guide

**Status:** Phase 0 (M0.1) Walking Skeleton. See [M0.1_COMPLETE.md](M0.1_COMPLETE.md) for full documentation.

## Quick start

### Building

```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
go build ./cmd/midctl -o midctl
```

### Your first route

1. Create `examples/hello.yaml`:
```yaml
version: 1
sources:
  http-source:
    type: http
sinks:
  output:
    type: file
    path: ./output/messages.jsonl
  errors:
    type: file
    path: ./output/errors.jsonl
routes:
  hello:
    from: http-source
    auth: none
    error_path:
      target: errors
    steps:
      - filter:
          expr: "amount > 0"
      - translate:
          expr: "{ greeting: 'Hello', message: name, total: amount * 1.1 }"
```

2. Validate:
```bash
./midctl validate examples/hello.yaml
```

3. Run:
```bash
./midctl run examples/hello.yaml
# In another terminal:
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"name":"world","amount":100}'
# Check output:
tail -f output/messages.jsonl
tail -f output/errors.jsonl
```

## Project structure overview

### Phase 0 (M0.1) — What's Implemented ✅

| Directory | Purpose |
|---|---|
| `cmd/midctl` | CLI tool — validate, run routes |
| `internal/config` | YAML parsing and JSON Schema validation |
| `internal/engine` | Executor, message envelope, bounded channels, dead-letter queue |
| `internal/steps` | Step implementations: filter, translate |
| `internal/expr` | JSONata expression evaluation |
| `internal/adapters` | HTTP source, file sink adapters |
| `internal/factory` | Pipeline builder pattern |
| `schemas` | JSON Schema for route validation |
| `examples` | Sample route configurations |
| `test/fixtures` | Test fixture definitions |

### M0.2+ — Deferred

| Directory | Purpose | Phase |
|---|---|---|
| `cmd/dimd` | Engine daemon | M0.2 |
| `internal/route` | Route model, DAG compilation | M0.2 |
| `internal/authz` | Authorization (RBAC/ABAC) | M0.5 |
| `internal/lineage` | Lineage store, retention, purge | M0.4 |
| `internal/observability` | Metrics, tracing, viewer | M0.3 |
| `pkg/sdk` | Public SPI for extensions | M0.2+ |

## Testing

### Unit tests (Phase 0 — 120/121 passing)
```bash
go test ./...
```

### Race detector (required for CI)
```bash
go test -race ./...
```

### Route fixture tests (M0.2+)
```bash
./midctl test examples/
```

### Test coverage by package
- internal/expr: 9/9 ✅
- internal/engine: 35/35 ✅
- internal/adapters/http: 8/8 ✅
- internal/adapters/file: 12/12 ✅
- internal/config: 15/15 ✅
- internal/steps: 26/26 ✅
- internal/factory: 7/7 ✅
- cmd/midctl: 8/8 ✅

## Phase 0 Features

### Supported Step Types
- ✅ **filter** — Boolean predicates (drop/pass messages)
- ✅ **translate** — JSONata body transformations
- ⏳ **route** — Content-based routing (M0.2)
- ⏳ **wiretap** — Copy to secondary sink (M0.2)
- ⏳ **idempotent** — Deduplication (M0.2)
- ⏳ **authorize** — RBAC/ABAC enforcement (M0.5)

### Configuration Format

```yaml
version: 1

sources:
  <name>:
    type: http|file|sftp|exec
    # adapter-specific config

sinks:
  <name>:
    type: http|file|sftp|exec
    # adapter-specific config

routes:
  <name>:
    from: <source-name>
    auth: none  # or authorization object (M0.5+)
    error_path:
      target: <sink-name>  # where failed messages go
      retry:  # optional
        max_attempts: 3
        backoff_ms: 1000
    steps:
      - filter:
          expr: "<jsonata-expression>"
      - translate:
          expr: "<jsonata-expression>"
```

## Key developer notes

### Phase 0 Constraints (by design)
- Single-worker processing (M0.2: worker pools)
- HTTP source hardcoded to port 8080, path /ingest (M0.2: configurable)
- File sink output to ./output/ (M0.2: configurable)
- No retry logic (M0.2: retry with backoff)
- No authorization enforcement (M0.5: RBAC/ABAC)
- No lineage tracking (M0.4: SQLite store)
- No observability (M0.3: metrics/tracing)

### Adding a new step type (M0.2+)
1. Define step struct in `internal/steps/<name>.go`
2. Implement the step interface
3. Add to the route schema in `schemas/route.schema.json`
4. Write fixture tests in `test/fixtures/`
5. Update `internal/engine/executor.go` to instantiate it

### Adding a new adapter (source or sink)
1. Implement `adapters.Source` or `adapters.Sink` interface
2. Place in `internal/adapters/<type>/`
3. Register in the adapter factory
4. Write integration tests in `test/integration/`

### JSONata expressions
All expressions (`filter.expr`, `translate.expr`, `route.cases[].when`, `authorize` ABAC rules, etc.) are JSONata. See `internal/expr/jsonata.go`.

The functions registry (`internal/expr/functions.go`) allows both native Go plugins and WASM functions to be called from JSONata:
```yaml
functions:
  normalizePhone: { type: plugin, runtime: go, ref: plugins/normalize_phone }
  scoreRisk: { type: wasm, ref: plugins/risk_score.wasm }
```

Then in a `translate` expression: `{ phone: $normalizePhone(body.phone) }`

### Lineage and retention policies
Routes declare retention policies for their lineage records:

```yaml
routes:
  payment-processing:
    lineage: { retention_policy: pci }  # static assignment
    # OR
    lineage:
      retention_policy: default
      retention_policy_expr: 'body.card_number ? "pci" : "default"'  # dynamic per message
      subject_id_expr: 'principal.subject ? principal.subject : body.customer_id'
```

The automatic reaper purges old records per policy; `midctl lineage purge` triggers manual purge with an evidence log.

### Hot reload
Routes reload when their config changes; in-flight messages complete on the old DAG while new messages use the new DAG. No outage, no message loss. `route_version` tags every message so you know which DAG version processed it.

## Debugging

### Explain a route's DAG
```bash
./midctl explain examples/hello.yaml
```

### Tail live spans for a route
```bash
./midctl trace tail hello
```

### Reconstruct a message's journey
```bash
./midctl provenance <correlation-id>
```

### Export lineage for audit
```bash
./midctl lineage export --route payment-processing --since 2026-08-01 --format csv
```

## Configuration reference

See `design/eip-middleware-design.md` (§6) for the full config model. A route YAML file has:
- `version` — config format version
- `sources` — named message inputs
- `sinks` — named message outputs
- `routes` — pipelines from source through steps to sink(s)
- `steps` — ordered sequence of filters, translators, routers, etc.
- `error_path` — where failed/retried messages go
- `auth` — authorization declaration (mandatory)
- `lineage` — retention policies and evidence tracking
- `functions` — registered custom functions (Go plugins or WASM)
- `imports` — reusable fragments from other files

All secrets are `${SECRET:name}` references, never inline.

## What's not in Phase 0

These ship in Phase 1 or later:
- Kafka and AMQP adapters (Phase 1)
- PBAC (policy-based access control) via the PDP contract (Phase 1)
- Schema-registry-backed data contracts; inline JSON Schema only for now (Phase 1)
- OpenLineage export (Phase 1)
- Aggregator/splitter/claim-check steps (Phase 2)
- Domain/namespace labeling (proposal, not Phase 0)
- Control-plane API and multi-tenant isolation (Phase 3)

See `design/phase-0-implementation-plan.md` (§2) for the full list.

## Next steps

- Read `design/eip-middleware-design.md` for the architecture
- Read `design/phase-0-implementation-plan.md` for the engineering roadmap
- Browse `examples/` for worked examples
- Run tests and explore the codebase
