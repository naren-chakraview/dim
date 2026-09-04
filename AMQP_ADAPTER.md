# AMQP Adapter (Phase 1 — Track B)

Production-grade AMQP (Advanced Message Queuing Protocol) integration for RabbitMQ and compatible brokers.

## Overview

- **AMQPSource**: Consumes from AMQP brokers with manual acknowledgment
- **AMQPSink**: Produces to AMQP brokers with configurable delivery
- Queue and exchange management
- Manual acknowledgment for reliability
- QoS prefetching and flow control
- Metrics collection and monitoring

## Implementation

### AMQPSource

```go
source, err := NewAMQPSource(
    "amqp://guest:guest@localhost:5672/",  // URL
    "orders",                                // Queue name
    outChan,                                 // Output channel
)
defer source.Close()

err = source.Start(ctx)  // Blocks until context cancelled
```

**Features:**
- Queue and exchange declaration
- Manual acknowledgment (default) or auto-ack (configurable)
- QoS prefetching (default: 10)
- Message requeue on error
- JSON auto-parsing
- Metadata headers (exchange, routing key, timestamp, etc.)
- Message counter for monitoring

### AMQPSink

```go
sink, err := NewAMQPSink(
    "amqp://guest:guest@localhost:5672/",  // URL
    "processed-orders",                      // Exchange name
)
defer sink.Close()

err = sink.Write(ctx, messages...)  // Produces to exchange
```

**Features:**
- Exchange declaration and binding
- Configurable delivery mode (transient/persistent)
- Routing key extraction from headers
- JSON serialization
- Metrics collection (published, failed, bytes)
- Mandatory/immediate publishing options

## Configuration

### SourceConfig

```go
config := SourceConfig{
    URL:            "amqp://guest:guest@localhost:5672/",
    QueueName:      "orders",
    ExchangeName:   "orders-exchange",  // Optional
    ExchangeType:   "topic",            // direct, fanout, topic, headers
    RoutingKey:     "order.*",
    Durable:        true,               // Queue survives broker restart
    AutoDelete:     false,
    AutoAck:        false,              // Manual ack for reliability
    PrefetchCount:  10,                 // QoS window
    ReconnectDelay: 5 * time.Second,
}

source, err := NewAMQPSourceWithConfig(config, outChan)
```

### SinkConfig

```go
config := SinkConfig{
    URL:           "amqp://guest:guest@localhost:5672/",
    ExchangeName:  "processed-orders",
    ExchangeType:  "topic",            // direct, fanout, topic, headers
    RoutingKey:    "order.processed",  // Default; can override per message
    Durable:       true,               // Exchange survives broker restart
    AutoDelete:    false,
    DeliveryMode:  amqp.Persistent,   // 1=Transient, 2=Persistent
    ContentType:   "application/json",
}

sink, err := NewAMQPSinkWithConfig(config)
```

## Message Mapping

### dim Message → AMQP Message

- **Body** → Publishing.Body (JSON serialized)
- **Headers** → Publishing.Headers (AMQP table)
- **amqp_routing_key header** → Routing key (overrides default)
- **correlation_id header** → MessageId & CorrelationId

### AMQP Message → dim Message

- **Body** → Body (JSON-parsed if valid, else string)
- **Exchange, RoutingKey, DeliveryTag** → Headers with `amqp_*` prefix
- **Timestamp, MessageId, CorrelationId** → Message headers
- **Redelivered flag** → Header for tracking retries
- **AMQP Headers** → Headers with `amqp_header_` prefix

## Usage Example

```yaml
# config.yaml
sources:
  amqp-orders:
    type: amqp
    url: amqp://guest:guest@rabbitmq:5672/
    queue_name: orders
    exchange_name: orders-exchange
    exchange_type: topic
    routing_key: "order.*"

sinks:
  amqp-valid:
    type: amqp
    url: amqp://guest:guest@rabbitmq:5672/
    exchange_name: orders-valid
    exchange_type: direct
    routing_key: valid

  amqp-dlq:
    type: amqp
    url: amqp://guest:guest@rabbitmq:5672/
    exchange_name: orders-dlq
    exchange_type: direct
    routing_key: dlq

routes:
  order-processing:
    from: amqp-orders
    error_path:
      target: amqp-dlq
    steps:
      - filter: { expr: 'body.amount > 0' }
      - translate: { expr: '{ order_id: body.id, total: body.amount }' }
```

Run:
```bash
./dimctl run config.yaml
```

## Dependencies

Uses **github.com/rabbitmq/amqp091-go** (official Go AMQP client).

Add to go.mod:
```bash
go get github.com/rabbitmq/amqp091-go@latest
```

## Testing

### Docker Compose

```yaml
version: '3.8'
services:
  rabbitmq:
    image: rabbitmq:3.12-management
    ports:
      - "5672:5672"      # AMQP port
      - "15672:15672"    # Management UI
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
```

Start: `docker-compose up -d`
Management UI: http://localhost:15672 (guest/guest)

### Running Tests

```bash
# Start RabbitMQ
docker-compose up -d

# Run integration tests
go test -run TestAMQP ./internal/adapters/amqp -v

# Stop RabbitMQ
docker-compose down
```

## Comparison: Kafka vs. AMQP

| Feature | Kafka | AMQP (RabbitMQ) |
|---------|-------|-----------------|
| Delivery guarantee | At-least-once | Exactly-once (per queue) |
| Scalability | Horizontal (partitions) | Vertical (queues) |
| Retention | Log-based, configurable | Queue-based, message lifecycle |
| Message order | Per partition | Per queue (FIFO) |
| Use case | Event streaming | Task queues, work distribution |
| Complexity | Higher throughput, complex offset mgmt | Simpler model, easier setup |

**Use Kafka when:** You need high throughput, distributed consumption, event streaming, replay capability.

**Use AMQP when:** You need reliable task distribution, simpler deployment, guaranteed delivery, routing flexibility.

## Monitoring

### Metrics (per sink)

```go
metrics := sink.GetMetrics()
// {
//   "messages_published": 1000,
//   "messages_failed": 5,
//   "bytes_published": 2048576,
//   "last_error": error
// }
```

### Message Count (per source)

```go
count := source.GetMessageCount()  // Total messages consumed
```

## Phase 1+ Roadmap

### Immediate
- ✅ AMQPSource implementation
- ✅ AMQPSink implementation
- [ ] Add dependencies to go.mod (go get github.com/rabbitmq/amqp091-go)
- [ ] Full integration test suite

### Phase 1+
- Alternative exchanges: fanout, headers types with advanced routing
- Consumer group patterns for work distribution
- Dead-letter exchange integration for DLQ routing
- Connection pooling for high-throughput scenarios

### Phase 2+
- Message compression (gzip, deflate)
- TLS/SASL authentication
- RabbitMQ clustering support
- Publisher confirms for guaranteed delivery

## References

- Implementation: `internal/adapters/amqp/`
- Tests: `internal/adapters/amqp/*_test.go`
- RabbitMQ client: https://pkg.go.dev/github.com/rabbitmq/amqp091-go
- RabbitMQ docs: https://www.rabbitmq.com/documentation.html
- AMQP spec: https://www.amqp.org/
