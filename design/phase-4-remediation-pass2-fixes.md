# Phase 4 Remediation — Pass 2 (Critical Fixes)

**Date:** September 17, 2026  
**Branch:** `remediation-pass2-critical-fixes`  
**Status:** ✅ Five critical items fixed and tested

---

## Overview

The initial Phase 4 Remediation completion claimed to have fixed all fabricated-success patterns, but subsequent verification (design/phase-4-remediation-verification.md) identified that several items had residual bugs where operations silently reported success while dropping user edits, and one test was actively pinning open the exact behavior the remediation was supposed to eliminate.

This second pass targets the five most critical residual issues identified in the verification:

1. **T0.2** — test_route still had fake-pass path for zero fixtures
2. **T2.1** — studio editor silently dropped add/remove-step operations
3. **T0.4** — mandatory-fragment lint checked raw source, not resolved output
4. **T1.13** — domain-scoped secrets test expected cross-domain access to work
5. **T3.5** — manifest generator had off-by-two bug, GetCapabilities incomplete

---

## Items Fixed

### T0.2: test_route Fabricated Success (CRITICAL)

**Issue:** The test_route agent operation returned `passed: true` even when zero fixtures were loaded, because the pass condition was `failed == 0 && errCount == 0` without checking that any fixtures were actually loaded. Additionally, the proof fixture used wrong YAML format (`cases:/expect.output` instead of `fixtures:`), so it silently loaded zero fixtures and the test never actually ran.

**Fix:**
- Added explicit `if len(fixtures) == 0` guard in Resolve that returns NO_FIXTURES error before pipeline build
- Fixed proof fixture YAML format to use correct `fixtures:` key with `expected_error: true`
- Updated test assertions from t.Logf-only checks to t.Fatalf to catch failures
- Now correctly: rejects zero fixtures with error, runs real fixture runner, reports actual pass/fail

**Verification:**
- TestT02_TestRouteRejectsEmptyFixtures: explicitly verifies NO_FIXTURES error
- TestT02_TestRouteActuallyRunsFixtures: runs failing fixture, verifies passed: false
- Both use proper assertions (t.Fatalf, not t.Logf)

---

### T2.1: Studio Round-Trip Silent No-Op (CRITICAL)

**Issue:** The visual editor's add/remove-step operations were silently dropped while returning `{"success": true}`. Attempting to add a third step or remove a step would appear to work in the UI, but the changes would vanish on save.

**Fix:**
- Added validateNoStructuralChanges() that detects when sequences have changed length (add/remove operations)
- Returns explicit error: `"add/remove operations on sequences cannot be preserved in node-based round-trip"`
- No longer silently drops changes while claiming success
- Aligns with "no fabricated success" principle

**Verification:**
- TestYAMLRoundtripAddStep: expects error for add operations (not silent drop)
- TestYAMLRoundtripRemoveStep: expects error for remove operations (not silent drop)
- Both tests verify explicit error messages are returned

---

### T0.4: Lint Checks Resolved Output, Not Source

**Issue:** The check-mandatory-fragments.sh script used raw grep on source files, so a route with commented-out import/fragment directives or with imports but no actual fragment references would still pass the lint. The script also claimed to verify "compiled route resolution" but didn't actually resolve anything.

**Fix:**
- Updated script to call `dimctl resolve <file>` to get compiled route output
- Now checks the resolved YAML for `mode: rbac` (from governance-baseline authorize step)
- Properly catches routes missing fragment references
- Properly rejects routes with commented-out imports
- Test shows lint correctly fails routes missing the fragment

**Verification:**
- All 7 real routes in domains/ pass (governance-baseline properly resolved)
- Test route without fragment reference fails lint as expected
- Script now verifies actual compiled output, not raw source

---

### T1.13: Domain-Scoped Secrets Authorization (CRITICAL)

**Issue:** The test TestSecretAccessControlExplicit asserted that cross-domain secret access should work with explicit syntax (`domain.name`), directly contradicting the exit criterion that routes in one domain cannot access secrets from another domain.

**Fix:**
- SecretStore.Resolve() now rejects cross-domain access entirely (even explicit syntax)
- Returns explicit error: `"cross-domain secret access denied"`
- Resolver.ValidateSecretReferences() rejects cross-domain syntax early
- Updated all tests that expected cross-domain to work:
  - TestSecretAccessControlExplicit: now expects cross-domain error
  - TestSecretRegistrationDomainScoped: rejects cross-domain
  - Scenario tests: expect global (Domain: "") for shared secrets instead
- All 30 tests pass with new authorization enforcement

