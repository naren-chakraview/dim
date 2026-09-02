# Phase 0 High Priority Tasks — Implementation Plan

**Date:** September 1, 2026  
**Scope:** Missing critical features from design's Phase 0 (M0.2.6, M0.2.7, M0.2.10, M0.4)  
**Status:** Planning

---

## Overview

The design document's Phase 0 includes 50 subtasks across 6 milestones. Current implementation has ~19-20 completed, leaving ~30-31 missing. Three features are high priority:

1. **M0.2.10: Hot Reload / DAG Generation** (highest-risk item per design)
2. **M0.4: Lineage System** (9 subtasks: store, retention, reaper, purge, export, provenance)
3. **M0.2.6: Config Composition** (imports/fragments)

---

## Dependency Analysis

```
M0.2.6: Imports/Fragments ──┐
                             ├──→ M0.2.7: Route_version ──┐
                             │                              ├──→ M0.2.10: Hot Reload
                             └─────────────────────────────┘
                             
M0.2.7: Route_version ─────→ M0.4.1: Lineage Store ──→ M0.4.2-9: Full Lineage System
```

**Implementation Order:**

### Phase A: Foundations (4-5 hours)
- **M0.2.6:** Config composition (imports/fragments) — ~2 hours
- **M0.2.7:** Route_version content hash — ~1-2 hours

### Phase B: Parallel Tracks (8-10 hours)
- **M0.2.10:** Hot reload / DAG generation — ~4-5 hours (highest risk)
- **M0.4:** Lineage system (9 subtasks) — ~4-5 hours

### Phase C: Integration & Testing (2-3 hours)

**Total:** ~14-18 hours

---

## Phase A: Foundations

### M0.2.6: Config Composition (Imports/Fragments)

**Purpose:** Enable reusable config fragments and composition

**What to Build:**

1. **Fragment Resolution:**
   - Load `.yaml` files with `$import: path/to/fragment.yaml` directives
   - Recursive resolution (fragments can import other fragments)
   - Cycle detection

2. **Config Merging:**
   - Merge fragment sources, sinks, steps
   - Deep merge maps, concatenate arrays
   - Override semantics (child overrides parent)

3. **Updated Config Loader:**
   - `internal/config/fragments.go` — Fragment loading & resolution
   - `internal/config/merge.go` — Config merging logic
   - Update `internal/config/loader.go` to call fragment resolver

**Exit Criteria:**
- Unit tests: fragment loading, cycle detection, merge semantics
- Integration test: a route composed from 2+ fragments resolves to identical config as if written inline
- Example: `examples/fragments/base.yaml` + `examples/fragments/auth.yaml` → merged route

**Subtasks:**
- M0.2.6.1: Fragment loading (resolve `$import` directives)
- M0.2.6.2: Fragment merging (combine sources, sinks, steps)
- M0.2.6.3: Cycle detection (prevent infinite imports)
- M0.2.6.4: Integration & examples

**Estimated Effort:** 2 hours

---

### M0.2.7: Route_version Content Hash

**Purpose:** Stable, deterministic versioning of routes for lineage tracking

**What to Build:**

1. **Hash Computation:**
   - Serialize resolved route to canonical JSON (sorted keys)
   - SHA256 hash of serialized route
   - Deterministic: same route → same hash always

2. **API:**
   - `internal/route/version.go` with `ComputeRouteVersion(resolvedConfig) → string`
   - Exposed on resolved RouteSpec

3. **Stamping:**
   - Attach route_version to all processed messages
   - Include in lineage records (M0.4)

**Exit Criteria:**
- Unit test: identical resolved configs produce identical hashes
- Unit test: any change to resolved config changes hash
- Unit test: hash is deterministic across multiple runs
- Integration test: route_version stamped on message metadata

**Subtasks:**
- M0.2.7.1: Canonical JSON serialization
- M0.2.7.2: SHA256 hashing
- M0.2.7.3: API & stamping

**Estimated Effort:** 1-2 hours

---

## Phase B: Parallel Implementation

### M0.2.10: Hot Reload / DAG Generation State Machine

**Purpose:** Zero-downtime route updates with graceful draining

**Design Reference:** `design/phase-0-implementation-plan.md` §14.1, §5 (M0.2.10 detailed spec)

**What to Build:**

1. **Generation Lifecycle:**
   - Current generation (handling traffic)
   - New generation (being deployed)
   - Straggler tracking (old generation finishing in-flight messages)
   - Concurrent-generation cap (safety valve)

2. **State Transitions:**
   - ACTIVE → DRAINING (new generation takes new traffic)
   - DRAINING → CLOSED (old generation finishes in-flight, exits)
   - Max 2 concurrent active generations (cap safety valve)

3. **Implementation:**
   - `internal/engine/generation.go` — Generation state machine
   - Atomic generation swaps
   - In-flight message tracking per generation
   - Drain timeout (30s default)

4. **Testing:**
   - `-race`-enabled integration test: reload mid-traffic
   - Assert: old generation drains, new generation handles new traffic, zero loss/duplication

