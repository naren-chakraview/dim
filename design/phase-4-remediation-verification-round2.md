# Phase 4 Remediation — Verification, Round 2

**Reviewed against:** the priority list from the second remediation prompt (the two critical items, plus items 3–5, plus the "everything else" list of 8 remaining tasklist items)
**Repo:** `github.com/naren-chakraview/dim`, HEAD `bf40666` ("Merge PR #87 — docs/phase4-remediation-pass2-updates"), 2026-09-17 — **still no version tag cut**, `v0.10.0` remains latest
**Method:** four parallel source-level reviews, same discipline as every prior round — trust nothing self-reported, trace and where possible execute the actual code. Two reviewers built working Go toolchains from source or extracted byte-verified copies of the real code into standalone harnesses specifically to get real execution results rather than static reading, given this project's history.
**Date:** 2026-09-17

## Verdict up front

**Not ready for the next phase.** This round made real progress on two of the five items it targeted, left two more partially done with a specific and non-trivial residual gap each, and left one — domain-scoped secrets — with a newly *demonstrated, reproducible* bypass that's a more concrete problem than the original "unwired" finding. The eight lower-priority items from the punch list were not touched at all: 0 of 8, and one of them (dead studio bundle files) got measurably worse.

## What's genuinely fixed

- **`test_route`'s zero-fixtures fake-pass is closed for real**, and executed, not just read: `TestRoute` now hard-errors `NO_FIXTURES` on zero loaded fixtures, and the proof fixture's YAML key was corrected so it actually loads (confirmed: 1 fixture loads under the corrected key, 0 under the old one). Residual: the test's own assertions for "a real failing fixture reports `passed: false`" are skipped on an early-return path — the commit's own message documents that its actual run took that path — so the claim is true in the production code but not actually demonstrated by its own test. Worth a five-minute follow-up, not a real defect.
- **The governance-fragment lint now genuinely checks the compiled route**, not a grep. Executed against three constructed cases: a real import passes, every reference commented out now correctly fails, and a route with the import present but the fragment step itself commented out also correctly fails. One narrower bypass remains — a route with no import at all but a hand-rolled inline policy matching the same text pattern still passes — worth a follow-up but far less severe than the original hole.

## What's partially fixed, with a specific gap worth naming

- **Studio add/remove-step:** the fix took the safer of two paths — attempting to add or remove a step now returns an error rather than silently succeeding, which is real progress and matches the "fail loudly" requirement. But two of the originally-flagged silent no-ops are still live and unchanged: **adding or deleting a top-level key in the route file is still silently dropped, with the save endpoint still returning success.** The file-reformatting issue (2-space indent becoming 4, blank lines stripped on every save) is also still present — `SetIndent` was never added. The second, older save handler is unchanged.
- **The capability manifest fix landed in the wrong place, again.** The off-by-two enum bug is genuinely fixed *in the generator function* — verified with an executed before/after comparison. But the manifest actually served to agents (`GetCapabilities`) is still a hand-written literal that never calls the generator; someone manually patched it to add the missing step/adapter types instead of wiring the fix through. The result is now **three independently-hardcoded lists** (the served literal, the orphaned generator, and the JSON schema) that disagree with each other, and the served manifest now advertises `sftp` and `exec` as adapters and `aggregate`/`split` as step types — **none of which exist anywhere in the actual codebase.** The shipped `docs/schemas/capability-manifest.json` artifact was never regenerated and still contains every corrupted enum value from before. The CI manifest check remains unable to catch any of this: I had a reviewer actually add a new adapter package and confirm the CI check still passes.

## What's not fixed, and the one that's more serious than before

- **Domain-scoped secrets: still not wired, and now has a demonstrated live bypass.** The commit for this item is literally titled "document domain-scoped secret resolver limitation and defer to Tier 2" — `file_source.go` is still bare `os.Getenv`. The one real improvement is that `TestSecretAccessControlExplicit` was honestly rewritten to assert denial rather than permission, and the store-level authorization check is real. But nothing in the live resolution path calls it — and a reviewer **reproduced an actual bypass**: `ResolveInRoute` falls back to unscoped `os.Getenv` after the store denies a cross-domain request, so an environment variable named `payments.api-key` silently leaks the value to a route in a different domain, denial notwithstanding. `dimctl secret-migrate` still doesn't exist, and its own dangling reference now surfaces in live CLI output rather than just docs.
- **The pkg/agent version boundary is unchanged** — zero importers, still a decorative duplicate of the internal constant actually served.
- **All 8 of the "everything else" items: 0 fixed.** Hot reload, scaffold `auth:`+real-validate testing, catalog filters, the 3 missing studio step-type schemas, the Tier 1 viewer pairing, dead bundle cleanup, the still-inert `checkUnusedImports` critique check, and the impact-analysis gaps (missing `connection` query case, unindexed step-level contract references, untouched sibling test) are all exactly where the last verification round left them. **Dead bundle files actually increased from 12 to 14** — later commits added more unused build artifacts than the one cleanup commit removed.

## Net assessment

Of the 15 items this round targeted (5 named priorities + 8 lower-priority + the CI/pkg-boundary sub-items under the manifest fix and secrets fix): **2 solidly fixed, 2 partially fixed with a clear remaining gap, and 11 not fixed** — including one that regressed and one that went from "gap" to "reproducible live bypass." That's a lower completion rate than the prior remediation round, and the one new regression (secrets) is a security-relevant finding, not just an incomplete feature.

## Recommendation

This isn't ready to hand off to the next phase. Before that conversation:

1. **Fix the secrets bypass first** — it's the only finding in this round that's actively worse than what it started as. Remove or gate the `os.Getenv` fallback in `ResolveInRoute` so a denied cross-domain reference actually fails rather than silently succeeding through a naming coincidence, then wire `file_source.go` to the real resolver for real.
2. **Close the two remaining silent no-ops in the studio** (top-level key add/delete) — same severity class as the add/remove-step bug from last round, just narrower now.
3. **Finish the manifest wiring** — point `GetCapabilities` at the real generator (or delete the generator and build the logic directly into what's served), remove the phantom `sftp`/`exec`/`aggregate`/`split` entries, regenerate the shipped JSON artifact, and make the CI check registry-aware so it can catch a real new adapter — verified this round that it currently can't.
4. **Start the 8 untouched items** — they were never attempted this round, not partially done.

The two genuinely-closed items (`test_route`'s fake-pass, the governance lint) are good, real progress and shouldn't need to be revisited beyond their minor residuals noted above.
