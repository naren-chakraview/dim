# dim — Developer Guide

**Status:** Phase 0 (v0.5.0) Complete ✅ | Phase 1 (v0.6.0-beta) Complete ✅ | Phase 2 (v0.7.0-beta) Complete ✅ | Phase 3 (v0.8.0-beta) Complete ✅

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

### Phase 3 Track A — Complete ✅

| Directory | Purpose |
|---|---|
| `internal/cluster` | Distributed deduplication (PostgreSQL backend) and lineage storage (M3.1) |
| `internal/tenant` | Rate limiting and worker slot management per domain (M3.4) |
| `internal/adapters/claimcheck` | Claim-check store (in-memory, S3-backed) for large payloads (M3.2) |
| `internal/steps/claim_check.go` | Claim-check step — extract and store large payloads (M3.2) |
| `internal/steps/claim_resolve.go` | Claim-resolve step — retrieve claim-checked payloads (M3.2) |
| `pkg/sdk` | Plugin SDK with native-Go and WASM support (M3.3) |
| `deploy/docker-compose.e2e.yml` | KRaft-mode Kafka, PostgreSQL, MinIO for e2e testing (M3.5) |
| `.github/workflows/e2e.yml` | CI/CD gating with real service integration tests (M3.5) |

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

### Local integration tests with Docker
```bash
# Run full e2e environment (Kafka, PostgreSQL, Zookeeper)
scripts/e2e-test.sh

# Or manage services manually
cd deploy
docker-compose -f docker-compose.e2e.yml -p dim-e2e up -d
# ... run your tests ...
docker-compose -f docker-compose.e2e.yml -p dim-e2e down -v
```

See **[docs/E2E_TESTING.md](docs/E2E_TESTING.md)** for complete integration testing guide.

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
- internal/adapters/claimcheck: 8/8 ✅ (M3.2)
- internal/config: 15/15 ✅
- internal/steps: 32/32 ✅ (includes claim-check, claim-resolve)
- internal/factory: 7/7 ✅
- internal/cluster: 12/12 ✅ (M3.1 - dedup, lineage)
- internal/tenant: 6/6 ✅ (M3.4 - rate limiting, slots)
- cmd/dimctl: 8/8 ✅
- pkg/sdk: 5/5 ✅ (M3.3 - plugin SDK)

---

## Adding a new step type

1. Define step struct in `internal/steps/<name>.go`
2. Implement the step interface (`Execute()`, `Drain()`)
3. Add spec to config schema in `internal/config/schema.go`
4. Add to step factory counter and switch case in `internal/steps/factory.go`
5. Write unit tests in `internal/steps/<name>_test.go`
6. Write integration tests in `internal/steps/<name>_integration_test.go`

See `internal/steps/aggregate.go` (M2.1) for stateful step with hot-reload; `internal/steps/claim_check.go` (M3.2) for claim-check step with external store integration.

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

## Plugin system (M3.3)

Routes register custom functions at the top level:

```yaml
version: 1

functions:
  validate_cc:
    type: plugin
    runtime: native
    ref: ./plugins/validate_cc
  encrypt_data:
    type: plugin
    runtime: wasm
    ref: ./plugins/encrypt_data.wasm
```

Plugins are called from JSONata expressions: `$validate_cc(card_number)`, `$encrypt_data(ssn)`.

**Native Go plugins:** Built as standalone binaries using `pkg/sdk`. See `examples/plugin/native-go/` for reference implementations.

**WASM plugins:** Compiled to WebAssembly from Rust, C, or other languages. Sandboxed in-process execution. See `examples/plugin/wasm/` for examples.

See `design/phase-3-implementation-plan.md` (§M3.3) for SDK architecture.

## Multi-tenant isolation (M3.4)

Routes can declare a `domain` field to associate them with a tenant:

```yaml
routes:
  customer-a-orders:
    domain: customer-a
    from: api
    # ...
```

The executor enforces per-tenant quotas (message rate limits and worker slot allocation) via `tenant.MessageRateLimiter` and `tenant.WorkerSlotManager`. This prevents one tenant's traffic from degrading another's.

**Factory wiring (pending):** The factory must be updated to pass tenant manager through the pipeline, creating rate limiters and slot managers for each unique domain. See `design/PHASE-3-COMPLETION-REPORT.md` (§ M3.4 Factory Wiring).

## Distributed clustering (M3.1)

Routes in cluster mode use shared storage to eliminate duplicate processing. Cluster state is initialized from environment:

```bash
export DIMD_CLUSTER_HOSTS="postgres://user:pass@host1:5432/dim,postgres://user:pass@host2:5432/dim"
```

The `cluster.PostgresDedupStore` deduplicates messages across instances, and `cluster.PostgresLineageBackend` provides unified lineage visibility.

**How it works:**
- Message arrives at instance A → dedup store records ID
- Same message arrives at instance B → dedup store recognizes duplicate, skips processing
- Lineage recorded in shared PostgreSQL (queryable cluster-wide via `dimctl lineage`)

See `internal/cluster/postgres_dedup_store.go` and `design/M3.1.1_DISTRIBUTION_MODEL.md` for implementation.

## Claim-check pattern (M3.2)

Routes can use claim-check and claim-resolve steps to handle large payloads efficiently:

```yaml
routes:
  order-with-attachments:
    from: api
    steps:
      - claim_check:
          payload_field: attachments    # Field to store externally
          remove_payload: true          # Remove from message
          ticket_field_path: _claim_check
          content_type: application/octet-stream
    sinks:
      - kafka
```

The store abstraction (`internal/adapters/claimcheck/store.go`) supports in-memory (testing) and S3 backends. Messages carry lightweight tickets (~100 bytes) instead of full payloads (~5MB+).

See `internal/steps/claim_check.go` and `internal/adapters/claimcheck/` for implementation.

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
- Read `design/PHASE-3-COMPLETION-REPORT.md` for Phase 3 summary and follow-up work
- Browse `examples/` for worked examples
- See `docs/` for user-facing guides (especially `docs/PHASE3_FEATURES.md` for end-user advanced features)

## Phase 3 Follow-up Work (Maintainers Only)

Per `design/PHASE-3-COMPLETION-REPORT.md`, these follow-ups enable production-ready deployment:

1. **M3.4 Factory Wiring** (HIGH priority) — Wire tenant manager through pipeline
   - Files: `internal/factory/pipeline.go`, executor creation
   - Status: Architect pending; wiring enables M3.4 end-to-end

2. **M3.2 S3 Backend** (MEDIUM priority) — Replace in-memory claim-check store
   - Files: Create `internal/adapters/claimcheck/s3_store.go`
   - Status: Enables production-scale payload handling

3. **M3.1 Cluster Demo** (MEDIUM priority) — Multi-instance dedup verification
   - Status: End-to-end test with two instances needed

4. **Final Re-Verification** (AFTER follow-ups) — Validate all exit criteria
