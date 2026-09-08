# Phase 3 Remediation Task List — for Claude Code (WSL)

**Purpose:** Fix the gaps found in `design/phase-3-review-findings.md` (reviewed at commit `81367a3`, tag `v0.9.1`) before Phase 4 starts. This is a task list to hand to Claude Code, not a new design document — each section names the defect, the concrete files to touch, and how to prove it's actually fixed this time (reachable from the real running system, not just "unit tests pass").

**How to use this:** Work top to bottom or milestone by milestone — each section is self-contained enough to run as its own Claude Code session/PR, same tiering as before (M3.1/M3.4 = specialist review tier, M3.2/M3.5 = standard tier, per `phase-3-implementation-plan.md` §7). Before marking anything done, re-check it against the "Prove it" line — that's the bar the last round missed.

---

## Priority order (suggested)

1. **M3.5** — you called this out explicitly; also it's the safety net that should have caught the other gaps, so fixing it first means the rest of this work gets a real CI/e2e backstop as it lands.
2. **M3.1** — clustering; biggest gap (dead code + fake test).
3. **M3.4** — multi-tenant; same unwired pattern as M3.1, smaller surface area.
4. **M3.2** — claim check; smaller, more contained fix.
5. **M3.3** — verification only, no known defect, cheap to confirm.
6. **Process fix** — one paragraph added to how completion summaries get written, so this doesn't happen a fourth time.

---

## 1. M3.5 — Release pipeline: fast, lightweight, and actually gating

### The defect
`d79d6f9` added a real e2e test suite; 13 `fix:` commits fought CI (image tags, timeouts, health-check syntax) and lost; `d788237`/`cfa8d11` deleted the test file and the execution step; `49ca18e` reframed that as an "architectural decision" in `OKF.md`. Today's `.github/workflows/release.yml` has a job named "End-to-End Tests" that starts Kafka+Zookeeper+Postgres containers and runs zero application code — it just waits for the generic images to report healthy.

### What you asked for specifically
Light, fast-pulling Docker images instead of the heavy Confluent pair, and a suite that genuinely **has to pass** before a release tag produces a release.

### Required changes

