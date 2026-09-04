# M1.8: Replay Tooling — Dead-Letter Recovery & Audit Trail

**Phase:** Phase 1  
**Milestone:** M1.8 (Replay tooling)  
**Status:** Design & Implementation Plan  
**Date:** 2026-09-04

## Overview

M1.8 implements comprehensive replay capabilities for recovering messages from dead-letter queues (DLQs). The `dimctl replay` command allows selective replay of failed messages back into routes with safety guarantees:
- **Idempotency safety**: Replayed messages deduplicated if idempotent step configured
- **Audit trail**: Lineage records every replay operation
- **Selective replay**: Filter by error type, timestamp, correlation ID

## Subtasks

### M1.8.1: dimctl replay Command

**Objective:** Implement `dimctl replay` for DLQ recovery

**Command Structure:**
```bash
dimctl replay [options] <target-route>
  --from <sink-name>                # Source sink (e.g., orders-dlq)
  --filter <expression>              # Filter (e.g., 'error_type == "timeout"')
  --limit <count>                    # Max messages to replay (default: 100)
  --dry-run                          # Preview without replaying
  --parallel <count>                 # Parallel threads (default: 1)
```

**Examples:**
```bash
# Replay all messages from orders-dlq
dimctl replay --from orders-dlq process-orders

# Replay only timeout errors
dimctl replay --from orders-dlq --filter 'error_type == "timeout"' process-orders

# Preview what would be replayed
dimctl replay --from orders-dlq --filter 'error_type == "timeout"' --dry-run process-orders

# Replay 10 messages in parallel
dimctl replay --from orders-dlq --limit 10 --parallel 4 process-orders
```

**Implementation:**
```go
type ReplayCmd struct {
    From      string // Source sink (DLQ)
    TargetRoute string
    Filter    string // JSONata expression
    Limit     int    // Max messages
    DryRun    bool
    Parallel  int
}

func (rc *ReplayCmd) Execute(ctx context.Context) (*ReplayResult, error) {
    // 1. Load DLQ messages
    // 2. Apply filter (JSONata)
    // 3. Validate messages can be replayed
    // 4. (optional) Dry-run preview
    // 5. Enqueue messages back to source
    // 6. Return result summary
}

type ReplayResult struct {
    Total      int
    Replayed   int
    Failed     int
    Skipped    int
    Errors     []string
    Duration   time.Duration
}
```

**Exit Criteria:**
✅ `dimctl replay` command parses arguments  
✅ Filters DLQ messages by expression  
✅ Dry-run preview works  
✅ Parallel replay respects concurrency limit  
✅ Result summary accurate  

### M1.8.2: Replay Safety with Idempotency

**Objective:** Ensure replayed messages don't duplicate

**Idempotent Deduplication Flow:**
```
Original message (in-flight, fails) → DLQ
  correlation_id = "corr-12345"
  attempt_count = 1

Replay triggered → Message re-enters route
  Same correlation_id = "corr-12345"
  Idempotent step checks: have we seen this before?
  Yes → silent drop (deduplicated)
  No → process normally

Result: No duplicate processing
```

**Implementation:**
```go
// IdempotentStep checks if message was already processed
type IdempotentStep struct {
    keyExpr string
    seen    map[string]bool
}

func (is *IdempotentStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
    // Extract key from message
    key, err := is.extractKey(msg)
    if err != nil {
        return nil, err
    }

    // Check if seen
    if is.seen[key] {
        return nil, &SilentDropError{
            Key: key,
            Reason: "duplicate (idempotent dedup)",
        }
    }

    // Mark as seen
    is.seen[key] = true
    return msg, nil
}
```

**Test Cases:**
- Original message processed (idempotent registers key)
- Replayed message detected as duplicate (silent drop)
- Different message key (processed normally)
- Replay with no idempotent step (processes, may duplicate)

**Exit Criteria:**
✅ Idempotent step prevents duplicate processing  
✅ Replayed messages safely deduplicated  
✅ Different messages processed normally  
✅ Safety verified by tests  

### M1.8.3: Replay Audit Trail

**Objective:** Record replay operations in lineage

