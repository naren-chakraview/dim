# dim — Object Knowledge Framework

A living document for tracking business intent, architectural decisions, concurrency patterns, state management, security validation, and change history. This grows alongside the codebase as each phase is implemented.

## Business intent

**Problem:** Integration middleware today forces a choice between powerful-but-heavyweight (NiFi, Camel, Airflow) and lightweight-but-limited (bash scripts, single-purpose tools). Teams building data infrastructure need observability, lineage, and reliability built-in from the start, not bolted on later.

**Solution:** A declarative, single-binary, YAML-first router built on EIP primitives. Config is source of truth. Every route declares its error handling, authorization, and retention stance. Observability and lineage are automatic, not opt-in. Supports both streaming and batch via scheduling.

**Target users:** Platform teams and domain engineers building self-serve integration pipelines, especially in data mesh contexts. Teams that already use Kafka, want OpenLineage catalogs, or need audit trails.

**Success measures:**
- Routes defined in YAML, reviewed in PRs, tested locally
- Hot reload without outages or message loss
- Every message's journey reconstructable from lineage
- Compliance via retention policies + evidence logs
- Self-serve debugging (built-in viewer, `midctl trace tail`, `midctl provenance`)

## Architectural decisions

### Language: Go (not Rust, not Node.js)

**Decision:** Build in Go.

**Rationale:**
- Single static binary (single `go build` produces deployable artifact, no runtime install step)
- Goroutines and channels map directly to the pipeline model (concurrency primitives are native)
- `wazero` (WASM runtime) is pure Go, no cgo (matters for single-binary goal)
- `modernc.org/sqlite` (embedded store) is pure Go, no cgo
- OTel, Prometheus, OPA ecosystems are first-party-maintained in Go
- OPA itself (Phase 1 PDP reference) is Go, embeddable as a library

**Tradeoff:** Rust would be faster; TypeScript/Node would be more familiar to some teams. Go wins on ecosystem fit and distribution simplicity.

**Related:** See phase-0-implementation-plan.md §3 for full stack decisions (JSONata library, WASM runtime, plugin transport, lineage store, etc.).

### Config composition: imports and fragments

**Decision:** Routes can `import` reusable config files; those files can define `fragments` (named, unparameterized, literal step lists).

**Rationale:**
- Allows domain teams to compose shared governance fragments (e.g., mandatory auth checks) without code duplication
- Unparameterized in Phase 0 to keep scope bounded; parameterization deferred to Phase 2
- Fragment inclusion is compositional: a route compiles into a single resolved config before DAG generation
- Supports the "mandatory governance fragment" pattern (data-mesh-reference-architecture.md §5)

**Enforceability:** `midctl validate` can be extended (Phase 1) to lint for required fragments, making governance computational not just aspirational.

### Plugin vs. WASM for custom functions

**Decision:** Both are first-class from Phase 0 start. No forced default.

**Rationale:**
- Native Go plugins: performance-sensitive, trusted logic, direct access to Go ecosystem
- WASM: sandboxed, language-agnostic, distributable independently of `dimd` builds
- Plugin transport: `hashicorp/go-plugin` (subprocess RPC), not Go's built-in `plugin` package (fragile across toolchain versions)

**Tradeoff:** `hashicorp/go-plugin` has higher per-call latency than in-process `.so` calls, but avoids the fragility and keeps plugins independently deployable.

### Hot reload: new generation takeover with background draining

**Decision:** On config change, new DAG takes traffic immediately; old DAG drains stragglers in the background until in-flight count reaches zero. No pause, no message loss, no abort.

