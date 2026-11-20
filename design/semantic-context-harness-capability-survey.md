# Semantic/Context Harness — What dim Can and Can't Already Do

**Scope note:** this is not a design for the harness. The harness — a chakraview.com-branded toolkit that gives a customer's AI agents a domain-scoped sense of what words mean in context, so "approved" or "deploy" doesn't carry its meaning from one task into another — is explicitly its own piece of software and needs its own dedicated design later. This document only answers the narrower question asked first: of everything that already exists in `dim`, what's usable as-is, what's close but needs extension, and what's simply absent, if `dim` became the substrate under that harness.

No decision is made here about whether the harness is a new phase of `dim` itself or a separate, lighter-weight service that sits beside it — that was explicitly left open. The findings below are relevant to either choice: anything marked "reusable" is reusable under either structure; anything marked "absent" has to be built either way, the only question is in which codebase.

## The core distinction the vision rests on

"Semantic" here does not mean structural (a JSON schema, a field type, a `contract_version`). `dim` already has a strong structural-contract concept — PDP-validated, versioned, enforced at the route boundary. That's a solved problem and not what was asked for. The vision is about *meaning*: the same string, in two different domains or tasks, resolving to two different, non-transferable things. That's a fundamentally different kind of object than anything `dim` validates today, and no part of the existing contract system does it, even loosely. This is the one finding that should drive every other decision: **a term/meaning registry, scoped by domain, is new work no matter which codebase it lands in.**

## What's directly reusable: the fail-closed, domain-scoped resolution pattern

This is the strongest finding in dim's favor. The exact shape the vision needs — "look up X in the context of domain D; if X doesn't exist in D's scope, fail closed, never silently fall back to some other scope's value" — already exists, just war-tested for a long time, in `internal/secrets/resolver.go` and `store.go`. It took three rounds of it being broken in subtle ways (a denial returning `nil` error, an adapter-level env-var fallback that didn't know the difference between "no domain configured" and "denied") before it was actually fixed, which is itself useful: it's a working demonstration of exactly the failure modes a semantic-resolution layer would need to avoid (a term lookup that quietly inherits the wrong domain's definition instead of erroring is the semantic equivalent of a secret leaking cross-domain). A terminology store built the same way — `Resolve(requestingDomain, term) (definition, error)`, errors distinguishable via `errors.Is` between "not defined here" and "explicitly scoped elsewhere" — is a close structural copy of code that already exists and is now correct. This is the single best argument for building the semantic layer *as* a `dim` concept rather than reinventing domain-scoping from zero in a separate service.

## What generalizes well: lineage and impact analysis

The lineage store and the M4.9 impact-analysis work (querying "what does changing X affect") are aimed at schema and route changes today, but the underlying question — "if I change this, what downstream things does that reach" — is exactly the governance question a meaning registry needs too: if "approved" is redefined in the `finance` domain, what routes, agents, or downstream contracts were relying on the old meaning? The indexing approach (and its known bug, from the last remediation round, where the connection-uncertainty case is unreachable because the extractor never populates a connection name) is worth reading before building the equivalent for terms, both as a pattern to copy and as a specific bug class to not repeat.

## What generalizes well: GitOps authoring and scaffolding

Track A's self-service pipeline (M4.1 GitOps, M4.2 `dimctl scaffold`) is aimed at routes, but the pattern — human-reviewable YAML in git, a CLI that scaffolds a new one from a template — is exactly how you'd want a business's term definitions authored and approved too (a pull request that says "`deploy` in the `releng` domain now means X" is a natural, auditable unit, the same way a new route is). This is a pattern to copy, not code to reuse directly, since it's currently wired to routes specifically.

## What's a real, immediate gap: reaching "wherever the customer's data resides"

This is the most material finding, and it isn't hypothetical — I confirmed it directly in the running code. `dim` has adapter code for file, SFTP, Kafka, AMQP, S3, and database sources, but **none of them are wired into the live pipeline.** `BuildSingleRoutePipelineWithTracing` and `BuildMultiRoutePipelineWithTracing` hard-code an HTTP source for every route, unconditionally; the parsed `sources:` config block is referenced exactly twice in all of production code, both just a count for a status print. If the harness's job is to sit in front of a customer's actual systems — their database, their file drops, their message queues, wherever the data they want grounded actually lives — `dim` cannot do that today for anything but an HTTP push. This isn't specific to the semantic-layer use case; it's a standing gap noted in the separate Phase 4 status doc. But it becomes load-bearing the moment "wherever the customer's data resides" is a requirement, so it's worth flagging here specifically rather than leaving it as background Phase 4 debt.

## What's adjacent but not yet trustworthy: the agent-facing API (Track C)

M4.6 (agent operation interface) and M4.7 (capability manifest) are the closest existing concept to "agents query this system for something," but they're aimed at a different axis — letting an agent *operate dim itself* (configure routes), not letting an agent *ask dim what a word means*. They're also, as of the last verification round, not reliable: `GetCapabilities` (what's actually served to callers) is a hardcoded literal disconnected from the real schema, and the CI check meant to catch that kind of drift has failed three live-simulation tests in a row. The "agents get no shortcut" governance principle behind Track C — agent-authored or agent-consumed material gets identical validation to human-authored material — is exactly the right principle for a meaning registry too (an agent should no more be able to carry a stale or wrong-domain definition forward than it can bypass a contract), but the Track C *code* isn't in a state to build on top of yet.

## What's a viable extension point: the plugin SDK

If the harness eventually wants any intelligence beyond pure lookup — detecting that two domains are using the same word with drifted meanings, flagging a likely collision before an agent acts on it, embedding-based similarity, whatever — the WASM/native plugin SDK (`pkg/sdk`) is a real, working extension point for that kind of custom step today, independent of everything else on this list.

## Net assessment

| Need | Status in dim today |
|---|---|
| Domain-scoped resolution, fail-closed on cross-domain reach | **Reusable now** — the secrets resolver is a working template, just fixed |
| Audit of what a meaning-change affects | **Generalizes well** — lineage/impact-analysis exists, aimed at the wrong object (schema, not terms), has one known indexing bug to avoid repeating |
| Governed, reviewable authoring of definitions | **Generalizes well** — GitOps/scaffold pattern exists, currently wired to routes only |
| Reaching the customer's actual systems (DB, files, queues) | **Real gap** — adapter code exists, nothing wires it into a live pipeline except HTTP |
| Agent-facing query surface | **Adjacent, not trustworthy yet** — right principle (no agent shortcut), wrong axis (operate vs. query), currently broken in the one place it's built |
| Meaning/term registry as a first-class object | **Absent entirely** — contracts are structural, not semantic; nothing in dim today represents "what a word means here" |
| Custom intelligence (collision detection, embeddings) | **Viable extension point** — plugin SDK, unused for this so far |

The honest summary: `dim` is a better starting substrate for this than it might look at first glance, almost entirely because of one specific, recently-hardened piece of it — the domain-scoped, fail-closed resolver pattern — plus a lineage/impact-analysis engine and a GitOps authoring pattern that both generalize cleanly from "route" to "term." But the thing the vision is actually *about* (meaning, not structure) doesn't exist in any form yet, and the thing that would make it useful against real customer systems (non-HTTP source connectivity) is currently broken across the board, not just for this use case. Any dedicated harness design should treat the resolver pattern as a reusable foundation, and treat source connectivity as a prerequisite to solve either inside dim or deliberately outside it, rather than something the harness can quietly depend on today.