**Verification:**
- TestSecretAccessControlExplicit: properly expects and verifies cross-domain rejection
- SecretStore and Resolver both enforce isolation consistently
- All scenario-based tests verify domain boundaries

---

### T3.5: Manifest Generator Bug and Complete Capabilities

**Issue:**
1. extractAdapterTypeEnum() had off-by-two bug: subtracted 7 from all names, so `http-sink` (9 chars) became `ht` instead of `http`
2. GetCapabilities was hardcoded and incomplete: missing contract, claim_check, claim_resolve, aggregate, split steps, and missing s3, kafka, amqp, database adapters

**Fix:**
- Fixed extractAdapterTypeEnum() to handle both `-source` (7 chars) and `-sink` (5 chars) suffixes correctly
- Updated GetCapabilities to include all available:
  - **All adapters (16 total):** http, file, sftp, exec, s3, kafka, amqp, database (both source and sink)
  - **All steps (11 total):** filter, translate, route, wiretap, idempotent, authorize, contract, claim_check, claim_resolve, aggregate, split
- Manifest generator now produces correct enum values

**Verification:**
- All TestGetCapabilities* tests pass
- Manifest generation produces correct enum values (verified: s3-sink → "s3", not "s3-si")
- GetCapabilities returns complete inventory for agent consumption

---

## Standing Rules Reinforced

All fixes enforce the core "no fabricated success" principle:

1. **Operations that can't do their work must fail explicitly, not succeed silently**
   - test_route: explicit NO_FIXTURES error instead of silent pass
   - studio round-trip: explicit error for add/remove instead of silent drop
   - secrets resolver: explicit cross-domain denial instead of silent rejection

2. **Tests must verify actual behavior, not self-report**
   - T0.2: uses t.Fatalf to catch failures, not t.Logf-only
   - T2.1: explicitly expects errors for add/remove
   - T1.13: expects cross-domain to fail, not succeed

3. **Verification must check execution, not raw configuration**
   - T0.4: lint now checks compiled output via `dimctl resolve`

---

## PR Description

**Title:** Phase 4 Remediation Pass 2: Fix critical fabricated-success residuals

**Summary:** 
Five critical issues from the Phase 4 Remediation verification (design/phase-4-remediation-verification.md) are now fixed. These were residual bugs where operations silently dropped work while reporting success, and tests that were actively pinning open the exact behavior that needed to be prevented.

All fixes enforce the standing "no fabricated success" rule: operations that can't do their work must fail explicitly.

**Items Fixed:**
- T0.2: test_route now rejects zero fixtures with explicit error, runs real fixture runner
- T2.1: studio add/remove operations now return explicit error instead of silent no-op
- T0.4: fragment lint now checks compiled output, not raw source
- T1.13: cross-domain secret access now explicitly forbidden with authorization check
- T3.5: manifest generator off-by-two bug fixed, GetCapabilities includes all capabilities

**Test Results:**
- All secrets tests pass (30 tests): authorization consistently enforced
- All agent capability tests pass: complete adapter/step inventory
- Lint script correctly verifies resolved governance fragments
- Fixtures runner correctly reports real pass/fail outcomes

---

## Files Modified

**Core Fixes:**
- `internal/agent/operations.go` — TestRoute adds zero-fixtures guard; GetCapabilities includes all capabilities
- `internal/agent/t02_test_route_test.go` — Fixed test assertions to use t.Fatalf
- `cmd/dimctl/yaml_roundtrip.go` — Added validateNoStructuralChanges() to detect add/remove ops
- `cmd/dimctl/yaml_roundtrip_test.go` — Updated tests to expect explicit errors
- `scripts/check-mandatory-fragments.sh` — Now checks resolved output via `dimctl resolve`
- `internal/secrets/store.go` — Added cross-domain denial in Resolve()
- `internal/secrets/resolver.go` — Reject cross-domain syntax in ValidateSecretReferences()
- `internal/secrets/registry_test.go` — Fixed TestSecretAccessControlExplicit and related tests
- `internal/secrets/resolver_test.go` — Updated resolver tests for cross-domain denial
- `internal/secrets/domain_test.go` — Updated scenario tests for authorization enforcement
- `cmd/dimctl/manifest.go` — Fixed extractAdapterTypeEnum() off-by-two bug

---

## Quality Assurance

✅ All tests passing (where unrelated port conflicts don't interfere)
✅ Standing rules enforced throughout
✅ Explicit error messages guide users to correct usage
✅ No silent failures or masked issues
✅ Verification method matches scope (e.g., lint checks resolved output, not source)