**Rationale:**
- New routes/changes go live immediately without a maintenance window
- In-flight messages complete with correct semantics (old DAG's version of the logic)
- `route_version` stamped on every message so you know which DAG processed it
- Safety valve: cap concurrent draining generations per route (default 3); beyond that, queue the reload

**Implementation note:** Requires careful state machine and goroutine lifecycle management. Highest-risk item in M0.2.10; `-race` tests are standing CI gate.

### Lineage store: per-instance, embedded, not shared

**Decision:** Each `dimd` instance runs its own embedded SQLite lineage store. No shared backend.

**Rationale:**
- Zero infrastructure requirement; single binary stays single binary
- Each instance fully self-sufficient for its own provenance queries
- Cross-instance aggregation handled on demand (`midctl lineage export`, sent to OpenLineage catalog)
- Per-instance retention policies + evidence logs; no coordination protocol needed

**Retention:** Named, configurable policies (default, pci, phi, public, custom). Static assignment (preferred) or dynamic per message (for mixed-sensitivity routes). Fail-open to `default` on policy resolution failure.

**Purge:** Automatic reaper (background job per policy cadence) + manual trigger (`midctl lineage purge`, including subject targeting). Every purge appends to append-only evidence log (separate, longer retention, own expiry warnings).

**Tradeoff:** Per-instance lineage means no "all instances" query without aggregation. Accepted because: (a) most queries are within one instance; (b) cross-instance is solved by export; (c) distributed store is Phase 3.

### Authorization: structural declaration, opt-in enforcement

**Decision:** Routes must declare `auth: none` or carry at least one `authorize` step. Declaration is validated at **warning level by default**. A team explicitly flips `validation.auth_declaration: enforce` to make it a hard failure.

**Rationale:**
- Visible guardrail: you can't accidentally ship unauthenticated route (you get a warning)
- Opt-in escalation: no engine-driven timeline change, no version bump surprise
- Per-team control: each team decides when to tighten (RBAC first, then PBAC once policies are written)

**Implementation:** RBAC (require_roles) and ABAC (JSONata predicates) ship in Phase 0. PBAC (policy engine) with formal PDP contract deferred to Phase 1 as a single schema+implementation unit (not stubbed in Phase 0 schema).

**Related:** principal propagation is structural too (attached to message metadata by source adapter, available to all downstream expressions).

### Data contracts: inline first, registry second

**Decision:** Phase 0 contracts are inline JSON Schema only (file beside route YAML). Registry-backed contracts (Apicurio reference, Confluent/AWS/Azure support) defer to Phase 1.

**Rationale:**
- Inline is zero-infrastructure: version in same repo as route
- Schema is DDD artifact: belongs with the domain team's config, not in a centralized registry first
- Registry upgrade path: once schema reuse and discovery matter, registry becomes valuable; doesn't require redesign

**Enforcement:** `enforce: true` on source/sink validates at boundary; violation is `contract_violation` classification (non-retryable, routes to `on_violation`). Both `route_version` and `contract_version` stamped on lineage so you can tell "logic change" apart from "schema change."

**Related:** Static conformance checking (`midctl validate`-time checks on `translate` output shape) is a best-effort convenience, not a substitute for runtime enforcement.

### Error classifications: distinct failure types matter

**Decision:** Four distinct failure classifications, each with its own handling:
1. **Retryable** (timeout, 5xx): retry with backoff, then dead-letter
2. **Non-retryable** (4xx, malformed input): no retry, straight to dead-letter
3. **Authorization denied** (`authorize` step): no retry, routes to `on_deny` (distinct sink, auditable outcome)
4. **Contract violation** (schema mismatch): no retry, routes to `on_violation` (distinct classification, tells compliance stories)

**Rationale:**
- Each type needs different handling (retry vs. no-retry)
- Each type has different meaning for observability/debugging (is this a real error, or a policy?)
- Dead-letter envelope carries `error_type` + `contract_version` (where applicable) so downstream can categorize

### Secrets: never inline, always resolved

**Decision:** All credentials/API keys use `${SECRET:name}` syntax, resolved by a pluggable secret provider (env, file, Vault-style) at config load time.

**Rationale:**
- Never a secret in the YAML file itself (not in the repo, not in version control)
- Pluggable so different teams can use their existing secret infrastructure
- Resolved at load time, not runtime, for early failure
- Logged/traced values are never secrets (redacted automatically)

## Concurrency patterns

### Channels and backpressure

**Pattern:** Bounded message channels between pipeline stages. Backpressure propagates upstream when a downstream stage is slower than upstream.

**Implementation:** `internal/engine/channel.go` defines a bounded queue. When full, the configured overflow policy applies (block or drop, configurable per route).

**Rationale:** Prevents unbounded memory growth under slow-sink scenarios; natural feedback loop encourages upstream to slow down or scale horizontally.

### Executor worker pool

**Pattern:** Configurable worker pool per stage. Multiple messages can be in flight simultaneously, but stage execution is serialized per message within a worker (no message interleaving).

**Implementation:** `internal/engine/executor.go` spawns a configurable number of worker goroutines per stage, pulling messages from an input channel and pushing results to output channels.

**Why this shape:** Allows parallelism across messages while keeping each message's journey deterministic (no race between parallel executions of the same message).

### Hot reload: generation state machine

**Pattern:** Old and new DAG generations coexist briefly. New gen takes new traffic immediately; old gen drains in-flight messages to completion, then tears down.

**State machine** (defined in generation.go):
```
[*] --> Compiling (config change detected)
     ├--> Rejected (validation fails)
     │      └--> [*] (old generation keeps running)
     └--> Active (validation passes)
              └--> Draining (superseded by newer generation)
                   └--> TornDown (in-flight count reaches zero)
                        └--> [*]
```

**Rationale:** No pause, no abort, no message loss. Keeps semantics correct (each message sees the DAG version that actually processed it).

**Concurrency:** Requires careful goroutine lifecycle (spawn draining gen, spawn monitoring goroutine, tear down when in-flight hits 0). Highest-risk item in M0.2.10; standing CI gate with `-race`.

### Lineage store writer serialization

**Pattern:** Single dedicated writer goroutine serializes all lineage writes; multiple readers can access concurrently (SQLite WAL mode).

**Implementation:** `internal/lineage/store.go` spawns a writer loop that serializes all writes through a channel. Readers call `Query()` directly on the store, which acquires read locks.

**Rationale:** Avoids SQLite's "database is locked" errors under concurrent writes while keeping reads fast.

## State management

### Route state during compilation

**Lifecycle:**
1. YAML loaded, `imports` resolved
2. Fully resolved config validated against schema
3. Route DAG compiled from step declarations
4. `route_version` (content hash) computed
5. DAG generation created (in `Active` state)

**Mutation:** Route state is immutable once compiled. Changes flow through a new compilation, new generation.

### Message state through pipeline

**Envelope carries:**
- `headers` (mutable by steps, e.g., `translate`)
- `body` (mutable by steps)
- `metadata` (immutable, set at ingest):
  - correlation ID, timestamp, route/stage provenance
  - `route_version` (which DAG processed this)
  - `contract_version` (where applicable)
  - `principal` (authenticated identity, if present)

**Safety:** Steps receive the current envelope state; mutations are local to that step's output channel.

### Generation lifecycle tracking

**Per-generation counters:**
- In-flight message count (incremented on ingress, decremented on egress)
- Concurrent draining generation cap (default 3 per route)

**Cleanup:** When in-flight hits 0, generation tears down (closes channels, releases resources).

## Security validation

### Expression safety

**Pattern:** JSONata expressions are pure (no side effects, no I/O). Stateful logic (database lookups, external API calls) must be functions, not inline expressions.

**Rationale:** Prevents accidental complexity, keeps expressions auditable.

### Authorization is mandatory

**Pattern:** Every route declares `auth: none` or `authorize` steps. Missing declaration is a warning (Phase 0) or hard failure (team opt-in).

**Rationale:** Forces explicit thought about who can access a route's data.

### Secrets redaction in logs and traces

**Pattern:** Anything sourced from `${SECRET:name}` or marked sensitive by authorization obligations is automatically redacted in logs and span attributes.

**Implementation:** Configurable at log level. Trace attribute redaction handled by span processors.

### Authorization obligations

**Pattern:** PDP (policy decision point) can return allow/deny + obligations (e.g., "allow, but redact SSN"). Obligations are applied to outbound messages.

**Rationale:** Lets policy engine impose data-protection rules without modifying the route config.

## Change history

**Phase 0 design:**
- v1–v8 of eip-middleware-design.md produced through iterative review, resolving open questions each round
- v1–v5 of phase-0-implementation-plan.md refined execution strategy (agent-driven, tiered review, risk register)
- Data mesh feasibility analysis → data mesh reference architecture (domain model proposal)
- Self-service feasibility study → GitOps + guardrails roadmap

**This scaffold (2026-09-01):**
- Go module initialized
- Directory structure laid out per phase-0-implementation-plan.md §4
- Empty Go files with package declarations (ready for implementation)
- GitHub Actions CI skeleton
- Design docs checked in
- User guide and README created
- OKF started

**Next phases:**
- M0.1 (walking skeleton): JSONata spike, minimal executor, filter/translate, HTTP/file adapters, dead-letter
- M0.2 (reliability): retry, hot reload, route step, fragments, `route_version`
- M0.3 (governance): authorization (RBAC/ABAC), contracts, `contract_version`
- M0.4 (lineage): embedded store, retention, reaper, purge, export
- M0.5 (observability): tracing, metrics, Tier 1 viewer, Grafana dashboard
- M0.6 (release): hardening, `-race` clean suite, cross-compile, v0.1.0 tag

## Known risks and mitigations

See phase-0-implementation-plan.md §9 for full risk register. Key items:

1. **JSONata Go library fidelity** — M0.1 spike comparing libraries against spec test suite; fallback to WASM-embedded JS if needed
2. **WASM ABI stability** — M0.2 spike to settle calling convention before load-bearing usage
3. **SQLite contention** — WAL mode + single writer goroutine; load testing in M0.4
4. **Hot-reload races** — `-race`-enabled integration tests as standing CI gate from M0.2 onward
5. **Plugin portability** — `hashicorp/go-plugin` (subprocess RPC) avoids Go toolchain version fragility

## Open questions (to resolve as implementation proceeds)

- Should `subject_id` be inferred automatically from `principal`, or always explicit? (design §18.1)
- Is alerting sufficient for purge-log expiry, or auto-export? (design §18.2)
- Is static contract conformance checking worth the partial-coverage risk? (design §18.3)
- Rename schema-registry `subject` to avoid collision with privacy `subject_id`? (design §18.4)

---

**Last updated:** 2026-09-01 (Phase 0 scaffold)
**Maintainer:** Naren Chakraview with Claude Code
