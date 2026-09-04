# Kafka Adapter (Phase 1 — Track B)

This document describes the Kafka adapter implementation for Phase 1.

## Overview

The Kafka adapter provides production-grade Kafka integration for the dim middleware:

- **KafkaSource**: Consumes messages from Kafka topics via consumer groups
- **KafkaSink**: Produces messages to Kafka topics
- Distributed consumption with consumer group management
- Automatic offset tracking and committing
- Header and key preservation through dim message lifecycle
- Metrics collection for monitoring

## Implementation

### KafkaSource

```go
source, err := NewKafkaSource(
    []string{"localhost:9092"},  // Brokers
    "orders",                     // Topic
    "order-service",              // Consumer group
    outChan,                       // Output channel
)
defer source.Close()

err = source.Start(ctx)  // Blocks until context cancelled
```

**Features:**
- Consumer group based distribution (multiple instances can consume from same topic)
- Automatic offset management and committing
- JSON parsing (auto-detects JSON, falls back to string)
- Kafka metadata preservation in message headers
  - `kafka_topic`, `kafka_partition`, `kafka_offset`
  - `kafka_timestamp`, `kafka_key`
  - Custom headers as `kafka_header_*`
- Lag tracking via `GetLag(ctx)`
- Graceful shutdown with context cancellation

### KafkaSink

```go
sink, err := NewKafkaSink(
    []string{"localhost:9092"},  // Brokers
    "processed-orders",           // Topic
)
defer sink.Close()

err = sink.Write(ctx, messages...)  // Produces messages
```

**Features:**
- Configurable compression (gzip, snappy, lz4, none)
- Configurable ACKs (none, leader, all)
- JSON serialization of message bodies
- Kafka key extraction from headers
- Metrics collection: messages written, failed, bytes, last error
- Retry configuration

## Configuration

### SourceConfig

```go
config := SourceConfig{
    Brokers:          []string{"localhost:9092"},
    Topic:            "orders",
    GroupID:          "order-processor",
    StartOffset:      -1,  // Start from latest
    MaxBytes:         1024 * 1024,  // 1MB batches
    CommitInterval:   1 * time.Second,
    SessionTimeout:   10 * time.Second,
    RebalanceTimeout: 60 * time.Second,
}

source, err := NewKafkaSourceWithConfig(config, outChan)
```

### SinkConfig

```go
config := SinkConfig{
    Brokers:     []string{"localhost:9092"},
    Topic:       "orders-out",
    Compression: "gzip",
    Acks:        "all",        // Wait for all replicas
    Timeout:     10 * time.Second,
    MaxAttempts: 3,
}

sink, err := NewKafkaSinkWithConfig(config)
```

## Message Mapping

### dim Message → Kafka Message

- **Body** → Value (JSON serialized if not already string/bytes)
- **Headers** → Kafka headers (with `kafka_header_` prefix for custom headers)
- **kafka_key header** (if present) → Message key

### Kafka Message → dim Message

- **Value** → Body (JSON-parsed if valid JSON, else string)
- **Topic, Partition, Offset, Timestamp, Key** → Headers with `kafka_*` prefix
- **Custom headers** → Headers with `kafka_header_` prefix

## Usage Example

```yaml
# config.yaml
sources:
  kafka-in:
    type: kafka
    brokers: ["kafka1:9092", "kafka2:9092"]
    topic: orders
    group_id: dim-order-processor

sinks:
  kafka-valid:
    type: kafka
    brokers: ["kafka1:9092", "kafka2:9092"]
    topic: orders-valid

  kafka-dlq:
    type: kafka
    brokers: ["kafka1:9092", "kafka2:9092"]
    topic: orders-dlq

routes:
  order-processing:
    from: kafka-in
    error_path:
      target: kafka-dlq
    steps:
      - filter: { expr: 'body.amount > 0' }
      - translate: { expr: '{ order_id: body.id, amount: body.amount }' }
```

Run:
```bash
./dimctl run config.yaml
```

Messages from Kafka are consumed, processed through the route, and produced to output topics.

## Dependencies

The Kafka adapter uses:
- **segmentio/kafka-go** — Pure Go Kafka client (preferred over Shopify/sarama for simplicity)
- Alternative: **Shopify/sarama** (if preferred; heavier but more feature-rich)

Add to go.mod:
```bash
go get github.com/segmentio/kafka-go@latest
```

## Testing

### Unit Tests
```bash
go test ./internal/adapters/kafka -v
```

### Integration Tests (requires Kafka running)
```bash
# Start Kafka (Docker)
docker-compose up -d kafka

# Run tests
go test -run TestKafka ./internal/adapters/kafka -v -count=1

# Stop Kafka
docker-compose down
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  zookeeper:
    image: confluentinc/cp-zookeeper:7.0.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

  kafka:
    image: confluentinc/cp-kafka:7.0.0
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
```

## Performance

- **Throughput**: Kafka's own throughput limits (typically 100k+msg/sec per broker)
- **Latency**: Sub-second message delivery with auto-commit every 1s
- **Scalability**: Horizontal scaling via consumer groups

## Monitoring

### Metrics (per sink)

```go
metrics := sink.GetMetrics()
// {
//   "messages_written": 1000,
//   "messages_failed": 5,
//   "bytes_written": 2048576,
//   "last_error": error
// }
```

### Lag (per source)

```go
lag, err := source.GetLag(ctx)
// map[int32]int64{
//   0: 100,  // Partition 0 is 100 messages behind
//   1: 250,  // Partition 1 is 250 messages behind
// }
```

## Comparison with Phase 0

| Feature | Phase 0 | Phase 1 (Kafka) |
|---------|---------|-----------------|
| Message sources | HTTP, local file, SFTP polling | +Kafka (distributed, durable) |
| Message sinks | File, HTTP | +Kafka |
| Consumer groups | N/A | Yes (horizontal scaling) |
| Offset management | N/A | Automatic with commit tracking |
| Durability | Depends on sink | Kafka cluster replication |

## Phase 1+ Roadmap

### Immediate (Phase 1)
- ✅ KafkaSource implementation
- ✅ KafkaSink implementation
- [ ] Add dependencies to go.mod (go get github.com/segmentio/kafka-go)
- [ ] Full integration test with real Kafka broker
- [ ] Example route using Kafka

### Phase 1 Follow-up
- AMQP adapter (RabbitMQ, etc.)
- S3 adapter for cloud object storage
- Database adapters (Postgres CDC, etc.)
- Replay tooling (rewind offsets, replay from DLQ)

### Phase 2+
- Kafka Streams-like aggregations
- Multi-partition correlation-based ordering
- Schema Registry integration (Confluent)

## References

- Implementation: `internal/adapters/kafka/`
- Tests: `internal/adapters/kafka/*_test.go`
- segmentio/kafka-go: https://pkg.go.dev/github.com/segmentio/kafka-go
- Kafka documentation: https://kafka.apache.org/documentation/
