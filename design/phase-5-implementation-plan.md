# Phase 5 Implementation Plan — Harness Foundations

**Origin:** the semantic/context harness capability survey (`design/semantic-context-harness-capability-survey.md`) identified three gaps between what `dim` can do today and what a chakraview.com harness would need. Two of them are scheduled here. The third is explicitly deferred.

**Relationship to Phase 4:** independent track. Phase 4's own remediation backlog (studio save-path bugs, the unwired capability manifest, the catalog filtering regression) is being picked up separately and is not a prerequisite for M5.1. M5.2 does depend on Track C's existing bugs being fixed first (see below).

## M5.1 — Live source connectivity (the big gap)

**Problem:** `dim` has working adapter code for file, SFTP, Kafka, AMQP, S3, and database sources, but none of them are reachable from a real route. `BuildSingleRoutePipelineWithTracing` and `BuildMultiRoutePipelineWithTracing` hard-code an HTTP source for every route, unconditionally; the parsed `sources:` config block is referenced in exactly two places in production code, both just a count for a status print. A harness meant to reach "wherever the customer's data resides" cannot do that today for anything but an HTTP push.

**Scope:**
- Add source-type dispatch to the pipeline factory: read each entry in `cfg.Sources`, switch on `.Type`, and construct the matching adapter (`file`, `sftp`, `kafka`, `amqp`, `s3`, `database`) instead of the current unconditional `http.NewHTTPSource`.
- Every constructed source must support domain-scoped secret resolution, consistent with the pattern M4.4 just established for file/SFTP. **Important asymmetry found while scoping this:** only `file.FileSource` and `file.SFTPSource` currently have the `NewXWithDomain(..., domain string, resolver *secrets.Resolver)` constructor variant. `kafka.NewKafkaSource`, `amqp.NewAMQPSource`, `s3.NewS3Source`, and `database.NewDatabaseSource` do not — they take no domain or resolver argument at all. Each of those needs the equivalent domain-aware constructor added before (or as part of) wiring it into the dispatch, or the fail-closed guarantee M4.4 just earned for file/SFTP quietly doesn't apply to the other four source types.
- A route's `sources:` entries should support the same `domain:` field the route itself carries (or inherit it from the owning route — this was flagged as an open design question in the earlier secrets investigation and should be settled as part of this milestone, not left implicit).
- Multi-route pipelines (`BuildMultiRoutePipelineWithTracing`) need the same dispatch, not just the single-route path.

**Acceptance criteria:**
- A route configured with `sources: { type: file, ... }` (or `sftp`, `kafka`, `amqp`, `s3`, `database`) actually receives messages from that source when run — proven by execution, not by a passing unit test with a mocked constructor.
- A cross-domain secret reference in any of the six source types fails closed, proven the same way T1.13's fix was finally proven: an actual attempt to leak a same-named env var or cross-domain secret, that fails.
- `dimctl catalog` (or the discovery surface) reports source reachability truthfully, not just a count.

## M5.2 — Agent-facing query surface (confirmed: MCP/A2A, not a new transport)

**Clarification, not new scope:** "agent-facing query surface" means the same transport Track C's M4.6 was already scoped for — MCP and A2A, plus REST/JSON-RPC as a fallback. Nothing here proposes a different interface. The point of calling this out as its own milestone is sequencing, not design: Track C's current implementation (`GetCapabilities`, the capability manifest, the impact-analysis query path) is not yet trustworthy enough to build on — the manifest is a hardcoded literal disconnected from the real schema, and the CI drift-check has failed three live-simulation tests in a row. M5.2 is: once Track C's own remediation is done (in the separate Phase 4 pass), confirm the MCP/A2A/REST-JSON-RPC surface is actually serving correct, current data, and treat that surface as the one both the harness's own tooling and any future term/meaning registry will ride — rather than either of them growing a second, parallel API.

**Acceptance criteria:**
- Deferred until Phase 4's Track C remediation lands. No independent work scheduled here beyond that confirmation.

## Deferred to roadmap — not scheduled

**Meaning/term registry.** This is the actual substance of the harness vision — a domain-scoped store of what a word means in context, resolved the same fail-closed way secrets are, with the lineage/impact-analysis pattern available to answer "what does redefining this term affect." It needs real design work before it's schedulable: what a "term" is as a data model, how definitions get authored and reviewed (likely following the GitOps/scaffold pattern from Track A), how collision detection works (the plugin SDK is a viable extension point if embedding-based similarity or NLP is wanted), and whether it lives inside `dim` or as a separate service that calls into it. None of that is decided yet. Revisit once M5.1 is done and the harness itself has a dedicated design.
