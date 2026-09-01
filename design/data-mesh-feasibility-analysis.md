# Data Mesh Feasibility Analysis

**Type:** Evaluation only — nothing in this document is adopted into the design
**Evaluated against:** Declarative Integration Middleware — Design Document, v7
**Date:** 2026-09-01

## Purpose and how to read this

This is an assessment, not a proposal. It asks one question — how well does the design in `eip-middleware-design.md` (v7) hold up against the requirements that arise when a system is meant to support data mesh principles from the beginning — and answers it section by section, without editing that design. Where a gap is identified, this document says so and roughly sizes it, but does not specify a fix; that's deliberately left for a future design round once you've decided which gaps are worth closing.

Data mesh, as originally framed by Zhamak Dehghani, rests on four principles: domain-oriented decentralized data ownership, data as a product, self-serve data infrastructure as a platform, and federated computational governance. Each gets its own section below. A fifth, cross-cutting section covers the output-port/data-contract question, which cuts across all four and turned out to be the single largest gap.

## Summary

| Principle | Alignment | Headline finding |
|---|---|---|
| Domain-oriented decentralized ownership | Partial | Routes are already independently deployable, declarative units — a good structural fit — but there's no namespace/domain concept and no described config-ownership or self-service deployment workflow. |
| Data as a product | Weak-to-partial | Addressable endpoints and OpenLineage-based discoverability exist; the core missing piece is a **data contract** — a versioned, published schema and SLO for what a source/sink actually emits, distinct from `route_version`. |
| Self-serve infrastructure as a platform | Strong | This is the design's best-aligned area almost by accident: declarative config plus automatic observability/lineage/reliability *is* the self-serve-platform promise. Gaps are in provisioning workflow and multi-tenant resource isolation, not in the core mechanism. |
| Federated computational governance | Strong foundation, narrow scope | `authorize` (PBAC/ABAC), the mandatory `auth:` declaration, and the retention-policy-plus-evidence-log pattern are genuine computational-governance mechanisms already — but they only cover access and retention today. Nothing generalizes the pattern to interoperability or data-quality standards yet. |
| Data contracts / output ports (cross-cutting) | Weak | Fan-out to multiple sinks already maps naturally onto "one data product, several output ports," but nothing ties those sinks together as *one product* with one versioned contract. This is the connective tissue the other four sections keep pointing back to. |

The short version: the design's operational spine — declarative routes, structural reliability, observability, lineage, authorization — turns out to be unusually good raw material for a data mesh platform, better than a generic EIP engine would be, because so much of what's already built (route-as-unit, PBAC, retention-with-evidence, OpenLineage export) is either identical to or one generalization away from what data mesh asks for. What's missing isn't scattered engineering gaps; it's almost entirely one concept — a first-class, versioned **data product contract** distinct from route/transformation logic — plus the organizational scaffolding (domain namespacing, config ownership) that a routing engine had no prior reason to model.

## 1. Domain-oriented decentralized data ownership

**What data mesh requires:** domain teams independently own, build, and deploy the pipelines that produce their own data, without routing every change through a central integration team. Ownership is visible in the architecture, not just in a wiki page.

**What the design already supports:**
- Tenet 1 (config is the source of truth) plus `imports`/`fragments` (§6.2) mean a route is already a self-contained, independently reviewable, independently deployable unit — a domain team's routes are files in a directory, not entries in someone else's codebase.
- Hot reload (§13.1) operates per route, not engine-wide — one domain's route changing and reloading has no effect on another domain's routes running in the same or a different instance. That's a meaningful structural precondition for independent deployment.
- Route-level `auth`, `lineage`, and `error_path` (§6, §10.2, §7.1) are configured per route, not centrally, which is the right granularity for domain autonomy — a domain team sets its own retention policy and its own error handling without needing an engine-wide change.

