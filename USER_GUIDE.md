# dim — User Guide

**Status:** Phase 0 scaffold. This guide evolves alongside the codebase.

## Quick start

### Installing from source

```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
go build ./cmd/dimd -o dimd
go build ./cmd/midctl -o midctl
```

### Your first route

1. Create `examples/hello.yaml`:
```yaml
version: 1
sources:
  input: { type: http, path: /input, method: POST }
sinks:
  output: { type: file, path: /tmp/output.jsonl }
routes:
  hello:
    from: input
    auth: none
    error_path:
      target: output
      retry: { max_attempts: 1 }
    steps:
      - translate: { expr: '{ greeting: "Hello", message: body }' }
```

2. Run:
```bash
./dimd examples/hello.yaml &
curl -X POST http://localhost:8080/input -d '{"name":"world"}'
cat /tmp/output.jsonl
```

## Project structure overview

| Directory | Purpose |
|---|---|
| `cmd/dimd` | Engine daemon — listens for messages, processes routes |
| `cmd/midctl` | CLI tool — validate, test, explain, debug routes |
| `internal/config` | YAML parsing, imports/fragments resolution |
| `internal/route` | Route model, DAG compilation, route_version hashing |
| `internal/engine` | Core executor: channels, backpressure, hot reload |
| `internal/steps` | EIP step implementations: filter, translate, route, authorize, etc. |
| `internal/expr` | Expression evaluation: JSONata, functions registry, WASM/plugin runtimes |
| `internal/adapters` | Message sources and sinks: HTTP, file, SFTP |
| `internal/lineage` | Data lineage store, retention policies, purge mechanism, evidence log |
| `internal/authz` | Authorization: principal propagation, RBAC/ABAC |
| `internal/observability` | Metrics, tracing, built-in viewer |
| `pkg/sdk` | Public SPI for custom functions and adapters |

## Testing

### Unit tests
```bash
go test ./internal/...
```

### Integration tests (race-clean)
```bash
go test -race ./test/integration/...
```

### Route fixture tests
```bash
./midctl test examples/
```

## Key developer notes

### Adding a new step type
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
