# dim — Declarative Integration Middleware

A declarative, configuration-driven integration middleware built on Enterprise Integration Patterns (EIP). Routes and transforms messages between heterogeneous systems using YAML-defined routes rather than hand-written glue code.

**Status:** Phase 0 — core engine scaffold, pre-implementation.

## Documentation

This project is designed per the specifications in `design/`:

- **[eip-middleware-design.md](design/eip-middleware-design.md)** — Core architecture and EIP mapping
- **[phase-0-implementation-plan.md](design/phase-0-implementation-plan.md)** — Phase 0 engineering plan
- **[data-mesh-feasibility-analysis.md](design/data-mesh-feasibility-analysis.md)** — Evaluation against data mesh principles
- **[data-mesh-reference-architecture.md](design/data-mesh-reference-architecture.md)** — Reference architecture and domain model proposal
- **[self-service-feasibility-study.md](design/self-service-feasibility-study.md)** — Self-service deployment feasibility

## Building

```bash
go build ./...
```

## Testing

```bash
go test ./...
```

## Development

This is a pre-implementation scaffold. Directory structure and Go module are in place; actual implementation of Phase 0 milestones follows the plan in `phase-0-implementation-plan.md`.

## License

Apache 2.0