**Gaps:**
- **No namespace/domain concept.** `sources`, `sinks`, `functions`, and `fragments` all live in one flat global namespace (§6). There's no way to express "this route belongs to the payments domain" for ownership, discoverability, or config-level access control (who may edit which routes) — that's a different concern from `authorize`, which governs *data flowing through* a route at runtime, not *who may change the route's definition*.
- **No described multi-tenant isolation within a shared instance.** The deployment model (§13) already supports running separate instances per domain, which sidesteps the isolation problem entirely — but if the ambition is many domains sharing infrastructure, resource quotas and blast-radius isolation aren't designed (the roadmap, §18 Phase 3, already flags "multi-tenant policy isolation" as future work, which is the right instinct, just not yet fleshed out).
- **No self-service deployment/config-ownership workflow.** The document doesn't describe how a domain team proposes and ships its own route changes independently — GitOps, a control-plane API scoped to a domain's own routes, review ownership. This is likely more platform/process design than engine design, but it's a real precondition for "decentralized," not just an implementation detail.

## 2. Data as a product

**What data mesh requires:** each data product should be discoverable, addressable, trustworthy (with published SLOs), self-describing (a schema, not tribal knowledge), interoperable with the rest of the mesh via shared conventions, secure, and versioned — as a first-class published contract, not an implicit side effect of a pipeline existing.

**What the design already supports:**
- **Addressability** is solid: `sources`/`sinks` are named, stable endpoints (§6) — architecturally, a sink is already exactly the shape of a data product's output port.
- **Discoverability** gets real support from the OpenLineage export path (§10.5): job/run/dataset facets, with `route_version` attached, give an external catalog (Marquez or a commercial one) enough to know what data products exist and what touched them. This is a genuine, non-coincidental overlap — OpenLineage's dataset/job model maps closely onto a catalog entry.
- **Security** — principal propagation, `authorize`, OBO (§12) — is more thorough than most data mesh platforms bother to design at the pipeline level, since data mesh discussions usually treat this as "the platform's problem" without specifying it.

**Gaps — this is where the analysis turns up the design's biggest single hole:**
- **No data contract.** `route_version` (§10.1) versions the route's *transformation logic*. A consumer of a data product cares about whether the *shape of the data* changed — its schema — which is a related but distinct thing. Nothing in the design publishes, versions, or validates a payload schema independently of the route config that happens to produce it. A `translate` step can silently change the output shape and the only signal is a new `route_version` — which conflates "the logic changed" with "the data contract changed," when in practice many logic changes don't touch the contract and, more importantly, some do and today there's no automated way to flag that.
- **No SLO publication.** §9.1's metrics are real operational signals (latency, error rate, throughput) but they're internal to the engine's own observability stack, not a published, queryable contract a downstream domain can depend on ("this data product refreshes every 5 minutes, 99.9% of the time"). Data mesh wants that SLO to be a first-class, discoverable artifact, not something you infer from a Grafana dashboard you may not have access to.
- **No interoperability layer.** JSONata gives each route total flexibility to shape its own output, which is good for domain autonomy but has no corresponding mechanism encouraging or enforcing shared conventions across domains (common ID formats, common timestamp/currency conventions, a shared vocabulary) — that's arguably as much a governance question (§4 below) as a schema one, but it belongs in this principle too since interoperability is explicitly one of data mesh's data-product quality dimensions.

## 3. Self-serve data infrastructure as a platform

**What data mesh requires:** a platform domain teams can build against without needing deep infrastructure expertise — standing up a new data product's pipeline should come with observability, lineage, reliability, and governance "for free," not something each domain team re-implements.

**What the design already supports — this is the strongest match of the four:**
- Declarative config (tenet 1) + pluggable adapters (tenet 6) + `imports`/`fragments` (§6.2) mean a domain team writes YAML, not code, to stand up a pipeline — and gets dead-letter handling, retry, metrics, tracing, lineage, and authorization automatically, just by the route existing (§7, §9, §10, §12 are all structural, not opt-in add-ons a team has to remember to wire up). This is, almost exactly, what a data mesh "self-serve platform" is supposed to feel like from a domain engineer's seat.
- The Tier 1 built-in viewer and `midctl` tooling (§9.3, §9.4, §10.7, §10.3) give domain teams self-service debugging and provenance lookup without needing to file a ticket with a platform team — again, directly the self-serve promise, not adjacent to it.
- `fragments` (§6.2) are architecturally capable of encoding a "standard data-product wrapper" (a shared fragment every domain route composes for baseline validation/dedup/tagging) — this isn't a designed feature today, but it's a natural, low-effort extension of a mechanism that already exists, which is worth noting precisely because it means this gap is cheap to close later if wanted.

