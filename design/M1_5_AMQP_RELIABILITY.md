# M1.5: AMQP Adapter — Reliability & Hot-Reload Integration

**Phase:** Phase 1  
**Milestone:** M1.5 (AMQP adapter with reliability)  
**Status:** M1.5.1-3 Complete; M1.5.4 (hot-reload integration) — Implementation Plan  
**Date:** 2026-09-04

## Overview

M1.5 extends AMQP/RabbitMQ support with the same reliability and hot-reload integration bar as M1.4 (Kafka). AMQP sources and sinks now properly handle connection failures, graceful shutdown, and generation draining during hot-reload.

## Completed Components

### M1.5.1: AMQP Client Library Selection ✅
**Decision:** `github.com/rabbitmq/amqp091-go` (Official RabbitMQ Go client)

**Rationale:**
- Official RabbitMQ client (maintained by RabbitMQ core team)
- Pure Go (no cgo dependencies)
- Supports all AMQP features needed: exchanges, queues, manual ack, prefetch
- No dynamic version loading (safe for single binary)

**Already in go.mod:** github.com/rabbitmq/amqp091-go v1.9.0

### M1.5.2: AMQP Source Adapter ✅
**File:** `internal/adapters/amqp/amqp_source.go`

**Features:**
- Queue consumption with manual ack (configurable)
- Prefetch count (default: 10)
- Exchange binding (direct, fanout, topic, headers)
- Message deserialization (JSON → Go map)
- HealthCheck() validates connection and queue
- Checkpoint() returns last acked offset (for restart)

**Configuration:**
```yaml
sources:
  rabbitmq-in:
    type: amqp
    url: amqp://guest:guest@localhost:5672/
    queue_name: orders
    exchange_name: order-events
    routing_key: order.*
    auto_ack: false
    prefetch: 10
```

### M1.5.3: AMQP Sink Adapter ✅
**File:** `internal/adapters/amqp/amqp_sink.go`

**Features:**
- Message publishing to queue or exchange
- Configurable exchange type (direct, fanout, topic)
- Routing key per message
- Write() returns []Result (per M1.1 adapter spec)
- HealthCheck() validates connection
- Compression support (optional)

**Configuration:**
```yaml
sinks:
  rabbitmq-out:
    type: amqp
    url: amqp://guest:guest@localhost:5672/
    exchange_name: orders-processed
    routing_key: order.success
```

## M1.5.4: Hot-Reload Integration — Implementation Plan

**Objective:** AMQP adapters survive hot-reload without losing messages or unexpected duplication.

### Design Pattern

Same as Kafka (M1.4):

```
Old generation (Active → Draining):
  - Stops accepting new messages
  - In-flight messages complete with old version
  - Channel remains open for acking/nacking
  - Background drain monitor waits for in-flight → 0

New generation (Compiling → Active):
  - New source creates new AMQP connection
  - New sink gets new connection
  - Starts accepting messages immediately

Message delivery contract:
  - At-least-once by default (manual ack)
  - Duplicate detection: via idempotent step (if configured)
```

### Implementation Details

**1. Generation Lifecycle Tracking**

```go
// In internal/engine/generation.go, add AMQP-specific handling:

type GenerationState struct {
    // ... existing fields ...
    inFlightCount       int64        // Messages currently processing
    inFlightMu          sync.RWMutex
}

func (g *Generation) IncrementInFlight() {
    g.inFlightMu.Lock()
    g.inFlightCount++
    g.inFlightMu.Unlock()
}

func (g *Generation) DecrementInFlight() {
    g.inFlightMu.Lock()
    g.inFlightCount--
    count := g.inFlightCount
    g.inFlightMu.Unlock()
    
    if count == 0 {
        close(g.DoneCh) // Signal generation is complete
    }
}
```

**2. AMQP Source Drain Integration**

```go
// In AMQPSource:

type AMQPSource struct {
    // ... existing fields ...
    generation  *engine.Generation  // Reference to generation for tracking
    inFlightAck map[uint64]bool      // Track acked delivery tags
}

// When a message is consumed:
func (s *AMQPSource) consumeLoop(ctx context.Context) {
    for delivery := range s.deliveries {
        s.generation.IncrementInFlight()  // ← Mark in-flight
        
        msg := engine.NewMessage(delivery.Body, ...)
        msg.onCompletion = func(err error) {
            defer s.generation.DecrementInFlight()  // ← Mark done
            
            if err != nil {
                s.channel.Nack(delivery.DeliveryTag, false, true) // Requeue on error
            } else {
                s.channel.Ack(delivery.DeliveryTag, false)        // Ack on success
            }
        }
        
        select {
        case s.messageChan <- msg:
        case <-s.done:
            return
        }
    }
}
```

**3. Hot-Reload Draining**

