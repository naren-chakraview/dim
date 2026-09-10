# Task 7: Update phase-4-implementation-plan.md to track M4.1 completion

## Files to Modify

- Modify: `design/phase-4-implementation-plan.md`

## Interfaces

- Consumes: Completion status of all M4.1 subtasks (T1-T6)
- Produces: Updated design document with M4.1 marked complete

## Steps to Execute

### Step 1: Locate M4.1 section in phase-4-implementation-plan.md

Find the section titled "### M4.1 — GitOps deployment pipeline" (approximately line 24-34 in the plan).

### Step 2: Add completion header to M4.1 section

At the very top of the M4.1 section, insert this block:

```markdown
**Status:** IMPLEMENTED (2026-09-09)

**Summary:**
- M4.1.1 ✅ CI pipeline config: `dimctl validate` + `dimctl test` + mandatory-fragment lint in `.github/workflows/ci.yml`
- M4.1.2 ✅ Config delivery: `deploy/git-sync.sh` with systemd/cron setup docs
- M4.1.3 ✅ Domain-scoped review: `domains/CODEOWNERS` configured with platform-team and per-domain leads
- M4.1.4 ✅ Worked example: `domains/payments/order-payment.yaml` route, validated and tested
```

### Step 3: Add exit criteria verification section

Below the summary, add:

```markdown

**Exit Criteria Verification:**
- ✅ Route changes in PRs are validated and tested automatically (CI gates M4.1.1)
- ✅ Merging to main reaches running instances without manual restart (git-sync + hot-reload, M4.1.2)
- ✅ Reviewer is domain lead, not central team (CODEOWNERS auto-assignment, M4.1.3)
- ✅ `dimctl provenance` shows new `route_version` post-deploy (verified with worked example, M4.1.4)

**Files Delivered:**
- domains/{README.md, CODEOWNERS, governance/{fragments.yaml, common-steps.yaml}}
- domains/payments/{DOMAIN.yaml, order-payment.yaml, order-payment.route_test.yaml}
- scripts/{validate-routes.sh, test-routes.sh, check-mandatory-fragments.sh}
- deploy/{git-sync.sh, README.md}
- docs/self-service/{GITOPS_WORKFLOW.md, GIT_SYNC_SETUP.md}
- .github/workflows/ci.yml (updated with merge gates 8, 9, 10)
```

### Step 4: Verify existing M4.1 sections are intact

The existing "Scope", "Subtasks", and "Exit criteria" sections of M4.1 (from the original plan) should remain unchanged. Only add the completion header and exit criteria verification sections above them.

### Step 5: Verify no other changes to the document

Do NOT modify any other sections of the phase-4-implementation-plan.md. This task updates only M4.1.

### Step 6: Commit

```bash
git add design/phase-4-implementation-plan.md
git commit -m "docs: mark M4.1 (GitOps pipeline) complete with exit criteria verification"
```

## Success Criteria

- `design/phase-4-implementation-plan.md` updated with M4.1 completion header
- Status line shows: "**Status:** IMPLEMENTED (2026-09-09)"
- Summary lists all four M4.1 subtasks with ✅ checkmarks
- Exit criteria verification section added (showing all 4 criteria met)
- Files delivered section lists all 10+ files created/modified
- Commit present in git log with message matching spec
- No other sections of the document modified

## Notes

- This is purely documentation — no code changes
- The completion header acts as a quick reference for what M4.1 delivered
- Future readers can see exactly what was built and what criteria were verified
- The file modifications list allows someone to quickly find all M4.1 artifacts in the repo