**Gaps:**
- No standardized "data product" template or scaffold — nothing generates a new domain's route skeleton with the right shape pre-filled.
- No provisioning workflow beyond "write YAML and deploy" — fine for a small number of sophisticated teams, less so at real data-mesh scale with many less infrastructure-savvy domain teams.
- Multi-tenant resource quotas/fairness, as noted in §1 above, aren't designed — a genuinely shared self-serve platform running many domains' workloads needs noisy-neighbor protection that a single-team deployment doesn't.

## 4. Federated computational governance

**What data mesh requires:** global standards — security, privacy, interoperability, quality — enforced *automatically*, computationally, across every domain's pipelines, with the standards themselves set through a federated process (platform team plus domain representatives), not centrally dictated and manually checked.

**What the design already supports — the second-strongest match, and arguably the most surprising one:**
- The `authorize` step with pluggable PBAC over a formally specified, neutral PDP contract (§12.2, §12.3) *is* a computational-governance mechanism in the textbook sense: policy-as-code, evaluated automatically, per message, engine-agnostic so an org's existing policy stack can be the actual source of truth.
- The mandatory `auth:` declaration, schema-enforced once a team opts into `enforce` (§6.3, §12.5), is computational governance applied to the *shape of a route's config itself* — a route that doesn't declare its authorization stance fails validation, automatically, rather than relying on someone noticing in review.
- Retention policies plus the reaper plus the purge evidence log (§10.2, §10.4) are, right now, the single best-realized example in the whole design of data mesh's federated-computational-governance idea: a policy is declared once (a named `retention_policy`), applied automatically across every domain's routes that reference it, and — critically — *provably* enforced via the evidence log, not just assumed. If nothing else in the design changes, this pattern is the one worth recognizing as already "data-mesh-shaped" governance, just currently scoped to one concern (retention).

**Gaps:**
- **Narrow scope.** Authorization and retention are covered; interoperability conformance and data-quality rules are not. There's no generalized "policy gate" step beyond `authorize` — extending the same pattern to, say, a `validate-contract` step that computationally checks a message against a published data contract (see §2's gap) is a natural, structurally consistent extension, but it doesn't exist today.
- **No mechanism to make governance mandatory across domain-owned config.** `imports`/`fragments` (§6.2) could technically carry a centrally-governed "every route must import this" fragment, but nothing today distinguishes a domain's own fragments from ones a federated governance body requires — there's no way to guarantee a domain team's route actually pulled in the mandatory governance fragment short of manual review, which undercuts the "computational" half of "federated computational governance."
- **No described federation process.** This is partly organizational rather than technical, and the design is right not to prescribe an org chart — but it could describe *where* federally-agreed policy gets expressed technically (a versioned, centrally-owned fragment/policy library that domain teams import but don't modify) even without specifying who sits on the governance body.

## 5. Cross-cutting: data products need multiple output ports on one contract

Data mesh data products commonly need to be consumable more than one way from a single canonical source — the same domain data as a streaming topic for one class of consumer and as a batch file/table for another, kept in sync and describing the same underlying thing.

The design's `route` step's fan-out capability (a single case can list more than one `to` target, §4) already maps onto this well structurally — one route, transformed once, can write to a Kafka sink and a file sink from the same pipeline run. This is a real strength worth calling out because it wasn't designed for data mesh and still fits.

What's missing is the connective tissue: nothing today declares "these three sinks are the same data product's output ports and must stay contract-consistent." That's the data-contract concept from §2 again, from a different angle — without it, "multiple ports, one product" is something a domain team can build by convention, but the engine has no way to enforce or even represent that the sinks are related.

## What this suggests about ordering, if you decide to act on it

Not a plan — just an observation, since a plan wasn't asked for this round: nearly every gap identified above traces back to one missing concept (a first-class, versioned data contract, decoupled from `route_version`) plus one missing organizational concept (a domain/namespace layer). Both are additive to the current design rather than corrections to it — nothing in v7 would need to be walked back to add either. That's worth knowing now even though neither is being specified yet.
