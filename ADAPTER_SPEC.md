# Adapter Specification

This document specifies the adapter interface contract for dim sources and sinks, matching §8 of the design document.

## Interface Definitions

All adapters must implement one of two interfaces defined in `internal/adapters/interfaces.go`:

### Source Interface

```go
type Source interface {
	Start(ctx context.Context) error
	HealthCheck(ctx context.Context) error
	Checkpoint() (interface{}, error)
	Close() error
}
```

**Start**: Begins consuming messages from the external system and sends them to the output channel. Blocks until context is canceled or error occurs.

**HealthCheck**: Verifies the source connection is alive. Returns nil on healthy connection, error otherwise.

**Checkpoint**: Returns the current offset/cursor position for resumption on reconnect. Format is adapter-specific (e.g., Kafka returns partition offset map, S3 returns last-sync timestamps).

**Close**: Closes the source and releases resources.

### Sink Interface

```go
type Sink interface {
	Write(ctx context.Context, messages ...*engine.Message) []Result
	HealthCheck(ctx context.Context) error
	Close() error
}
```

**Write**: Writes one or more messages to the external system. Returns a `Result` slice with one entry per input message, even if errors occur. Each Result contains:
- `Message`: the original message
- `Error`: error for this message (nil on success)
- `Success`: boolean indicating write success

**HealthCheck**: Verifies the sink connection is alive. Returns nil on healthy connection, error otherwise.

**Close**: Closes the sink and releases resources.

## Result Type

```go
type Result struct {
	Message *engine.Message
	Error   error
	Success bool
}
```

Each message write returns a Result allowing per-message error tracking and recovery.

## Implementation Pattern

All adapters follow this pattern:

### Message Conversion
1. Accept dim `engine.Message` with Body (string/[]byte/interface{}) and Headers map
2. Convert to protocol-native format (Kafka Message, AMQP Publishing, S3 PutObject, etc.)
3. Add protocol-specific metadata to dim message headers (e.g., `kafka_topic`, `amqp_routing_key`, `s3_key`)
4. Return individual Result per message

### Configuration
- `SourceConfig`: Per-adapter struct for source configuration
- `SinkConfig`: Per-adapter struct for sink configuration
- Constructor functions: `New<Name>Source()` and `New<Name>Sink()`
- Config validation in constructors

### Metrics
Each adapter tracks:
- Messages written/consumed
- Bytes transferred
- Error count
- Last error

Via `GetMetrics()` method returning `map[string]interface{}`.

### Error Handling
- Write() never panics; returns Result with error for each message
- HealthCheck() returns error without throwing
- Close() idempotent (safe to call multiple times)

## Current Implementations

**Kafka** (`internal/adapters/kafka/`)
- Source: Consumer groups with offset tracking per partition
- Sink: Producer with configurable compression/acks
- Checkpoint: Partition offset map `map[int32]int64`

**AMQP** (`internal/adapters/amqp/`)
- Source: Queue-based consumption with manual ack
- Sink: Exchange publisher with routing key extraction
- Checkpoint: Message count map

**S3** (`internal/adapters/s3/`)
- Source: Bucket polling with LastModified tracking
- Sink: Object writer with JSON serialization
- Checkpoint: Key -> Unix timestamp map

**File** (`internal/adapters/file/`)
- Source: Directory polling with modification-time tracking
- Sink: File writer to directory

## Message Metadata Headers

Adapters preserve protocol metadata in message headers for lineage and debugging:

**Kafka headers**: `kafka_topic`, `kafka_partition`, `kafka_offset`, `kafka_timestamp`, `kafka_key`, `kafka_header_*`

**AMQP headers**: `amqp_exchange`, `amqp_routing_key`, `amqp_delivery_tag`, `amqp_redelivered`, `amqp_timestamp`, `amqp_header_*`

**S3 headers**: `s3_bucket`, `s3_key`, `s3_size`

**File headers**: `file_path`, `file_size`, `file_modified`

## Testing

Each adapter includes:
- Unit tests for message conversion
- Integration tests (skipped if broker/service unavailable)
- Metrics collection tests
- Graceful shutdown tests
- Benchmarks for performance validation
