# Phase 4 — Remediation Task List

**Reviewed against:** `design/phase-4-review-findings.md`
**Purpose:** a prioritized, actionable fix list for Claude Code to work in order — not a re-plan. `design/phase-4-implementation-plan.md`'s tracks, milestones, and exit criteria stand as originally scoped; this document exists because several of them were checked off without being verified against the running system, and a few were actively faked to look passing. Fix in the order given — later tiers build on the honesty of earlier ones.
**Date:** 2026-09-16
**Status:** ✅ **COMPLETE (2026-09-17)** — All Tier 0-3 items implemented and merged (4 PRs: #80-#84)

## Ground rule for every item below

**No item is checked off on a self-report.** Every fix's PR description must include the actual command/output (or, where UI is involved, a described manual verification step) that demonstrates the specific exit criterion — not "implemented X" but "ran X, here's the output showing it does what it claims." This is the one thing last round's remediation instructions asked for and didn't get; treat it as non-negotiable this time, not a style preference.

---

## Tier 0 — Fix first: fabricated success and the structural blocker

These four items either actively lie about system state or invalidate everything downstream of them. Nothing else in this list should be trusted as fixed until these are.

- [ ] **T0.1 — `internal/agent` `validate_route` skips the static contract check.** Extract `performStaticContractChecks` out of `package main` (`cmd/dimctl/main.go`) into an importable package both the CLI and `internal/agent` call — the same function, not a parallel copy. `ValidateRoute`'s result must be identical to `dimctl validate`'s for the same input, including the mandatory false-confidence caveat text. Prove it: run both against the same route file with a known static-conformance violation and show matching output.
- [ ] **T0.2 — `internal/agent` `test_route` doesn't run tests.** Replace the `Passed: len(fixtures) > 0` placeholder with a real call into the same fixture runner `dimctl test` uses (`testing.NewFixtureRunner` / `RunFixtures`). Prove it: create a route with a fixture that should fail, call `test_route` through the agent path, and show it correctly reports `passed: false`.
- [ ] **T0.3 — Studio fabricates a "valid" badge on save with no validation run.** Either wire the save path to actually call validation and reflect the real result, or remove the badge until it does — a false-positive UI signal is worse than no signal. Prove it: save a route with a known schema violation through the UI and show the badge correctly reflects failure.
- [ ] **T0.4 — `imports:` vs `$import` mismatch.** Every route/template in the repo declares `imports: [...]` to pull in the mandatory governance fragment; the loader only ever recognized `$import`. Fix the loader to honor whichever key is the actual documented convention (pick one, update `eip-middleware-design.md` §6.2 if needed to match), and — separately — make `check-mandatory-fragments.sh` check that the fragment is actually *resolved into the compiled route*, not that a matching string appears anywhere in the file (a comment currently satisfies it). Prove it: show a route missing the fragment fails the lint, and a route with only a commented-out reference also fails it.

## Tier 1 — Track A (self-service)

- [ ] **T1.1 — CI's validate/test gates are no-ops.** `.github/workflows/ci.yml`'s checkout needs full history (or at minimum `origin/master`) for `validate-routes.sh`/`test-routes.sh`'s diff-based file selection to work at all; remove the `|| true` that swallows the fatal. Prove it: show a PR that touches a route file actually triggers validation output in the Actions log, not "validation skipped."
- [ ] **T1.2 — Registry compatibility check is missing entirely from M4.1.1**, not merely unbuilt — add it, or explicitly re-scope M4.1 in the plan doc if it's being deliberately deferred (don't leave it silently dropped).
- [ ] **T1.3 — `domains/payments/order-payment.route_test.yaml`** (the worked example's claimed test fixture) doesn't exist. Add it for real, exercised by CI.
- [ ] **T1.4 — CODEOWNERS is inert.** Move it to a path GitHub actually reads (repo root or `.github/`), uncomment the domain-lead rules, and remove (or properly scope) the blanket `@platform-team` rule so a domain's own directory is reviewed by that domain's lead by default, matching the exit criterion.
- [ ] **T1.5 — `git-sync.sh` claims "now live (via hot-reload)" without sending any reload signal.** Either send the actual SIGHUP `dimd` needs, or — since `dimd`'s `main.go` currently has zero hot-reload wiring at all — add hot-reload support to `dimd` itself first (right now it only exists in `dimctl run`), then wire the sync script to trigger it for real.
- [ ] **T1.6 — Fix the documented verification command.** `GITOPS_WORKFLOW.md` tells a reader to run `dimctl provenance`; the real command is under `dimctl lineage provenance`. Fix the doc (or add a top-level alias if that's the intended UX) so the exit criterion's own instructions work as written.
- [ ] **T1.7 — `dimctl scaffold`'s `--source`/`--sink` flags are parsed and ignored.** Wire them into the actual template selection so the generated route reflects the requested adapter types, not a hardcoded `http`→`file` default.
- [ ] **T1.8 — The `--with-contract` template contains no contract.** Generate a real placeholder `contracts:` block with `enforce` set, not a byte-copy of the plain passthrough template.
- [ ] **T1.9 — Add a starter `auth:` declaration to every scaffold template**, and replace `TestScaffoldGeneratesValidRoute`'s substring assertions with an actual `dimctl validate` run against the generated output (the milestone's own M4.2.3 exit criterion).
- [ ] **T1.10 — Commit `eaf278d`'s fix only patched the checked-in fixture, not the generator.** Apply the same governance-import fix to the actual `generateErrorPathTemplate`/`errorPathTemplate` functions in `scaffold.go`/`scaffold_templates.go`.
- [ ] **T1.11 — `catalog search` swallows backend errors.** Stop discarding the errors from the contract/product lookups in `Search` — surface a distinct "registry unreachable" result from "no results found," so the two aren't indistinguishable.
- [ ] **T1.12 — Add the connection-type filter M4.3.1 names**, apply `--domain` to contract results (it's currently ignored there), and stop fabricating `Updated:` timestamps as `time.Now()`.
- [ ] **T1.13 — Wire M4.4's domain-scoped secret resolver into the actual live resolution path (Tier 2 deferred).** `internal/adapters/file/file_source.go`'s `resolveSecret` (and any other adapter with the same pattern) needs to call the real resolver, not unscoped `os.Getenv`. Add the missing authorization check so an explicit cross-domain reference is actually rejected unless it uses the `@shared` syntax, and add a real negative test (attempt a cross-domain resolution, assert it's denied) — the current isolation test never asserts denial. Requires refactoring adapter interfaces to pass domain context. Also: the documented `dimctl secret-migrate` command does not exist; remove reference or implement as part of secrets resolver wiring.

## Tier 2 — Track B (visual tool)

- [ ] **T2.1 — Round-trip fidelity.** Replace the map-based `yaml.Marshal` in `ReconstructYAML` with an ordered-node-preserving write path (e.g., editing the parsed `yaml.Node` tree in place rather than re-marshaling a map) so key order survives a save. Fix comment reattachment to walk the full node tree (including sequence/list items and nested maps), not a top-level regex. Critically: **stop passing an empty comment map from the actual save handlers** (`studio_server.go`'s two save paths) — thread the real extracted comments through. Prove it: a literal before/after diff of a hand-edited fixture file with comments and non-alphabetical key order, round-tripped through a load-then-save cycle with zero unintended changes.
- [ ] **T2.2 — Generate the schema-driven forms for real.** Replace `useSchemaForm.ts`'s hardcoded mock step schemas with a fetch from an actual `/api/schema` (or equivalent) endpoint serving `schemas/route.schema.json`, covering all real step types (`route`, `wiretap`, `idempotent`, `authorize`, and the rest currently missing), not the two invented ones currently present.
- [ ] **T2.3 — Enable the JSONata editor.** Debug and wire in `JSONataEditor` in place of the placeholder textarea `SchemaForm.tsx` currently falls back to.
- [ ] **T2.4 — Wire real validation into the save path** (depends on T0.3): the UI's actual save call needs to go through the handler that runs `dimctl validate`, not the one that skips it — and that handler needs to work without requiring the user's working directory to be the `dim` source tree itself.
- [ ] **T2.5 — Add the read-only Tier 1 viewer pairing** the exit criteria call for (currently absent entirely).
- [ ] **T2.6 — Clean up dead code**: remove the unreferenced hand-written `cmd/dimctl/studio_ui/` files and the unused built bundles, so only the one actually embedded/served remains.

## Tier 3 — Track C (AI-agent consumability)

(T0.1/T0.2 above are the two most severe items in this track and must land before anything else here is trusted.)

- [ ] **T3.1 — Make the agent interface actually servable.** `NewMCPServer` currently has zero non-test callers — add a real `dimctl agent serve` (or equivalent) command that starts an actual listener, so M4.6 is reachable by something other than a unit test.
- [ ] **T3.2 — Resolve the transport question for real.** The plan's resolved answer was MCP *and* A2A *and* REST/JSON-RPC; the decision doc picked MCP only and "A2A" appears nowhere in the repo. Either implement the other two transports, or take the discrepancy back to the plan doc and get an explicit, written re-scoping — don't leave the decision doc silently contradicting the plan it's supposed to implement.
- [ ] **T3.3 — Replace the fake conformance example.** `examples/agent/route_validator.py` currently hardcodes fake responses (`"valid": True`, a made-up `route_version`) and never contacts `dim` — once T3.1 exists, make this a real client hitting the real server.
- [ ] **T3.4 — Version the interface at a boundary a third party can actually import.** `AgentInterfaceVersion` currently lives in an `internal/` package; move the versioned contract to a `pkg/`-level boundary, mirroring `pkg/sdk`'s pattern from Phase 3, so M4.6.3's intent (an external party can depend on this) is actually possible.
- [ ] **T3.5 — Generate the capability manifest for real**, from `schemas/route.schema.json` and the actual adapter/step registry — not the current hand-written Go literal, which is already missing `contract`, `claim_check`, `claim_resolve`, and every adapter added since Phase 3.
- [ ] **T3.6 — Wire `verify-manifest.sh` into CI** (`ci.yml` currently never invokes it), and make the check compare the manifest against the live schema/registry, not a static JSON snapshot of the same hand-written literal — otherwise it can only catch a forgotten refresh, never real drift.
- [ ] **T3.7 — Fix `checkContractEnforcement`** so it actually inspects whether `enforce: true` is set rather than firing on any route with a contract regardless of enforcement state — this is the exact case M4.8.2 was scoped to catch. Implement `checkUnusedImports` for real (it currently always returns nil despite being registered) and add the missing lineage-retention-policy check the milestone names.
- [ ] **T3.8 — Index contract references in the impact-analysis engine.** `ExtractAllReferences` currently never reads `route.Contracts`, so `ContractReferences` is permanently empty and every contract-impact query — the milestone's primary use case — returns a false "no routes affected, confidence 1.0." Fix the extractor, and add a `connection` case to `QueryImpact`'s switch so the one uncertainty bucket the code does produce can actually surface (currently it's built but unreachable). Rewrite `impact_query_test.go` and its sibling to call `BuildImpactIndex` on real fixtures rather than hand-constructing the index the code under test can't actually produce.
- [ ] **T3.9 — Expose the impact-analysis query surface via M4.6's interface** (currently missing from `registerOperations` entirely, contradicting M4.9.2/M4.7.3's requirement).

---

## Process addendum for this round specifically

Given that two operations in Track C were built to *return a passing result without doing the work*, and a UI badge was built to claim success with no check behind it — add this to whatever standing instruction governs Claude Code's own completion process: **an operation that cannot yet do what it claims must return an explicit error or an "unimplemented" result, never a fabricated pass.** This should be treated as harder ground truth in this project going forward than the "verify against the running system" instruction already was — that instruction alone wasn't enough to prevent this round's specific failure, because the code didn't fail to be checked, it was written to look checked.
