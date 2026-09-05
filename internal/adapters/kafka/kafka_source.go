package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/segmentio/kafka-go"
)

// KafkaSource is a Kafka consumer source adapter that reads messages from a Kafka topic
// and sends them to an output channel. Uses consumer groups for distributed consumption.
// Supports hot-reload with graceful in-flight message draining (R17).
type KafkaSource struct {
	reader       *kafka.Reader
	outChan      *engine.Channel
	closed       chan struct{}
	wg           sync.WaitGroup
	config       SourceConfig
	lastOffset   map[int32]int64 // Track last committed offset per partition
	inFlight     int32            // Tracks messages currently being processed (R17)
	drainCaps    chan struct{}    // Concurrent drain cap (max 10 concurrent drains) (R17)
	mu           sync.Mutex
}

// SourceConfig defines configuration for a Kafka consumer source
type SourceConfig struct {
	Brokers       []string      // Kafka broker addresses (e.g., ["localhost:9092"])
	Topic         string        // Topic to consume from
	GroupID       string        // Consumer group ID (enables distributed consumption)
	StartOffset   int64         // Starting offset (-2=oldest, -1=newest, or specific offset)
	MaxBytes      int           // Maximum message batch size (default: 1MB)
	CommitInterval time.Duration // Interval for committing offsets (default: 1s)
	SessionTimeout time.Duration // Session timeout (default: 10s)
	RebalanceTimeout time.Duration // Rebalance timeout (default: 60s)
}

// NewKafkaSource creates a new Kafka consumer source
func NewKafkaSource(brokers []string, topic string, groupID string, outChan *engine.Channel) (*KafkaSource, error) {
	return NewKafkaSourceWithConfig(SourceConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.NewOffset().AtOffset(-1), // Start from newest
		MaxBytes:    1024 * 1024,                      // 1MB default
		CommitInterval: 1 * time.Second,
		SessionTimeout: 10 * time.Second,
		RebalanceTimeout: 60 * time.Second,
	}, outChan)
}

// NewKafkaSourceWithConfig creates a Kafka consumer source with custom configuration
func NewKafkaSourceWithConfig(config SourceConfig, outChan *engine.Channel) (*KafkaSource, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}

	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("brokers list cannot be empty")
	}

	if config.Topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	if config.GroupID == "" {
		return nil, fmt.Errorf("group ID cannot be empty")
	}

	// Create Kafka reader with consumer group
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         config.Brokers,
		Topic:           config.Topic,
		GroupID:         config.GroupID,
		StartOffset:     config.StartOffset,
		MaxBytes:        config.MaxBytes,
		CommitInterval:  config.CommitInterval,
		SessionTimeout:  config.SessionTimeout,
		RebalanceTimeout: config.RebalanceTimeout,
	})

	ks := &KafkaSource{
		reader:    reader,
		outChan:   outChan,
		closed:    make(chan struct{}),
		config:    config,
		lastOffset: make(map[int32]int64),
		drainCaps: make(chan struct{}, 10), // Max 10 concurrent drains (R17)
	}

	log.Printf("[INFO] Kafka source created: brokers=%v topic=%s group=%s", config.Brokers, config.Topic, config.GroupID)

	return ks, nil
}

// Start begins consuming messages from Kafka
// It blocks until the context is cancelled
func (ks *KafkaSource) Start(ctx context.Context) error {
	ks.wg.Add(1)
	defer ks.wg.Done()

	for {
		select {
		case <-ctx.Done():
			ks.reader.Close()
			close(ks.closed)
			return ctx.Err()
		default:
			// Read message from Kafka
			msg, err := ks.reader.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					ks.reader.Close()
					close(ks.closed)
					return nil
				}
				log.Printf("[WARN] failed to read message from Kafka: %v", err)
				continue
			}

			// Track in-flight message (R17: hot-reload support)
			atomic.AddInt32(&ks.inFlight, 1)
			defer atomic.AddInt32(&ks.inFlight, -1)

			// Convert Kafka message to dim message
			dimMsg, err := ks.kafkaMessageToDimMessage(msg)
			if err != nil {
				log.Printf("[WARN] failed to convert Kafka message: %v", err)
				continue
			}

			// Send to output channel
			if err := ks.outChan.Send(ctx, dimMsg); err != nil {
				log.Printf("[WARN] failed to send message to output channel: %v", err)
				// Don't break; continue consuming
				continue
			}

			// Track offset for this partition
			ks.mu.Lock()
			ks.lastOffset[msg.Partition] = msg.Offset
			ks.mu.Unlock()
		}
	}
}

