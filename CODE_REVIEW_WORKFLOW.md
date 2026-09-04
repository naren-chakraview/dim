# Code Review Workflow (R12)

This document re-establishes the PR-based, risk-tiered human review workflow for Phase 1 and beyond. Based on phase-0-implementation-plan.md's §6, adapted for Phase 1 scope.

## Policy

**Every PR merges through human review, not direct commit.** Review tier depends on risk level.

## Review Tiers

### Tier 1: Standard Review

**Scope:** Most feature work, bug fixes, documentation, tests, refactoring that poses no cross-module risk.

**Required Reviewers:** 1 (self-select by subsystem familiarity)

**Review Time:** 24-48 hours for merge eligibility

**Checklist:**
- [ ] Code compiles (`go build ./...`)
- [ ] Tests pass (`go test -race ./...`)
- [ ] No new race conditions detected
- [ ] Follows project style and patterns
- [ ] Comments explain the why, not the what
- [ ] Backward compatibility maintained (if applicable)
- [ ] No security vulnerabilities introduced

**Approval:** 1 review approval (LGTM) → can merge

**Examples:**
- New step type (filter, translate, route)
- New adapter (source or sink)
- Bug fixes in isolated packages
- Test improvements
- Documentation updates
- Internal refactoring

### Tier 2: Specialist Review

**Scope:** High-risk subsystems where changes can cascade or break invariants across modules.

**Required Reviewers:** 2 (both subsystem specialists; cannot both be the author's team)

**Review Time:** 48-72 hours for merge eligibility

**Checklist:** Tier 1 + the following:
- [ ] Cross-module impacts identified and tested
- [ ] Concurrency invariants preserved (if applicable)
- [ ] Error handling complete and tested
- [ ] Lineage/audit trail not corrupted
- [ ] Authorization checks not bypassed
- [ ] No deadlocks introduced
- [ ] Performance impact analyzed (if applicable)

**Approval:** 2 specialist review approvals → can merge

**Specialist domains (from phase-0-implementation-plan.md §6):**
- **DAG generation & hot reload** — Changes to `internal/route/`, `internal/engine/generation.go`, route compilation logic
- **Authorization** — Changes to `internal/steps/authorize.go`, `internal/authz/`, principal propagation, role/attribute evaluation
- **Lineage & purge** — Changes to `internal/lineage/`, retention policies, purge evidence log, automatic reaper

**Examples:**
- Hot reload logic changes
- Authorization scope expansion
- Lineage retention policy changes
- Purge evidence log modifications
- Core executor refactoring
- Message schema changes

### Tier 3: Governance Review

**Scope:** Policy, architecture, process changes that affect the whole project.

**Required Reviewers:** Project lead + affected subsystem specialists

**Examples:**
- Architectural decisions (e.g., "move from file-based to gRPC services")
- Process changes (this document itself)
- Go version/dependency bumps
- Schema/migration changes affecting all routes

## PR Template

Every PR should include:

```markdown
## Summary
1-3 sentences: what the PR does and why

## Changes
- Bulleted list of what changed
- Subsystems affected (list them explicitly)

## Test Plan
- How to test this change
- What should pass/fail
- Any new tests added

## Risk Assessment
- Tier 1 / Tier 2 / Tier 3
- Rationale (e.g., "affects authorization only, no DAG changes")
- Cross-module impacts (if any)

## Checklist
- [ ] Compiles
- [ ] Tests pass (-race)
- [ ] No race conditions
- [ ] Backward compatible
- [ ] [If Tier 2] Cross-module impacts tested
```

## Review Approval Process

### For the Author

1. **Open PR** with full context (see PR Template above)
2. **Specify review tier** in description (Tier 1 vs Tier 2)
3. **Request reviewers** of appropriate expertise
4. **Wait for approvals** (see "Review Time" above)
5. **Merge after approvals** received (squash-commit unless history is important)
6. **Delete branch** after merge
7. **Update status** in any tracking (GitHub Projects, etc.)

### For Reviewers

1. **Identify your subsystem expertise** (see Specialist domains above)
2. **Self-select PRs** in your area; request others review outside your area
3. **Run the code** if possible; don't rely on inspection alone
4. **Check the test plan** — if it's incomplete, ask for more tests
5. **Approve with LGTM** or request changes
6. **Re-review after changes** — don't approve until satisfied

### Blocking Issues

Do NOT merge if:
- [ ] Tests do not pass
- [ ] Race detector finds issues
- [ ] Compiles but doesn't run (`go build` passes but code panics)
- [ ] Security vulnerability introduced
- [ ] Backward compatibility broken without migration path
- [ ] For Tier 2: cross-module impacts not tested

## Merge Criteria

A PR is ready to merge when:

1. **All tests pass** locally and in CI
2. **Appropriate tier reviewers have approved** (1 for Tier 1, 2 for Tier 2)
3. **No blocking issues** remain open
4. **At least 24 hours have passed** since the PR opened (allows for async feedback)

## CI Requirements

All PRs must pass:

- [ ] `go build ./...` — compiles cleanly
- [ ] `go vet ./...` — no obvious mistakes
- [ ] `golangci-lint run` — style and best practices
- [ ] `go test ./...` — all unit tests pass
- [ ] `go test -race ./...` — no race conditions on integration tests
- [ ] `govulncheck ./...` — no known vulnerabilities
- [ ] Cross-compile targets — builds for linux/amd64 and darwin/arm64 succeed

## Examples

### Example: Tier 1 (Standard) PR

Title: "Add FileSource adapter with local directory polling"

Risk: Low — new adapter, isolated from core engine

Reviewers: 1 (adapter subsystem expert)

Changed files:
- `internal/adapters/file/file_source.go` — new code
- `internal/adapters/file/file_source_test.go` — tests

### Example: Tier 2 (Specialist) PR

Title: "Implement hot-reload generation state machine with graceful draining"

Risk: High — affects core executor, involves concurrency, generation lifecycle

Reviewers: 2 (must include: executor expert + hot-reload expert)

Changed files:
- `internal/engine/generation.go` — state machine
- `internal/engine/executor.go` — generation integration
- `internal/factory/pipeline.go` — generation creation

Cross-module impacts:
- Affects lineage tracking (generation version)
- Affects metrics collection (per-generation metrics)
- Affects authorization (generation principal propagation)

## Escalation

If a PR is blocked on review:

1. **Comment with @review-request** and explain blocker
2. **Async escalate** in team Slack/Discord if review is delayed >72 hours
3. **Document blocker** so it doesn't recur

## Metrics

Track to ensure process is working:

- Average review time (goal: Tier 1 <48h, Tier 2 <72h)
- Number of PRs per week
- Blocking issues discovered in review (should be >0; indicates thorough review)
- Post-merge bugs in reviewed code (should be low)

## Phase 1 Status

**Starting Point:** Phase 0 landed as one squashed commit (606e63c) with no PR history.

**Phase 1 Target:** Every PR reviewed before merge, per this workflow.

**Phase 1 Track A Status:**
- [x] R1-R10: All merged via reviewed PRs (#1-#9)
- [x] R11: Specialist review (ordering feature governance)
- [x] R12: This workflow document
- Remaining: R6-R7 specialist reviews, R14 to follow

**Phase 1 Track B:** All new scope items (Kafka, AMQP, PBAC, OBO, OpenLineage, schema-registry) must follow this workflow.

## References

- Original: `design/phase-0-implementation-plan.md` (§6)
- Review tiers: [](#review-tiers)
- Specialist domains: [](#tier-2-specialist-review)
- CI strategy: `.github/workflows/ci.yml`