**Lineage Enhancement:**
```
Original message lineage:
  correlation_id: corr-12345
  route_version: v1
  stages: [source → filter → authorize → sink]
  status: ERROR (DLQ'd after authorize)

Replayed message lineage:
  correlation_id: corr-12345
  route_version: v2 (current)
  stages: [replay-source → filter → authorize → sink]
  replay_source_ref: {
    original_correlation_id: corr-12345
    original_route_version: v1
    replay_timestamp: 2026-09-04T12:34:56Z
    replay_count: 1
  }
```

**Implementation:**
```go
// ReplayLineageEntry records replay metadata
type ReplayLineageEntry struct {
    OriginalCorrelationID string
    OriginalRouteVersion  string
    ReplayTimestamp       time.Time
    ReplayAttempt         int
    ReplayInitiator       string // User or automation
}

func (msg *engine.Message) RecordReplay(entry *ReplayLineageEntry) {
    msg.Metadata.ReplayHistory = append(msg.Metadata.ReplayHistory, entry)
    msg.Metadata.ReplayCount++
}
```

**Exit Criteria:**
✅ Replay recorded in message lineage  
✅ Original message reference preserved  
✅ Replay timestamp and count tracked  
✅ Audit trail queryable via `dimctl lineage`  

## Configuration

### Replay Configuration in Route YAML

```yaml
routes:
  process-orders:
    from: webhook-in
    error_path:
      target: orders-dlq
      retry:
        max_attempts: 3
        backoff_ms: 500
    
    steps:
      # Idempotent step ensures replayed messages aren't re-processed
      - idempotent:
          key_expr: body.order_id
      
      - authorize:
          mode: rbac
          require_roles: [seller]
          on_deny:
            target: unauthorized-dlq
```

### Docker Compose for Local Testing

```yaml
# docker-compose.replay.yml
services:
  dim:
    image: dim:latest
    ports:
      - "8080:8080"  # Main server
      - "8081:8081"  # Observability viewer
    volumes:
      - ./examples/order-processing:/config
      - ./output:/output  # DLQ files
```

## Testing Strategy

### Unit Tests (M1.8.1)
- Replay command parsing
- Filter expression evaluation
- Dry-run preview accuracy
- Parallel execution

### Integration Tests (M1.8.2-3)
- End-to-end replay with idempotent dedup
- Lineage recording verification
- DLQ message recovery
- Audit trail queries

### End-to-End Test Scenario

```bash
# 1. Run route with failures (create DLQ)
dimctl run examples/order-processing/order-processing.yaml

# 2. Send message that will fail authorization
curl -X POST http://localhost:8080/orders \
  -H "Authorization: Bearer <jwt-viewer>" \
  -d '{"order_id": "123", ...}'
# Message goes to unauthorized-dlq

# 3. Preview replay
dimctl replay --from unauthorized-dlq \
  --filter 'error_type == "authorization_denied"' \
  --dry-run process-orders
# Output: 1 message would be replayed

# 4. Replay with corrected auth
curl -X POST http://localhost:8080/orders \
  -H "Authorization: Bearer <jwt-seller>" \
  -d '{"order_id": "123", ...}'
# Message re-enters route, processes normally

# 5. Verify audit trail
dimctl lineage query --correlation-id <corr-id>
# Output shows original failure + replay recovery
```

## Exit Criteria

### M1.8.1 ✅
- `dimctl replay` command implemented
- Filters DLQ messages by expression
- Dry-run preview works
- Parallel replay supported
- Result summary accurate

### M1.8.2 ✅
- Idempotent step prevents duplicates
- Replayed messages safely deduplicated
- Different messages processed normally
- Safety verified by tests

### M1.8.3 ✅
- Replay recorded in lineage
- Original message reference preserved
- Replay timestamp and count tracked
- Audit trail queryable

## References

- **Design §7.3:** Replay tooling (design/eip-middleware-design.md §7.3)
- **Idempotent Step:** internal/steps/idempotent.go
- **Lineage Store:** internal/lineage/store.go
- **DLQ Pattern:** Error handling (error_path in route config)