**Exit Criteria (per design §5):**
- `-race`-clean integration test reloads a route mid-traffic
- In-flight messages on old generation complete
- New messages route through new generation
- No message lost or duplicated

**Subtasks:**
- M0.2.10.1: Generation struct & state machine
- M0.2.10.2: Atomic generation swap
- M0.2.10.3: In-flight tracking & draining
- M0.2.10.4: Safety valve (concurrent-generation cap)
- M0.2.10.5: Integration tests with `-race`

**Estimated Effort:** 4-5 hours

**Risk Notes:** Design calls this "highest-risk item" — requires careful concurrency handling, extensive `-race` testing

---

### M0.4: Complete Lineage System

**Purpose:** Full audit trail of message processing with retention, purge, and export

**Design Reference:** `design/phase-0-implementation-plan.md` §10

**Subtasks:**

#### M0.4.1: Embedded SQLite Lineage Store (1-1.5 hours)
- `internal/lineage/store.go` using `modernc.org/sqlite`
- WAL mode (concurrent reads, serialized writes)
- Single-writer goroutine pattern (serialize concurrent writes)
- Schema: messages (id, route, subject_id, payload, timestamps, route_version, contract_version, principal)

#### M0.4.2: Static Retention Policy Resolution (0.5-1 hour)
- `internal/lineage/retention.go`
- Parse static retention policies from config
- Example: `retention_policy: "30-days"`

#### M0.4.3: Dynamic Retention Policy Resolution (0.5-1 hour)
- `retention_policy_expr: "metadata.tier == 'premium' ? '90-days' : '30-days'"`
- Fail-open to default on unresolved policy
- Per-message dynamic assignment

#### M0.4.4: Subject ID Indexing (0.5-1 hour)
- Extract subject_id via `subject_id_expr` (JSONata)
- Index lineage records by subject for provenance queries
- Example: `subject_id_expr: "principal.id"`

#### M0.4.5: Automatic Reaper (1-1.5 hours)
- `internal/lineage/reaper.go`
- Per-policy configurable cadence (e.g., hourly, daily)
- Background goroutine: delete records past retention window
- Concurrent with normal message flow

#### M0.4.6: Manual Purge with Subject Targeting (1-1.5 hours)
- `internal/lineage/purge.go`
- `midctl lineage purge [--subject <id>] [--before <date>]`
- Delete records matching criteria
- Log purge action

#### M0.4.7: Purge Evidence Log (0.5-1 hour)
- `internal/lineage/purgelog.go`
- Append-only log of all purge actions
- Bounded retention (separate from message retention)
- `expiry_warning_lead` mechanism: warn before evidence expires

#### M0.4.8: Export (CSV/NDJSON) (1-1.5 hours)
- `internal/lineage/export.go`
- `midctl lineage export [--format csv|ndjson] [--since <date>] [--until <date>]`
- Stream to file or stdout

#### M0.4.9: Provenance Query (1 hour)
- `internal/lineage/query.go`
- `midctl provenance [--message-id <id>] [--subject-id <id>]`
- Trace full processing chain for a message or subject

**Total M0.4 Estimated Effort:** 4-5 hours

**Exit Criteria (per design §5):**
- Message processed through route with retention policy is queryable via `midctl provenance`
- Message is reaped automatically once retention window elapses
- `midctl lineage purge --subject <id>` removes records and logs action
- Evidence log survives past purged records' retention
- `midctl lineage export` produces valid CSV/NDJSON

---

## Proposed Execution Strategy

### Option 1: Sequential (14-18 hours)
```
M0.2.6 (2h) → M0.2.7 (1.5h) → M0.2.10 (4.5h) → M0.4 (4.5h) → Integration (1.5h)
```

### Option 2: Parallel After Foundations (10-12 hours wall clock) ⭐ Recommended
```
M0.2.6 (2h)
    ↓
M0.2.7 (1.5h)
    ├→ M0.2.10 (4.5h) ──┐
    └→ M0.4 (4.5h) ────→ Integration (1.5h)
```

- Phase A: M0.2.6 (serial, 2h)
- Transition: M0.2.7 (serial, 1.5h, unblocks M0.2.10 & M0.4)
- Phase B: M0.2.10 + M0.4 in parallel (4.5h each)
- Phase C: Integration & testing (1.5h)
- **Total wall clock:** ~10 hours (vs 14-18 sequential)

---

## Quality Gates

- All tests pass with `-race` flag (especially M0.2.10)
- Integration tests for M0.2.6 (composition), M0.2.10 (hot reload), M0.4 (purge/export)
- Example configs demonstrating each feature
- Documentation with usage examples

---

## Success Criteria

✅ Config composition (imports/fragments) working  
✅ Route versioning (route_version hash) stable  
✅ Hot reload with zero message loss (M0.2.10)  
✅ Full lineage system operational (M0.4: store, retention, purge, export, provenance)  
✅ All tests pass with `-race` flag  
✅ Documentation & examples for each feature  

---

## Next Steps

1. **Confirm approach** (sequential vs parallel)
2. **Launch Phase A agents** (M0.2.6, then M0.2.7)
3. **Launch Phase B agents** (M0.2.10 + M0.4 in parallel)
4. **Integration & release**