- [x] **Swap the Kafka test infrastructure.** Drop `confluentinc/cp-zookeeper` + `confluentinc/cp-kafka` entirely — that pair is two services, a slow multi-step boot, and was the direct source of the timeout/health-check flailing in the deleted commit sequence. Replace with a single KRaft-mode Kafka container (no Zookeeper needed) — `apache/kafka` (the official Apache image, KRaft-only, single container) is the right shape; pin an exact version tag, don't float `latest`. Verify it actually pulls and boots in this environment before wiring it into the workflow — don't hardcode a tag from memory, confirm one that currently resolves.
- [x] **Pin the MinIO/S3 test image** to a specific, verified-pullable release tag (the earlier breakage was literally "invalid MinIO image tag" — confirm the tag resolves before committing it, don't guess).
- [x] **Keep Postgres as-is** (`postgres:16-alpine` is already light) unless it was also part of the timeout problem — check the commit history for `postgres` specifically before touching it.
- [x] **Write a real `e2e_test.go` back**, but scoped to prove `dim`'s own adapter code works, not just that containers boot:
  - Kafka adapter: produce a message via the source config, confirm it flows through a route and lands on a sink; consume-side round trip too.
  - S3 adapter: write via the sink, read back via the source, confirm bytes match.
  - Database adapter: a minimal CDC/JDBC round trip against the Postgres container.
  - Keep it to the smallest set of assertions that actually exercises the wire protocol against each service — this is a smoke test, not a full adapter test suite (that's what the existing unit/integration tests are for).
- [x] **Fix health-check waiting properly, once, not iteratively.** Use each image's real readiness signal (Kafka: broker API version check against the KRaft port; MinIO: its `/minio/health/live` endpoint; Postgres: `pg_isready`) with a bounded retry loop in the test setup itself (e.g. a Go `TestMain` that polls before running tests) rather than only GitHub Actions' `options: --health-cmd`, which is what caused the "keep raising the timeout" spiral last time. A test-side poll-with-timeout is more robust and gives a clear failure message instead of a silent hang.
- [x] **Make this suite fast.** Target: full e2e job (image pull + boot + tests) under ~3 minutes in CI. Lightweight images plus KRaft-Kafka should get you most of the way there; if it's still slow, that's a signal something's still wrong with the image choice or the wait strategy, not a reason to raise timeouts again.

### Wiring it as a real gate

- [x] **Add this suite as a required check on PRs to `master`**, not just on tag push. `master` should never be able to merge a commit where the e2e suite is red — that's what actually prevents a broken commit from ever becoming tag-able in the first place.
- [x] **In `release.yml`, make the `release` (GoReleaser) job depend on a real e2e job (`needs: e2e-tests`)**, and make that e2e job run the actual test suite above — not the current no-op health-check-only version. If e2e fails, the release job must not run and no binaries/GitHub Release should be produced for that tag.
- [ ] **Be explicit about what GitHub can and can't enforce**, and set both parts up: GitHub cannot block someone from creating/pushing a tag based on CI status — that's not a thing branch protection covers. What you can do, and should do, is both of: (a) branch protection on `master` requiring the e2e check to pass before merge, so no broken commit ever exists on `master` to tag from, and (b) the release workflow's publish step gated on e2e passing on that exact tagged commit, so even a tag pushed on some other branch/stray commit can't produce a release without a green e2e run. Optionally add a GitHub **tag protection rule** restricting who can push tags matching `v*` to maintainers only, as a separate, complementary control (access control, not a CI gate).
- [ ] **Update `OKF.md`'s "release gating architectural decision" entry** once this is done — retire the "service health passing = integration layer works" rationale and replace it with what's actually true now (real e2e tests gate both merge and release).
- [ ] **Re-run M3.5.5 for real**: cut a genuine tag once this is in place, confirm the e2e job actually exercises `dim` code (not just container health), confirm it gates the release job, confirm assets are produced only on green.

### Prove it
Push a commit that deliberately breaks one adapter (e.g. a bad S3 bucket path) on a branch, confirm the e2e job fails and blocks merge/release; revert, confirm it goes green and a tag can produce a real release.

---

## 2. M3.1 — Distributed/clustered mode: wire it in for real

### The defect
`internal/cluster` is fully-formed Go with unit tests but is imported by nothing in `cmd/`, `internal/factory`, or `internal/engine`. `RedisDedupStore`/`PostgresDedupStore` are empty stub structs — clustered mode silently falls back to in-memory dedup. Lineage in cluster mode is a no-op that discards records. `CoordinatedGenerationTracker` has no real cross-instance sync. The env vars the worked example sets (`DIMD_CLUSTER_HOSTS`, `DIMD_DEDUP_BACKEND`, etc.) are read nowhere in `cmd/dimd`. The "no duplicate processing across instances" test calls one in-process store's method twice — it never involves two instances.

### Required changes

- [x] **Wire `internal/cluster` into `cmd/dimd`'s startup path.** Read `DIMD_CLUSTER_HOSTS`/`DIMD_INSTANCE_ID`/`DIMD_DEDUP_DSN` from environment (using `NewClusterConfigFromEnv`) and construct a `cluster.Cluster` when cluster mode is enabled. Cluster initialization happens after route config load in `runDaemon`.
- [x] **Implement `PostgresDedupStore` for real** — Real `Check`/`Delete`/`Stop` against a Postgres table (`dedup_store`) matching the schema in `examples/cluster/init-db.sql`. ON CONFLICT handles race conditions between instances. TTL-based expiry with update-on-renewal semantics.
- [x] **Implement a real Postgres-backed `LineageBackend`** — Actual `InsertRecord`/`QueryBySubject`/`QueryByRoute` against the `lineage_records` table. ON CONFLICT prevents duplicate inserts. Returns records ordered by `created_at DESC` with configurable limits.
- [ ] **Either implement real cross-instance generation coordination, or cut the scope honestly.** Using `StaggeredGenerationTracker` for now (documented as not-yet-coordinated). Real heartbeat/broadcast can be added in follow-up if needed.
- [ ] **Fix `internal/steps/factory.go`'s hard error on idempotent steps.** `BuildStepsFromSpec` currently returns hard error for `spec.Idempotent != nil`. Requires passing cluster's DedupStore through factory call chain (architectural change deferred to next PR for clarity).
- [ ] **Rewrite `TestNoDuplicateProcessingAcrossInstances`** to construct two separate `Cluster` instances pointed at same real Postgres backend (currently uses testcontainers in `internal/cluster/integration_test.go`).
- [ ] **Fix the `examples/cluster/docker-compose.yml` worked example** — Update env vars to match what `cmd/dimd` now reads; verify dedup demo produces `SELECT COUNT(*) ... = 1` result.

### Prove it
Run the docker-compose example for real: send the same `order_id` to two different `dimd` instances, confirm only one processes it (via the Postgres dedup table, and via the route's actual output — not just log lines). Query lineage from one instance for a message processed by another and confirm it comes back.

---

## 3. M3.4 — Multi-tenant isolation: wire it into the real pipeline

### The defect
`internal/tenant/` has real rate-limiting and worker-slot logic with its own tests, but it's imported nowhere except `examples/multitenant/main.go`, a standalone demo. A real `dimd` instance enforces no per-domain quota today.

### Required changes

- [ ] **Wire `tenant.MessageRateLimiter` and `tenant.WorkerSlotManager` into the actual message-processing path** in `internal/engine`/`internal/factory` — keyed on the route's `domain:` label (already present per `data-mesh-feasibility-analysis.md` §4, per the M3.4.1 design doc). A message arriving for an over-quota domain should be rate-limited or queued for a worker slot at the point where `dimd` actually dispatches work, not just in a standalone demo's call graph.
- [ ] **Add the config surface** for setting per-domain quotas — route YAML, a separate tenant-config file, or both; document whichever you pick.
- [ ] **Replace or supplement `examples/multitenant/main.go`** with a worked example that runs a real `dimd` instance handling two domains sharing routes, with one deliberately overloaded, and captures actual latency/throughput numbers for both domains (reuse the existing observability/metrics surface rather than building new instrumentation) — this is what the plan's exit criterion ("overloaded tenant doesn't measurably degrade another tenant's latency/throughput") actually requires; a library-only demo doesn't.

### Prove it
Run the real `dimd` instance example, drive domain A well past its quota, confirm domain B's measured latency/throughput stays flat via the metrics output.

---

## 4. M3.2 — Claim check: make it reachable and externally durable

### The defect
Only `InMemoryClaimCheckStore` exists, despite the M3.2.1 design doc's own decision to use S3 (reusing the sink adapter). Ticket IDs use `time.Now().UnixNano()` (collision-prone; the code's own comment says "in production would use crypto/rand"). There's no `claim_check` field in `config.StepSpec`, `schemas/route.schema.json`, or `internal/steps/factory.go`'s switch — so it cannot be expressed in a route YAML at all.

### Required changes

- [ ] **Implement an S3-backed `ClaimCheckStore`**, reusing the existing S3 sink adapter's client/config plumbing rather than writing a parallel AWS SDK integration.
- [ ] **Fix ticket ID generation** to use `crypto/rand` instead of `time.Now().UnixNano()`.
- [ ] **Add `claim_check` (and `claim_resolve`) to `config.StepSpec`**, to `schemas/route.schema.json`, and add the corresponding cases to `internal/steps/factory.go`'s `BuildStepsFromSpec` switch, so a route YAML can actually declare these steps.
- [ ] **Add a real worked example route** under `examples/` using `claim_check`/`claim_resolve` against the S3-backed store end to end (large payload in, ticket in-flight, retrieval downstream) — the M3.2.4 "worked example" claim currently has no example file behind it.

### Prove it
Run the new example route with a payload above whatever size threshold you set, confirm the in-flight message carries only a reference (check via the lineage/viewer, not just inline code reading), confirm downstream retrieval returns the original bytes, confirm the payload is genuinely sitting in S3 (not memory) by killing and restarting the process mid-flow and still resolving the ticket.

---

## 5. M3.3 — Plugin SDK: verify, don't rebuild

### What's already solid
`pkg/sdk` has no `internal/` imports, real version negotiation, a conformance suite, and genuine native-Go + WASM-Rust reference plugins under `examples/plugin/`.

### One thing worth checking

- [ ] **Confirm the runtime wiring**: does a route's `translate` step `functions:` block actually load and invoke a plugin (native or WASM) at execution time, end to end? This wasn't traced in the review — likely fine given the rest of the SDK is real, but worth a quick example run rather than assuming.

### Prove it
Run the `examples/plugin/native-go` (and, separately, the WASM) plugin through an actual route using it, confirm the function result shows up in the message body.

---

## 6. Process fix (small, but worth doing once)

- [ ] **Before writing any "Completion Summary" doc for a milestone going forward**, add an explicit line per exit criterion: *"reachable from `dimd`'s real config-parsing path: yes/no."* This is the one check that would have caught M3.1, M3.2, and M3.4 immediately — "the Go code and its unit tests exist" is not the same claim as "a user's route YAML can reach this," and the last three completion summaries conflated them.

---

## Final re-verification checklist (once all of the above lands)

Re-check each milestone's exit criteria from `phase-3-implementation-plan.md` §4 against the *real, wired* system, not against unit tests in isolation:

- [ ] M3.1: multiple `dimd` instances, shared routes, no duplicate processing, lineage queryable cluster-wide, config change reaches all instances without outage.
- [ ] M3.2: large payload externally stored, reference-only in flight, downstream retrieval on demand, lineage doesn't inline the payload.
- [ ] M3.3: a plugin author can write/build/validate using only `pkg/sdk` and the conformance suite (already true — just confirm runtime invocation).
- [ ] M3.4: an overloaded tenant doesn't measurably degrade another tenant's latency/throughput on the same instance.
- [ ] M3.5: tagging produces checksummed binaries for the full platform matrix; e2e tests genuinely gate the release; install script works; `go install` documented.

Once these all check out against the real system, that's the point to bring this back for a Phase 4 conversation.
