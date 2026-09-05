# dim — Developer Guide

**Status:** Phase 0 (v0.5.0) Complete ✅ | Phase 1 (v0.6.0-beta) Complete ✅ | Phase 2 (v0.7.0-beta) Complete ✅

For user-facing documentation, see [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) and [docs/LANGUAGE_REFERENCE.md](docs/LANGUAGE_REFERENCE.md).  
For architecture overview, see [README.md](README.md). For design decisions, see [OKF.md](OKF.md).

---

## Project structure overview

### Phase 0 — Complete ✅

| Directory | Purpose |
|---|---|
| `cmd/dimctl` | CLI tool — validate, run, lineage, trace, provenance |
| `internal/config` | YAML parsing and JSON Schema validation |
| `internal/engine` | Executor, message envelope, bounded channels, DLQ |
| `internal/steps` | Steps: filter, translate, authorize, idempotent, route |
| `internal/expr` | JSONata expression evaluation with pluggable functions |
| `internal/adapters/http` | HTTP source adapter (port 8080) |
| `internal/adapters/file` | File source/sink adapters with polling |
| `internal/lineage` | SQLite store, retention policies, purge evidence |
| `internal/observability` | OTel tracing, Prometheus metrics |
| `internal/factory` | Pipeline builder pattern |
| `schemas` | JSON Schema for route validation |
| `examples` | Sample route configurations and benchmarks |
| `test/fixtures` | Test fixture definitions |

### Phase 1 Track A — Complete ✅

| Directory | Purpose |
|---|---|
| `cmd/dimd` | Engine daemon with graceful shutdown (R9) |
| `internal/adapters/interfaces.go` | Source/Sink interface contracts (R17) |
| `internal/observability/otel_*` | Real OTel SDK with OTLP export (R7) |

### Phase 1 Track B — Complete ✅

| Directory | Purpose |
|---|---|
| `internal/adapters/kafka` | Kafka consumer/producer with consumer groups (R14) |
| `internal/adapters/amqp` | AMQP queue-based source/sink (R15) |
| `internal/adapters/s3` | S3 bucket polling and write (R16) |

### Phase 2 Track B — Complete ✅

| Directory | Purpose |
|---|---|
| `internal/steps/aggregate.go` | Aggregator step with state management (M2.1) |
| `internal/steps/split.go` | Splitter step with path-based routing (M2.2) |
| `internal/adapters/database` | JDBC sink, log-based & trigger-based CDC (M2.3) |
| `internal/config/fragments.go` | Fragment parameterization with late binding (M2.4) |
| `internal/steps/authorize.go` | Authorization obligations & redaction (M2.5) |
| `internal/lineage/purgelog.go` | Purge-log auto-export to S3 (M2.6) |
| `internal/validation/contract_check.go` | Static contract conformance checking (M2.7) |

---

## Building

```bash
git clone https://github.com/naren-chakraview/dim.git
cd dim
go build ./cmd/dimctl -o dimctl
go build ./cmd/dimd -o dimd  # optional daemon
```

## Testing

### Unit tests (Phase 0 — 440+ passing, Phase 1 expanding)
```bash
go test ./...
```

### Race detector (required for CI)
```bash
go test -race ./...
```

### Integration tests with broker detection
```bash
# Kafka (skips if broker unavailable)
go test ./internal/adapters/kafka -race -v

# AMQP (skips if broker unavailable)
go test ./internal/adapters/amqp -race -v

# S3 (skips if credentials unavailable)
go test ./internal/adapters/s3 -race -v

# Database adapters
go test ./internal/adapters/database -race -v
```

### Benchmarks
```bash
go test ./examples/bench -bench=. -benchmem -benchtime=10s
```

### Route fixture tests (Phase 0+)
```bash
./dimctl test examples/
```

### Test coverage by package
- internal/expr: 9/9 ✅
- internal/engine: 35/35 ✅
- internal/adapters/http: 8/8 ✅
- internal/adapters/file: 12/12 ✅
- internal/config: 15/15 ✅
- internal/steps: 26/26 ✅
- internal/factory: 7/7 ✅
- cmd/dimctl: 8/8 ✅

---

## Adding a new step type

1. Define step struct in `internal/steps/<name>.go`
2. Implement the step interface (`Execute()`, `Drain()`)
3. Add to the route schema in `schemas/route.schema.json`
4. Write fixture tests in `test/fixtures/`
5. Update `internal/factory/builder.go` to instantiate it

