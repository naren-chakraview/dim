# Code Reviewers & Subsystem Expertise

This document maps subsystems to expertise owners for review assignment.

## Subsystem Experts (Phase 1)

| Subsystem | Packages | Tier 2 Specialist | Knowledge Areas |
|-----------|----------|-------------------|-----------------|
| **Core Engine** | `internal/engine/` | — | executor, channels, message envelope, worker pools, hot reload generation state machine |
| **Configuration & Routes** | `internal/config/`, `internal/route/` | — | YAML parsing, schema validation, route versioning, contract versioning, imports/fragments |
| **Steps & Expression** | `internal/steps/`, `internal/expr/` | — | filter, translate, route, idempotent, wiretap steps; JSONata evaluation; functions registry |
| **Authorization (RBAC/ABAC)** | `internal/steps/authorize.go`, `internal/authz/` | ⚠️ **Specialist** | principal extraction, role-based access control, attribute-based policies, on_deny routing, authorization evidence |
| **Lineage & Audit** | `internal/lineage/` | ⚠️ **Specialist** | SQLite store, retention policies, purge evidence log, automatic reaper, subject indexing, export |
| **Adapters** | `internal/adapters/file/`, `internal/adapters/http/` | — | source/sink interfaces, polling logic, connection management, protocol handling |
| **Factory & Pipeline** | `internal/factory/` | — | pipeline builder, executor/router creation, multi-route composition, generation management |
| **Observability** | `internal/observability/` | — | metrics collection, tracing/spans, Prometheus text format, live viewer, OTEL integration |
| **Testing** | `internal/testing/` | — | fixture loading, fixture runner, result summarization |
| **Hot Reload** | `internal/engine/generation.go` | ⚠️ **Specialist** | generation state machine, draining with timeout, version tagging, active/draining/expired states |

## Review Assignment Guide

### Quick Decision Tree

1. **Does the PR touch `internal/steps/authorize.go` or `internal/authz/`?**
   → **Tier 2 specialist review required** (authorization expert)

2. **Does the PR change lineage, retention, purge, or reaper logic?**
   → **Tier 2 specialist review required** (lineage expert)

3. **Does the PR change generation state machine, draining, or hot reload?**
   → **Tier 2 specialist review required** (hot reload expert)

4. **Does the PR touch multiple subsystems above?**
   → **Tier 2 specialist review required** (assign 2 experts from different domains)

5. **Otherwise:**
   → **Tier 1 standard review** (1 reviewer from relevant subsystem)

### Examples

| PR | Files Changed | Tier | Suggested Reviewer |
|-------|-------------|------|-------------------|
| Add HTTP source JWT validation | `internal/adapters/http/` | Tier 1 | Adapter subsystem expert |
| Implement `scoreRisk` WASM plugin | `internal/expr/wasm_runtime.go` | Tier 1 | Expression subsystem expert |
| Fix retain policy purge reaper bug | `internal/lineage/reaper.go` | Tier 2 | Lineage specialist |
| Add RBAC mode to authorize step | `internal/steps/authorize.go` | Tier 2 | Authorization specialist |
| Refactor message envelope | `internal/engine/message.go` | Tier 1 | Engine subsystem expert |
| Implement generation takeover | `internal/engine/generation.go` + `internal/factory/` | Tier 2 | 2 specialists: hot reload + factory |

## Self-Selection Process

1. **Look up your subsystems** above
2. **Watch for PRs** in those areas
3. **Self-select as reviewer** (comment "I'll review this" or use GitHub's review request feature)
4. **If no one volunteers** after 24 hours, escalate to project lead

## Recusal

Do NOT review a PR if:
- You authored it
- You're on the same team as the author (for Tier 2 reviews)
- You have a conflict of interest

## Specialist Certification (Phase 1+)

A reviewer becomes a certified specialist for a subsystem by:

1. **Conducting 3+ Tier 2 reviews** in that subsystem
2. **Finding and fixing 1+ bugs** via review (evidence of scrutiny)
3. **Documenting subsystem invariants** (e.g., "generation state machine never transitions from Expired to Active")

Documented specialists earn the right to **approve Tier 2 reviews alone** for their subsystem (vs requiring 2 reviewers).

## Onboarding Reviewers (Phase 1)

**For new team members:**

1. Start with **Tier 1 reviews** in subsystems you're exploring
2. **Pair review** with an expert for your first Tier 2 review
3. **Read the subsystem code** end-to-end before Tier 2 reviewing
4. **Ask questions** in review comments (shows active scrutiny)
5. **After 3 reviews**, you're ready to self-select independently

## References

- Review workflow: `CODE_REVIEW_WORKFLOW.md`
- Phase 0 review strategy: `design/phase-0-implementation-plan.md` (§6)
- CI checks: `.github/workflows/ci.yml`
