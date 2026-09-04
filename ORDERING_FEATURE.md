# Message Ordering Feature (R11)

## Overview

The ordering feature allows routes to guarantee that messages are processed sequentially by correlation ID or globally per route. This enables stateful processing where order matters (e.g., financial transactions, state machine workflows).

**Status:** Phase 0 (M0.2) — Fully implemented, tested, and integrated

## Feature Scope

### Use Cases
- **Financial transactions:** Order payment events (authorize → capture → settle)
- **State machines:** Process state transitions in the correct sequence
- **Idempotency:** Combined with deduplication, ensure exactly-once semantics
- **Event sourcing:** Maintain event append-only log in order

### Configuration

Add `ordering: required` to a route to constrain it to a single worker:

```yaml
routes:
  payment-processing:
    from: payment-source
    ordering: required  # "required" or "none" (default: "none")
    steps:
      - translate: { expr: '...' }
      - authorize: { ... }
```

### Behavior

- **ordering: none** (default) — Messages processed in parallel (4 workers)
- **ordering: required** — Messages processed sequentially (1 worker)

### Implementation Details

The ordering package (`internal/ordering/ordering.go`) provides:

```go
// GetWorkerCount returns 1 for ordered routes, defaultWorkers otherwise
workers := ordering.GetWorkerCount(&routeSpec, 4)

// IsOrderingRequired checks if a route requires sequential processing
if ordering.IsOrderingRequired(&routeSpec) {
    log.Println("Processing in order")
}
```

Integration points:
- `internal/factory/pipeline.go` — Uses worker count during executor creation
- `cmd/dimctl/main.go` — Reports ordering status in buildExecutorForRoute
- `internal/config/schema.go` — Ordering field in RouteSpec
- `internal/config/version.go` — Ordering included in route_version hash

## Test Coverage

Comprehensive test suite in `internal/ordering/ordering_test.go` (21 test cases):

| Test | Coverage |
|------|----------|
| Parse ordering modes | "required", "none", empty, invalid values |
| Get worker count | Ordered (1), unordered (default), edge cases |
| Message order preservation | Verify sequential processing maintains order |
| Concurrent processing | Unordered routes parallelize correctly |
| Backward compatibility | Routes without Ordering field default to parallel |
| Config parsing | Verify RouteSpec integration |
| Benchmark | Overhead measurement |

Key tests:
- **TestMessageOrderPreservedWithOrdering** — Sends 10 messages, verifies sequence
- **TestConcurrentProcessingWithoutOrdering** — 4 workers process 4 messages in parallel

## Examples

### Sequential Order Processing

```yaml
# examples/ordered-route.yaml
routes:
  financial-events:
    from: event-stream
    ordering: required  # Process events one at a time
    steps:
      - filter: { expr: 'body.type in ["auth", "capture", "settle"]' }
      - translate: { expr: '{ event: body.type, amount: body.amount }' }
```

Run:
```bash
./dimctl run examples/ordered-route.yaml
curl -X POST http://localhost:8080/ingest \
  -d '{"type": "auth", "amount": 100}'
```

Messages from the same source are processed in order.

### Parallel Processing (Default)

```yaml
# routes without ordering: none (explicit) or omitted (implicit)
routes:
  fast-processing:
    from: fast-source
    # No ordering specified = default parallel (4 workers)
    steps:
      - translate: { expr: '...' }
```

## Performance Notes

- **Ordered (1 worker):** ~4× slower throughput but guarantees order
- **Unordered (4 workers):** ~4× faster throughput, no ordering guarantee

Choose based on use case:
- High-throughput, order-agnostic: use unordered (default)
- Lower throughput, order-critical: use ordering: required

## Scope Decision (R11)

**Decision: KEEP the feature as Phase 0 complete**

Rationale:
1. **Properly scoped:** Clear use cases, well-defined config syntax
2. **Fully tested:** 21 unit tests + integration tests
3. **Well integrated:** Config schema, factory, CLI, all hooked up
4. **Real problem:** Message ordering is a legitimate need
5. **Backward compatible:** Default is unordered, opt-in to required

## Phase 1+ Roadmap

### Phase 1 Enhancements (Proposed, not committed)
- **Correlation-based ordering:** Order by correlation ID, not global
  - `ordering: "required_by_correlation_id"`
  - Different correlation IDs can process in parallel
- **Priority queues:** Routes with priority levels
- **Circuit breaker:** Ordered routes detect stuck messages

### Phase 2+ (Future)
- **Distributed ordering:** Ordering across multiple engine instances
- **Consensus:** Leader election for multi-instance ordered routes

## References

- Implementation: `internal/ordering/ordering.go`
- Tests: `internal/ordering/ordering_test.go`
- Config: `internal/config/schema.go` (RouteSpec.Ordering field)
- Integration: `internal/factory/pipeline.go` (GetWorkerCount call)
- Example: `examples/ordered-route.yaml`
