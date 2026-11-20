# Phase 5 Implementation Plan

**Origin:** `design/semantic-context-harness-capability-survey.md` identified gaps between what `dim` can do today and what's needed next. Two are scheduled here. A third — a domain-scoped meaning/term registry — needs real design work before it's schedulable and is tracked in `design/roadmap.md` instead.

**Relationship to Phase 4:** Phase 4's remediation backlog is closed as of round 4 (`design/phase-4-remediation-verification-round4.md`) — studio save-path, capability manifest, catalog filtering, and the lower-severity items are all genuinely fixed, verified by execution. M5.1 was never blocked on that backlog. M5.2 was blocked on it and is now unblocked, with one caveat noted below.

## M5.1 — Live source connectivity

**Problem:** `dim` has working adapter code for file, SFTP, Kafka, AMQP, S3, and database sources, but none of them are reachable from a real route. `BuildSingleRoutePipelineWithTracing` and `BuildMultiRoutePipelineWithTracing` hard-code an HTTP source for every route, unconditionally; the parsed `sources:` config block is referenced in exactly two places in production code, both just a count for a status print. Confirmed independently by two separate reviewers during round 4 of Phase 4 remediation — nothing in that round touched this.

**Scope:**
- Add source-type dispatch to the pipeline factory: read each entry in `cfg.Sources`, switch on `.Type`, and construct the matching adapter (`file`, `sftp`, `kafka`, `amqp`, `s3`, `database`) instead of the current unconditional `http.NewHTTPSource`.
- Every constructed source must support domain-scoped secret resolution, consistent with the pattern M4.4 established for file/SFTP. **Asymmetry to close:** only `file.FileSource` and `file.SFTPSource` currently have the `NewXWithDomain(..., domain string, resolver *secrets.Resolver)` constructor variant. `kafka.NewKafkaSource`, `amqp.NewAMQPSource`, `s3.NewS3Source`, and `database.NewDatabaseSource` do not — they take no domain or resolver argument at all. Each needs the equivalent domain-aware constructor added before (or as part of) wiring it into the dispatch, or the fail-closed guarantee M4.4 earned for file/SFTP quietly doesn't apply to the other four source types.
- A route's `sources:` entries should support the same `domain:` field the route itself carries (or inherit it from the owning route — flagged earlier as an open design question and should be settled as part of this milestone, not left implicit).
- Multi-route pipelines (`BuildMultiRoutePipelineWithTracing`) need the same dispatch, not just the single-route path.
- **Fold in the round-4 finding on the file source:** `internal/adapters/file/file_source.go`'s `pollLocal` still does a plain `os.Stat`/`os.ReadFile` with no stability check — the fix that landed during Phase 4 remediation (`ba848be`) only changed the integration test's own fixture-writer, not the poller itself. A real external producer writing to that file mid-poll can still be read partially. This needs an actual production fix (a stability threshold, or detecting and waiting out an in-progress write) before the file source is wired up for real use — not deferred to "later," since M5.1 is the milestone making this adapter reachable in the first place.
- **Scoping note, not a blocker:** the Phase 4 round-4 fix for shared output/error sinks across routes still hardcodes the output sink's name to the literal `"output"` config key — distinct per-route sink names aren't supported. If M5.1's multi-route work combines with multi-source work in a way that assumes distinct per-route sinks, check this first; it isn't fixed yet.

**Acceptance criteria:**
- A route configured with `sources: { type: file, ... }` (or `sftp`, `kafka`, `amqp`, `s3`, `database`) actually receives messages from that source when run — proven by execution, not by a passing unit test with a mocked constructor.
- A cross-domain secret reference in any of the six source types fails closed, proven the same way T1.13's fix was finally proven: an actual attempt to leak a same-named env var or cross-domain secret, that fails.
- The file source's poller is proven stable against a real mid-write read: a test that writes a file slowly (e.g. in chunks with delays) while the source polls concurrently, confirming no partial/empty message is ever emitted — not just the existing fixture-level test that writes atomically by construction.
- `dimctl catalog` (or the discovery surface) reports source reachability truthfully, not just a count.

## M5.2 — Agent-facing query surface

**Clarification, not new scope:** "agent-facing query surface" means the same transport Track C's M4.6 was already scoped for — MCP and A2A, plus REST/JSON-RPC as a fallback. Nothing here proposes a different interface.

**Status update:** this milestone was previously blocked on Track C's data correctness, which round 4 confirmed is fixed — `GetCapabilities` is genuinely derived from the live schema, and the drift check now actually catches a staleness scenario it failed three times before. That blocker is cleared. What round 4 did **not** verify is whether M4.6's actual MCP/A2A wire-protocol transport exists and serves that data end-to-end — the round-4 reviews confirmed the underlying Go-level data and HTTP-layer correctness, not a live MCP or A2A client round-trip. Treat that as the open question this milestone starts from, not an assumption.

**Scope:**
- Confirm whether an MCP server and an A2A server actually exist and are reachable today. If they do, connect a real client of each protocol, invoke an operation, and confirm the now-correct capability/impact-analysis data comes back over the wire, not just from the internal Go call.
- If either transport doesn't exist yet, building it is in scope here — this milestone is "the query surface actually works end-to-end," not "re-confirm data that's already been confirmed."
- Treat this surface as the one any future work (including whatever eventually comes out of the meaning/term registry roadmap item) will ride, rather than growing a second, parallel API later.

**Acceptance criteria:**
- A real MCP client can invoke at least one read operation (e.g. fetch the capability manifest, or run an impact-analysis query) and get back data that matches what was verified correct in round 4.
- A real A2A client can do the same, over A2A's actual protocol framing, not a REST call dressed up as A2A.
- REST/JSON-RPC fallback is confirmed to serve the same data as the other two transports, not a separately-maintained copy that can drift.
