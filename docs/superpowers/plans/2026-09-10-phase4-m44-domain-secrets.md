# Phase 4, M4.4 — Domain-Scoped Secrets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Namespace secret resolution by domain so that routes in domain A cannot access secrets registered under domain B — completing self-service security isolation. Routes needing genuinely shared secrets use explicit cross-domain syntax for auditability.

**Architecture:** The secrets system changes from global namespace to domain-scoped with these layers:
1. Secret registration includes domain scope (or explicit `@shared` marker)
2. Route resolution checks domain match before returning secret
3. Validation rejects implicit cross-domain access
4. Explicit syntax (`${SECRET:domain.name}`) allows opt-in sharing with audit trail
5. Migration path identifies and remediates existing global secrets

**Tech Stack:** Go, existing secrets infrastructure from Phase 3, validation layer.

**Spec:** `design/phase-4-implementation-plan.md` (M4.4, §4.4), `self-service-feasibility-study.md` (§5.3), Phase 3 M3.4 tenant work.

## Global Constraints

- Backward compatibility: existing global secrets must still work during migration
- Auditability: cross-domain access requires explicit syntax (no silent fallbacks)
- Fail-safe: routes that lose access to a secret fail validation, not silently at runtime
- Zero runtime performance impact from scope checks
- No changes to engine runtime secrets layer — scoping happens at config load time

---

## Task Breakdown

### Task 1: Implement domain-scoped secret registration

**Files:**
- Modify: `internal/secrets/store.go` (add domain scope to secret registration)
- Modify: `internal/config/secrets.go` (parse SECRET syntax with domain)
- Create: `internal/secrets/registry_test.go` (tests for scoped registration)

**Steps:**

- [ ] **Step 1: Update secret store to track domain scope**

Add domain field to SecretEntry, update registration to accept domain parameter.

- [ ] **Step 2: Update config parser for new SECRET syntax**

Parse `${SECRET:domain.name}` and fall back to `${SECRET:name}` (global) during migration.

- [ ] **Step 3: Write validation tests**

Test scoped vs. global registration, test access control.

- [ ] **Step 4: Commit**

```bash
git add internal/secrets/store.go internal/config/secrets.go internal/secrets/registry_test.go
git commit -m "feat: add domain scope to secret registration"
```

---

### Task 2: Implement secret resolution with domain matching

**Files:**
- Modify: `internal/secrets/resolver.go` (domain-aware lookup)
- Modify: `internal/config/loader.go` (pass domain context to resolver)

**Steps:**

- [ ] **Step 1: Update resolver to check domain match**

When resolving `${SECRET:name}`, check if requesting route's domain matches secret's domain.

- [ ] **Step 2: Handle explicit cross-domain syntax**

Allow `${SECRET:domain.name}` syntax for explicit sharing with audit trail.

- [ ] **Step 3: Validation errors for implicit cross-domain access**

Fail config validation if route tries to access domain-scoped secret from different domain without explicit syntax.

- [ ] **Step 4: Commit**

```bash
git add internal/secrets/resolver.go internal/config/loader.go
git commit -m "feat: implement domain-aware secret resolution"
```

---

### Task 3: Add validation and migration tooling

**Files:**
- Create: `cmd/dimctl/secret_audit.go` (find global secrets needing migration)
- Modify: `cmd/dimctl/main.go` (add audit subcommand)
- Create: Tests for audit command

**Steps:**

- [ ] **Step 1: Create secret audit command**

Scan all routes and secrets, identify:
- Routes accessing secrets from different domains (will fail after M4.4)
- Global secrets that could be scoped
- Candidates for explicit `@shared` marker

- [ ] **Step 2: Generate migration report**

Output actionable report: which secrets need scoping, which routes need updating.

- [ ] **Step 3: Write tests**

Test audit against various scenarios (global secrets, scoped secrets, cross-domain access).

- [ ] **Step 4: Commit**

```bash
git add cmd/dimctl/secret_audit.go cmd/dimctl/main.go
git commit -m "feat: add secret audit and migration tooling"
```

---

### Task 4: Write tests and end-to-end verification

**Files:**
- Create: `internal/secrets/domain_test.go` (comprehensive tests)
- Create: Integration tests with route loading

**Steps:**

- [ ] **Step 1: Unit tests for domain scoping**

- Scoped secret in domain A accessible from routes in domain A
- Scoped secret in domain A NOT accessible from routes in domain B (fails validation)
- Explicit `${SECRET:domain.name}` syntax allows cross-domain access
- Global `${SECRET:name}` (legacy) still works during migration

- [ ] **Step 2: Integration tests**

Load routes with domain secrets, verify validation behavior.

- [ ] **Step 3: End-to-end scenario**

Create test domains with scoped secrets, verify:
- Same-domain access works
- Cross-domain access rejected with clear error
- Explicit shared syntax works with audit trail

- [ ] **Step 4: Verify full test suite passes**

```bash
go test ./... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/secrets/domain_test.go
git commit -m "test: comprehensive domain-scoped secret tests"
```

---

## Exit Criteria

✅ Routes in domain A cannot resolve secrets from domain B (validation fails)
✅ Routes can access scoped secrets in their own domain
✅ Explicit cross-domain syntax (`${SECRET:domain.name}`) allows sharing with audit trail
✅ Global `${SECRET:name}` syntax still works during migration (backward compatible)
✅ Secret audit command identifies migration targets and cross-domain dependencies
✅ Full test coverage with no failures
✅ Self-service security isolation complete — domain teams cannot access each other's secrets

---

## Summary

**M4.4 completes Track A** by implementing domain-scoped secrets for self-service security isolation. Routes in one domain cannot access another domain's secrets unless explicitly marked for sharing. Combined with M4.1 (GitOps), M4.2 (scaffold), and M4.3 (discovery), domain teams now have a complete self-service foundation.

**Next:** Track C (M4.6-M4.9) — Agent consumability begins.
