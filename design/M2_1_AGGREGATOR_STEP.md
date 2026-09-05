# M2.1 Aggregator Step — Design

**Status:** Design phase (M2.1.1)  
**Date:** 2026-09-04

## Overview

The Aggregator step collects related messages by a correlation strategy and emits a single combined message when a completion trigger fires. It mirrors the EIP Aggregator pattern, supporting both count-based and time-window-based completion strategies.

## Completion Strategies

### Count-Based Completion
- **Trigger:** Aggregation group reaches configured size
- **Config:**
  ```yaml
  - aggregate:
      correlation_key: body.order_id
      completion_strategy: count
      count: 10
      timeout_ms: 60000  # fallback time-window
  ```
- **Semantics:** Emit the combined message as soon as group reaches `count` messages; if not reached within `timeout_ms`, emit partial group

### Time-Window-Based Completion
- **Trigger:** Time window expires since first message in group
- **Config:**
  ```yaml
  - aggregate:
      correlation_key: body.order_id
      completion_strategy: time_window
      window_ms: 5000
      max_messages: 100  # safety valve
  ```
- **Semantics:** Emit the combined message every `window_ms` milliseconds or when group reaches `max_messages`, whichever comes first

### Correlation Key
- **Expression:** JSONata, evaluated per message
- **Example:** `body.order_id` (group by order ID)
- **Null handling:** Messages with null correlation keys are either:
  - Option A (chosen): Routed to error_path (cannot be aggregated)
  - Option B: Aggregated into their own group
- **Selected:** Option A (fail fast, auditable)

## Aggregation Output

### Combined Message Structure
```json
{
  "messages": [
    { "body": {...}, "headers": {...} },
    { "body": {...}, "headers": {...} }
  ],
  "correlation_key": "order-123",
  "count": 10,
  "first_ingested": "2026-09-04T12:34:00Z",
  "last_ingested": "2026-09-04T12:34:05Z"
}
```

### Lineage Handling
- **Parent relationships:** Aggregator output has N parent lineage records (one per input message)
- **Lineage facet:** `AggregatorFacet` captures:
  - `input_count`: number of messages aggregated
  - `correlation_key`: grouping key
  - `completion_strategy`: "count" or "time_window"
  - `completion_value`: triggering value (count or window in ms)
  - `completion_trigger`: "count_reached" / "timeout" / "window_expired"
- **Subject tracking:** Aggregator preserves first input's `subject_id` in output

### Hot Reload & State Durability

**Generation-Aware Draining:**
- On config reload (new generation starts), old generation's in-flight aggregation groups must be preserved
- Old generation cannot start new groups (new traffic routes to new generation)
- Old generation drains existing groups until:
  - Completion trigger fires (emit accumulated messages)
  - Timeout expires (emit partial group)
  - Generation is superseded by a newer generation (drain remaining groups immediately)

**In-Flight Group Tracking:**
- Aggregator maintains `drainCaps` channel (max 10 concurrent drains, per AMQP pattern)
- Each group has a `drainTimeout` (time remaining until forced flush)
- On drain request, all in-flight groups emit immediately, marking each as "drained"

**Shutdown Behavior:**
- On clean shutdown, all in-flight groups are flushed with `completion_trigger: "shutdown"`
- On forced shutdown (timeout), groups reaching age threshold are dropped with warning

## Configuration Schema

```yaml
routes:
  - name: order-batch
    from: line-items
    steps:
      - aggregate:
          correlation_key: body.order_id      # JSONata expression
          completion_strategy: count           # or "time_window"
          
          # Count strategy
          count: 10                            # emit when N messages collected
          timeout_ms: 60000                    # or after N ms (whichever first)
          
          # Time-window strategy
          window_ms: 5000                      # emit every N ms
          max_messages: 100                    # or when N messages collected
          
          # Common
          output_expr: |                       # optional: transform combined output
            {
              "order_id": correlation_key,
              "items": $map(messages, $ -> $.body),
              "total_items": count
            }
          
          # Error handling
          on_null_correlation: error_path      # or "skip" (not recommended)
          on_error: error_path
```

## Ordering Guarantees

**Within a group:** Messages are emitted in ingestion order (FIFO per correlation key)  
**Across groups:** No guarantee (groups emit as completion triggers fire)

## Examples

### Order Line-Item Batching
```yaml
- aggregate:
    correlation_key: body.order_id
    completion_strategy: count
    count: 10
    timeout_ms: 30000
```
Collects up to 10 line items for an order, or flushes every 30 seconds.

### Event Windowing
```yaml
- aggregate:
    correlation_key: body.session_id
    completion_strategy: time_window
    window_ms: 5000
    max_messages: 1000
```
Batches session events every 5 seconds or when session reaches 1000 events.

## Testing Strategy

- **Unit tests (M2.1.2):**
  - Count-based completion
  - Time-window completion
  - Null correlation key routing
  - Lineage attribute generation
  - Hot reload with in-flight groups

- **Integration tests (M2.1.3):**
  - End-to-end route with aggregation
  - Multiple correlation keys flowing simultaneously
  - Hot reload without message loss
  - Lineage verification (parent-child relationships)

## Exit Criteria

✓ Route can aggregate N messages by count trigger  
✓ Route can aggregate N messages by time-window trigger  
✓ Survives hot reload without losing in-flight groups  
✓ Lineage correctly attributes output to all inputs  
✓ Configuration validated (correlation_key, completion_strategy)  
✓ Output message structure matches spec  
✓ Tests: 10+ unit + integration tests, all passing
