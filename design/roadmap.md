# Roadmap — Unscheduled Items

Living list of things that are real, intended, and not yet schedulable — either because the shape of the work isn't decided, or because they're waiting on a prerequisite that isn't close enough to plan against yet. Nothing here has a milestone number. When an item is ready to be scheduled, move it into the relevant phase's implementation plan and delete it from here.

## Meaning/term registry

**Origin:** `design/semantic-context-harness-capability-survey.md`.

A domain-scoped store of what a word means in context (so "approved" or "deploy" doesn't carry its meaning from one task or domain into another), resolved the same fail-closed way `dim`'s secrets resolver works today, with the lineage/impact-analysis pattern available to answer "what does redefining this term affect."

It needs real design work before it's schedulable:
- What a "term" is as a data model — not a JSON-schema-style structural contract; `dim` has nothing like this today.
- How definitions get authored and reviewed — likely following the GitOps/scaffold pattern from Track A (Phase 4), where a pull request is the natural, auditable unit.
- How collision detection would work, if wanted at all — the plugin SDK (`pkg/sdk`) is a viable extension point for embedding-based similarity or NLP-driven detection, unused for this so far.
- Whether it lives inside `dim` as a new subsystem or as a separate service that calls into `dim`'s existing domain-scoping and lineage machinery — not decided.

**Revisit when:** Phase 5's M5.1 (live source connectivity) is done.
