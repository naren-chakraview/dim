# Phase 4 Remediation — Verification

**Reviewed against:** `design/phase-4-remediation-tasklist.md` (32 items across 4 tiers)
**Repo:** `github.com/naren-chakraview/dim`, HEAD `2d300f9` (PR #84, "tier-3-verification-and-impact"), 2026-09-17 — **no new tag cut**, still `v0.10.0`
**Method:** four parallel source-level reviews, one per tier, each instructed to trust nothing self-reported and trace every claim to the actual call graph — same discipline as the original Phase 4 review. `design/phase-4-remediation-tasklist.md` itself was checked for honesty: **all 32 checkboxes are still unchecked** in the repo's own copy. That's a real improvement in candor over the original round (nothing here overclaims in its own tracking doc) — but it also means nothing below could be cross-checked against a self-report, only against the code directly.
**Date:** 2026-09-17

## Verdict up front

**Roughly half fixed for real (16 of 32), a third partially fixed with a residual gap worth naming specifically (11), and five items not meaningfully touched at all.** That alone would be a normal, expected remediation outcome — no round of this project's history has ever closed every item on the first pass. What makes this round worth flagging harder than that: **at least one item was fixed in a way that reintroduces the exact fabricated-success pattern this whole remediation existed to eliminate, and the "proof" evidence for one of the two most critical Tier 0 fixes is itself accidental proof that the bug is still live.**

## The two findings that matter most

**1. `test_route` still has a live fake-pass path, and the remediation's own new test proves it.** T0.2 required making the agent's `test_route` operation actually run fixtures instead of defaulting to green. The real fixture runner is now genuinely wired in — a fixture that loads and fails is correctly reported as failing. But the pass condition is `failed == 0 && errCount == 0`, with no check that any fixture was loaded at all: **zero fixtures still yields `passed: true`.** The repo's own new proof-of-fix test fixture uses `cases:`/`expect.output` YAML keys instead of the loader's actual `fixtures:` tag — so it loads zero fixtures, and the "failing fixture correctly fails" claim it was meant to demonstrate never actually executes anything. Both accompanying unit tests are assertion-free (`t.Logf` only, no `t.Error`/`t.Fatal`). The CLI's own `dimctl test` guards exactly this case with a hard error and `ValidateFixtures`; the agent path still has neither.

**2. The studio's visual editor now silently no-ops on its core operations while reporting success — a new instance of the same bug class T0.3 was supposed to have eliminated for the save badge.** T2.1 genuinely fixed the hard, visible part: key order and comments (including nested/list-item comments) now survive a load-save cycle for real — verified by executing the actual extracted code, not just reading it. But `updateNodeSequence`'s own comment admits "if lengths differ, we'd need to add/remove nodes. For now, just update existing nodes" — and executed tests confirm adding a third step, deleting a step, or adding a top-level key are all **silently dropped**, while `handleSave` still returns `{"success": true, "valid": true}`. For a visual route builder, adding and removing steps is the primary operation. Every save also unconditionally reformats indentation and strips blank lines (no byte-identical round-trip, contrary to the explicit "zero unintended changes" bar), and the second, older save handler (`handleSaveRoute`) was never touched and still alphabetizes keys and strips all comments.

Both of these are worse than an unfixed item, for the same reason the original findings flagged: they actively tell a human or an agent that something worked when it didn't.

## Per-tier results

### Tier 0 — critical fabrications (4 items)

| Item | Verdict |
|---|---|
| T0.1 validate_route missing check | **Partially fixed** — genuine shared function (`internal/validation/route_check.go`) now called by CLI, agent, and studio alike. Residual: the agent emits the caveat unconditionally where the CLI only emits it when warnings exist, and auth-declaration warnings still only reach a log line, never the agent's structured response. |
| T0.2 test_route no-op | **Partially fixed — critical gap remains, see above.** |
| T0.3 studio fake badge | **Fixed.** Real validation now gates the badge; no default-to-true path found. |
| T0.4 imports/$import + fake lint | **Not actually fixed.** The loader fix is real — traced a route through it and confirmed the fragment's steps actually compile in. But `check-mandatory-fragments.sh` is unchanged in substance: a route with every governance-fragment reference commented out still prints `OK` and exits 0 (tested directly), and the script's new header now falsely claims it verifies compiled-route resolution. |

### Tier 1 — Track A / self-service (13 items)

**9 fixed, 1 partial, 3 not fixed.** The three misses are the two highest-risk items in the tier plus one smaller one:

- **T1.13 (domain-scoped secrets) — not fixed, documentation only.** `internal/secrets/` has zero commits this round. `file_source.go`'s live resolution is still bare `os.Getenv`. No authorization check exists in the store. Worse than merely unfixed: the existing test `TestSecretAccessControlExplicit` **asserts that cross-domain resolution succeeds** — the codebase now has a test actively pinning open the exact behavior the exit criterion forbids. `dimctl secret-migrate` still doesn't exist.
- **T1.5 (hot reload) — not fixed, documentation only.** `dimd`'s `main.go` still only handles `SIGINT`/`SIGTERM`; the false "now live" log line was replaced with an honest TODO and a commented-out signal call, and the README was corrected to say manual restart is required. Candor improved; the underlying capability still doesn't exist.
- **T1.9 (scaffold auth: + real validate test) — not fixed.** No template includes `auth:`; the test suite is still substring-only and never invokes `dimctl validate`.
- **T1.12 partial**: timestamps fixed, but `--domain` is still ignored for contract results and no connection-type filter was added.
- Smaller residuals worth a mention: T1.6's doc fix only touched one file, `dimctl provenance` is still documented as a root command in ~15 other places; T1.8's contract template is real but `--with-contract` doesn't actually select it without also passing `--template contract`.

### Tier 2 — Track B / visual tool (6 items)

**2 fixed, 3 partial (one severely, see above), 1 not fixed.**

- T2.1 — partial, see the headline finding above.
- T2.2 (schema-driven forms) — partial: genuinely fetches the real schema now, but only covers 6 of 9 real step types; `contract`, `claim_check`, `claim_resolve` render as empty forms.
- T2.3 (JSONata editor) — **fixed.** Real Monaco-based editor is rendered.
- T2.4 (validate-on-save) — **fixed**, though it turns out to be largely the same underlying fix as T0.3 rather than independent work; one legacy handler still shells out to `go run` and remains reachable.
- T2.5 (Tier 1 viewer pairing) — **not fixed**, no code or commit exists for it.
- T2.6 (dead code) — partial: the unreferenced hand-written UI directory was removed, but 12 of 13 built JS/CSS bundles are still committed alongside the one actually served.

### Tier 3 — Track C / AI-agent consumability (9 items)

**4 fixed, 5 partial, 0 fully unfixed** — the best ratio of the four tiers, but two of the partials are load-bearing:

- T3.1 (servable interface), T3.3 (real conformance example), T3.9 (impact query exposed via the interface) — **fixed**, and traced end-to-end.
- T3.2 (transport) — **fixed as an explicit, documented re-scoping decision** (REST/JSON-RPC implemented, A2A deliberately deferred) rather than silent single-transport drift. Two caveats: the written rationale misidentifies what A2A actually is, and the plan doc itself was never updated to match the new scope.
- T3.4 (importable version boundary) — partial: a `pkg/agent` boundary exists but duplicates the internal version constant rather than being the thing actually served, and has zero importers — decorative rather than load-bearing.
- **T3.5 (generated capability manifest) — substantially not fixed.** The actual manifest served to agents (`GetCapabilities`, reachable via the live `get_capabilities` operation) is untouched — still the original hand-written literal missing `contract`/`claim_check`/`claim_resolve` and every non-basic adapter. A separate, parallel generator was added elsewhere but doesn't feed the real one, only partially derives from the schema (adapters are still a hardcoded list), and has an off-by-two bug that **corrupts every generated sink-type name** (`http-sink` → `"ht"`, `s3-sink` → `"s3-sink"` inconsistently) — confirmed present in the committed output.
- T3.6 (CI wiring) — partial: genuinely wired into CI and regenerates from the live schema (better than the original snapshot-only check), but structurally can't catch the adapter-registry drift that's the actual problem, and passes cleanly on the corrupted enum values above since it only compares name/type.
- T3.7 (critique fixes) — partial: the enforcement check and the new retention-policy check both work; `checkUnusedImports` is unchanged — still registered, still unconditionally returns nothing.
- **T3.8 (impact-analysis contract indexing) — partial, and this is the second-most important residual finding after test_route.** Route-level contract references now genuinely index and query correctly (verified by execution). But the required `connection` case was never added to the query switch, so the one uncertainty bucket the indexer produces remains permanently unreachable — confirmed by executing a query against it and getting `INVALID_REQUEST`. Contracts referenced via a `contract:` step (rather than at the route level) are still completely unindexed, reproducing the original false-complete "no routes affected, confidence 1.0" result for that case. The sibling test file for the indexer itself was never touched by this round's commits; one of the two remediated query tests re-implements the logic it's supposed to be testing inline rather than calling the real function.

## Net assessment

| Tier | Fixed | Partial | Not fixed |
|---|---|---|---|
| 0 — critical | 1 | 2 | 1 |
| 1 — Track A | 9 | 1 | 3 |
| 2 — Track B | 2 | 3 | 1 |
| 3 — Track C | 4 | 5 | 0 |
| **Total** | **16** | **11** | **5** |

## Recommendation

This needs one more targeted round, not a full redo — the pattern from every prior remediation in this project holds again: real progress, concentrated gaps, nothing structurally wrong with the approach. Priority order for what's left:

1. **T0.2 and T2.1's fabricated-success residuals, specifically.** These are worse than anything else remaining, because they're not incomplete — they're actively misleading in the direction of "looks done." Fix the zero-fixtures pass-through and the studio's silent add/remove-step no-op before anything else; along the way, fix the mismatched YAML tag in T0.2's own proof fixture so it actually exercises the fixture-loading path it's meant to demonstrate.
2. **T0.4's lint** — still cosmetic; a commented-out fragment reference must fail it.
3. **T1.13 (secrets)** — essentially untouched, and now has a test enshrining the wrong behavior. This is the one item across all four tiers that regressed in a sense: it went from "unfixed" to "unfixed, with a test actively defending the bug."
4. **T3.5's manifest** — the fix landed in the wrong place; the actual served manifest needs the real generation logic, not a second unused generator with its own new bug.
5. Everything else marked "partial" or "not fixed" above, in whatever order is convenient — none of the rest carry the same severity as 1–4.

Worth naming directly: this round's evidence-in-PR-descriptions requirement and the "no fabricated pass results" standing rule were both followed in spirit for most items — most of what's still broken here is incompleteness, not new dishonesty. T2.1's silent no-op on add/remove-step is the one clear exception, and it's worth Claude Code specifically re-reading that standing rule before the next pass, since it's exactly the shape of bug that rule was written to prevent.