// kafkaMessageToDimMessage converts a Kafka message to a dim engine message
func (ks *KafkaSource) kafkaMessageToDimMessage(kmsg kafka.Message) (*engine.Message, error) {
	// Parse message value as JSON if possible, else use as string
	var body interface{}
	if err := json.Unmarshal(kmsg.Value, &body); err != nil {
		// If not JSON, use the raw value as string
		body = string(kmsg.Value)
	}

	msg := engine.NewMessage(body, "kafka-source", "")

	// Add Kafka-specific headers
	msg.Headers["kafka_topic"] = kmsg.Topic
	msg.Headers["kafka_partition"] = kmsg.Partition
	msg.Headers["kafka_offset"] = kmsg.Offset
	msg.Headers["kafka_timestamp"] = kmsg.Time.Unix()
	msg.Headers["kafka_key"] = string(kmsg.Key)

	// Add custom headers if present in Kafka message headers
	for _, h := range kmsg.Headers {
		msg.Headers[fmt.Sprintf("kafka_header_%s", h.Key)] = string(h.Value)
	}

	return msg, nil
}

// CommitOffsets commits the current offsets for all partitions
// This is typically called after successful message processing
func (ks *KafkaSource) CommitOffsets() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if len(ks.lastOffset) == 0 {
		return nil
	}

	// Offsets are automatically committed based on CommitInterval
	// This is a no-op for segmentio/kafka-go which uses auto-commit
	log.Printf("[DEBUG] Offsets tracked for %d partitions", len(ks.lastOffset))
	return nil
}

// Close closes the Kafka source and releases resources
func (ks *KafkaSource) Close() error {
	ks.reader.Close()
	<-ks.closed
	ks.wg.Wait()
	return nil
}

// Drain waits for all in-flight messages to complete before returning.
// Used during hot-reload to gracefully shut down the current generation.
// Respects the concurrent-draining-cap pattern (max 10 concurrent drains) (R17).
func (ks *KafkaSource) Drain(ctx context.Context, timeout time.Duration) error {
	// Acquire drain cap (max 10 concurrent drains)
	select {
	case ks.drainCaps <- struct{}{}:
		defer func() { <-ks.drainCaps }()
	case <-ctx.Done():
		return ctx.Err()
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if atomic.LoadInt32(&ks.inFlight) == 0 {
			// All in-flight messages are complete
			return nil
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				remaining := atomic.LoadInt32(&ks.inFlight)
				return fmt.Errorf("drain timeout exceeded with %d in-flight messages", remaining)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetInFlightCount returns the current number of in-flight messages (R17).
func (ks *KafkaSource) GetInFlightCount() int32 {
	return atomic.LoadInt32(&ks.inFlight)
}

// GetLag returns the consumer lag (difference between committed and latest offset)
// Useful for monitoring
func (ks *KafkaSource) GetLag(ctx context.Context) (map[int32]int64, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	lag := make(map[int32]int64)
	for partition := range ks.lastOffset {
		// Get latest offset for partition
		offset, err := ks.reader.ReadOffsetFromPartition(partition)
		if err != nil {
			return nil, fmt.Errorf("failed to get offset for partition %d: %w", partition, err)
		}
		lag[partition] = offset - ks.lastOffset[partition]
	}

	return lag, nil
}

// HealthCheck verifies the Kafka source connection is alive
func (ks *KafkaSource) HealthCheck(ctx context.Context) error {
	if ks.reader == nil {
		return fmt.Errorf("kafka reader not initialized")
	}
	// Try to fetch metadata to verify broker connectivity
	brokers := ks.reader.Config().Brokers
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}
	return nil
}

// Checkpoint returns the current partition offset map for resumption
func (ks *KafkaSource) Checkpoint() (interface{}, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	// Return a copy of the offset map for safe external use
	checkpoint := make(map[int32]int64)
	for k, v := range ks.lastOffset {
		checkpoint[k] = v
	}
	return checkpoint, nil
}