See `internal/steps/aggregate.go` (M2.1) for an example stateful step with hot-reload support.

## Adding a new adapter (source or sink)

1. Implement `adapters.Source` or `adapters.Sink` interface
2. Place in `internal/adapters/<type>/`
3. Register in `internal/factory/builder.go`
4. Write integration tests in `internal/adapters/<type>/<name>_test.go`

See `internal/adapters/database/database_sink.go` (M2.3) and `internal/adapters/kafka/kafka_source.go` (Phase 1) for examples.

## JSONata expressions

All expressions (`filter.expr`, `translate.expr`, `route.cases[].when`, `authorize` ABAC rules, etc.) are [JSONata](https://jsonata.org/). Evaluation happens in `internal/expr/evaluator.go`.

The functions registry (`internal/expr/functions.go`) allows both native Go plugins and WASM functions to be called from JSONata:

```yaml
functions:
  normalizePhone: { type: plugin, runtime: go, ref: plugins/normalize_phone }
  scoreRisk: { type: wasm, ref: plugins/risk_score.wasm }
```

Then in a `translate` expression:
```jsonata
{ phone: $normalizePhone(body.phone), risk: $scoreRisk(body) }
```

**Static contract checking (M2.7):** The system can now detect type mismatches and missing fields for a statically-analyzable JSONata subset. Limitations documented in `design/M2_7_STATIC_CONTRACT_CONFORMANCE.md`.

## Lineage and retention policies

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

The automatic reaper purges old records per policy; `dimctl lineage purge` triggers manual purge with an evidence log.

**Purge-log auto-export (M2.6):** Records marked for purge are automatically exported to S3 in JSONL format before deletion, providing evidence and an audit trail.

## Hot reload

Routes reload when their config changes; in-flight messages complete on the old DAG while new messages use the new DAG. No outage, no message loss. `route_version` tags every message so you know which DAG version processed it.

**Generation takeover pattern:** Hot reload uses a generation counter to track which worker generation owns a group of in-flight messages. See `internal/engine/executor.go` and OKF.md (§Hot reload) for details.

**Stateful step draining (M2.1, M2.2):** Aggregator and splitter steps preserve in-flight state across hot reloads, maintaining group membership and generation tracking.

## Observability

### Tracing

Routes emit OTel spans for:
- Message ingestion (source span)
- Each step execution
- Sink writes
- Authorization decisions (with PDP obligation details)
- Lineage recording

Export to Jaeger, Datadog, or any OTLP endpoint. See `internal/observability/otel.go`.

### Metrics

Prometheus metrics:
- `dim_messages_processed_total` — counter by route and outcome (success/error)
- `dim_message_latency_seconds` — histogram by route and step
- `dim_sink_writes_total` — counter by sink type
- `dim_authorization_decisions_total` — counter by result (allow/deny)
- `dim_lineage_records_stored_total` — counter by route

### Live viewer

Built-in HTTP endpoint (`/debug/routes`) shows real-time message counts and latency. See `internal/observability/viewer.go`.

## Configuration reference

See `design/eip-middleware-design.md` (§6) and `docs/LANGUAGE_REFERENCE.md` for the full config model and all available options.

---

## Understanding the codebase

Use graphify to explore the knowledge graph:

```bash
/graphify --update          # Re-extract graph after code changes
/graphify query "<question>"  # Traverse graph for answers
/graphify path "ModuleA" "ModuleB"  # Find shortest path between concepts
```

See `graphify-out/GRAPH_REPORT.md` for god nodes and community structure.

---

## Key design decisions

See [OKF.md](OKF.md) for:
- Language choice (Go)
- Config composition (imports, fragments, late binding)
- Hot reload with generation takeover
- Per-instance embedded lineage store
- Structural authorization declaration
- Concurrency patterns (channels, backpressure, executor pool)

---

## Debugging

### Explain a route's DAG
```bash
./dimctl explain examples/hello.yaml
```

### Tail live spans for a route
```bash
./dimctl trace tail hello
```

### Reconstruct a message's journey
```bash
./dimctl provenance <correlation-id>
```

### Export lineage for audit
```bash
./dimctl lineage export --route payment-processing --since 2026-08-01 --format csv
```

---

## Next steps

- Read `design/eip-middleware-design.md` for the full architecture
- Read `design/phase-*-implementation-plan.md` for the roadmap
- Browse `examples/` for worked examples
- See `docs/` for user-facing guides