```go
// When a new generation supersedes the old:

func (g *Generation) Drain(ctx context.Context) {
    // Stop accepting new messages
    close(g.StopAcceptingCh)
    
    // Wait for in-flight to reach zero (with timeout)
    timeout := time.AfterFunc(30*time.Second, func() {
        log.Printf("Generation drain timeout, forcing close after %.0f in-flight",
            g.inFlightCount)
    })
    
    select {
    case <-g.DoneCh:
        timeout.Stop()
        log.Println("Generation drained cleanly")
    case <-ctx.Done():
        timeout.Stop()
        return
    }
}
```

### Testing Strategy (M1.5.4)

**Integration Tests with `-race` flag:**

```bash
go test ./internal/adapters/amqp -race -timeout 30s
```

**Test scenarios:**

1. **Graceful reload mid-traffic**
   - Start AMQP source consuming
   - Send 10 messages
   - Trigger hot-reload (e.g., SIGHUP)
   - Verify all 10 messages processed
   - Verify no duplicates in output

2. **Reload with in-flight messages**
   - Source sends message to slow sink (100ms processing)
   - Trigger reload while message is in-flight
   - Verify message completes processing
   - Verify old generation drains before new one starts
   - New messages go to new generation

3. **Connection resilience**
   - Sink publishes, connection drops mid-send
   - Verify reconnection logic kicks in
   - Verify no data loss (retry mechanism)

4. **Concurrent draining generations**
   - Have 2 generations draining simultaneously
   - Verify cap on concurrent drains (default: 3)
   - Verify no resource exhaustion

### Code Files to Create

**New:**
- `internal/adapters/amqp/amqp_reliability_test.go` — M1.5.4 tests

**Modified:**
- `internal/adapters/amqp/amqp_source.go` — Add generation lifecycle hooks
- `internal/adapters/amqp/amqp_sink.go` — Add generation lifecycle hooks
- `internal/engine/generation.go` — Add in-flight tracking

## Configuration

### Source Example

```yaml
sources:
  rabbitmq-orders:
    type: amqp
    url: amqp://guest:guest@localhost:5672/
    queue_name: order-queue
    exchange_name: order-events
    routing_key: order.*
    auto_ack: false        # Manual ack for reliability
    prefetch: 10           # Prefetch count

routes:
  process-orders:
    from: rabbitmq-orders
    auth: none
    error_path:
      target: dlq-orders
      retry:
        max_attempts: 3
        backoff_ms: 500
    steps:
      - filter: body.total > 0
      - translate: |
          {
            "order_id": body.id,
            "total": body.total,
            "timestamp": $now()
          }
    
    sinks:
      - rabbitmq-orders-processed
```

### Sink Example

```yaml
sinks:
  rabbitmq-orders-processed:
    type: amqp
    url: amqp://guest:guest@localhost:5672/
    exchange_name: processed-orders
    routing_key: order.processed
```

## Docker Compose for Local Testing

```yaml
# deploy/docker-compose.amqp.yml

version: '3.8'
services:
  rabbitmq:
    image: rabbitmq:3.12-management
    ports:
      - "5672:5672"    # AMQP
      - "15672:15672"  # Management UI
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    healthcheck:
      test: rabbitmq-diagnostics -q ping
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - dim

networks:
  dim:
    name: dim-network
```

Run with:
```bash
docker-compose -f deploy/docker-compose.amqp.yml up
```

## Design Decisions

### Why Manual Ack by Default?
- **At-least-once semantics:** Ensures messages aren't lost if processor crashes
- **Tradeoff:** Possibility of duplicates (mitigated by idempotent step)
- **Configuration:** Teams can set `auto_ack: true` for higher throughput, lower reliability

### Why Prefetch Count?
- **Default 10:** Balance between throughput and memory
- **Tunable:** Can increase for bursty workloads, decrease for memory constraints
- **Rationale:** RabbitMQ pulls messages into client buffer (Prefetch)

### Generation Tracking via In-Flight Count
- **Explicit:** Clear semantics (counting messages being processed)
- **Testable:** Can verify count reaches zero
- **Race-safe:** Using atomic increment/decrement

## Exit Criteria

### M1.5.1 ✅
- Client library selected and pinned
- Rationale documented

### M1.5.2 ✅
- Source adapter consumes from queue
- Manual ack, prefetch configurable
- HealthCheck() and Checkpoint() working

### M1.5.3 ✅
- Sink adapter publishes to exchange/queue
- Write() returns []Result per spec
- HealthCheck() validates connection

### M1.5.4 (Ready to implement)
- Source tracks in-flight count
- Generation drain waits for in-flight → 0
- `-race` enabled integration tests pass
- Hot-reload mid-traffic tested
- Concurrent drains capped

## References

- **M1.4 (Kafka):** `design/` (similar hot-reload pattern)
- **Adapter Spec:** `internal/adapters/ADAPTER_SPEC.md`
- **RabbitMQ:** https://www.rabbitmq.com/
- **Client Docs:** https://pkg.go.dev/github.com/rabbitmq/amqp091-go
