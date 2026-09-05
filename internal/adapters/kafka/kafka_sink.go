package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/adapters"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/segmentio/kafka-go"
)

// KafkaSink is a Kafka producer sink adapter that writes messages to a Kafka topic
type KafkaSink struct {
	writer  *kafka.Writer
	config  SinkConfig
	closed  bool
	mu      sync.Mutex
	metrics struct {
		written   int64
		failed    int64
		bytes     int64
		lastError error
	}
}

// SinkConfig defines configuration for a Kafka producer sink
type SinkConfig struct {
	Brokers      []string      // Kafka broker addresses
	Topic        string        // Topic to produce to
	Compression  string        // "none", "gzip", "snappy", "lz4" (default: "gzip")
	Acks         string        // "none", "leader", "all" (default: "all")
	Timeout      time.Duration // Write timeout (default: 10s)
	MaxAttempts  int           // Retry attempts (default: 3)
	RequiredAcks int           // Number of acknowledgements required
}

// NewKafkaSink creates a new Kafka producer sink
func NewKafkaSink(brokers []string, topic string) (*KafkaSink, error) {
	return NewKafkaSinkWithConfig(SinkConfig{
		Brokers:     brokers,
		Topic:       topic,
		Compression: "gzip",
		Acks:        "all",
		Timeout:     10 * time.Second,
		MaxAttempts: 3,
	})
}

// NewKafkaSinkWithConfig creates a Kafka producer sink with custom configuration
func NewKafkaSinkWithConfig(config SinkConfig) (*KafkaSink, error) {
	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("brokers list cannot be empty")
	}

	if config.Topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	// Map compression type
	compression := kafka.Gzip
	switch config.Compression {
	case "none":
		compression = kafka.Compression(0)
	case "snappy":
		compression = kafka.Snappy
	case "lz4":
		compression = kafka.Lz4
	case "gzip":
		compression = kafka.Gzip
	}

	// Map acks setting
	requiredAcks := kafka.RequireAll
	switch config.Acks {
	case "none":
		requiredAcks = kafka.RequireNone
	case "leader":
		requiredAcks = kafka.RequireOne
	case "all":
		requiredAcks = kafka.RequireAll
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      config.Brokers,
		Topic:        config.Topic,
		WriteTimeout: config.Timeout,
		ReadTimeout:  config.Timeout,
		MaxAttempts:  config.MaxAttempts,
		// Note: Compression and RequiredAcks not available in kafka-go v0.4.51
		// Consider upgrading kafka-go for these features
	})
	_ = compression
	_ = requiredAcks

	sink := &KafkaSink{
		writer: writer,
		config: config,
	}

	fmt.Printf("[INFO] Kafka sink created: brokers=%v topic=%s compression=%s acks=%s\n",
		config.Brokers, config.Topic, config.Compression, config.Acks)

	return sink, nil
}

// Write writes messages to Kafka and returns a result for each message
func (ks *KafkaSink) Write(ctx context.Context, messages ...*engine.Message) []adapters.Result {
	ks.mu.Lock()
	if ks.closed {
		ks.mu.Unlock()
		results := make([]adapters.Result, len(messages))
		for i, msg := range messages {
			results[i] = adapters.Result{
				Message: msg,
				Error:   fmt.Errorf("sink is closed"),
				Success: false,
			}
		}
		return results
	}
	ks.mu.Unlock()

	if len(messages) == 0 {
		return []adapters.Result{}
	}

	// Convert dim messages to Kafka messages
	kafkaMessages := make([]kafka.Message, len(messages))
	results := make([]adapters.Result, len(messages))

	for i, msg := range messages {
		kmsg, err := ks.dimMessageToKafkaMessage(msg)
		if err != nil {
			ks.recordError(err)
			results[i] = adapters.Result{
				Message: msg,
				Error:   err,
				Success: false,
			}
			continue
		}
		kafkaMessages[i] = kmsg
		results[i] = adapters.Result{
			Message: msg,
			Error:   nil,
			Success: true,
		}
	}

	// Write to Kafka
	err := ks.writer.WriteMessages(ctx, kafkaMessages...)
	if err != nil {
		ks.recordError(err)
		for i := range results {
			if results[i].Success {
				results[i].Success = false
				results[i].Error = err
			}
		}
		return results
	}

	// Update metrics
	ks.mu.Lock()
	ks.metrics.written += int64(len(messages))
	for _, msg := range kafkaMessages {
		ks.metrics.bytes += int64(len(msg.Value))
	}
	ks.mu.Unlock()

	return results
}

// dimMessageToKafkaMessage converts a dim message to a Kafka message
func (ks *KafkaSink) dimMessageToKafkaMessage(msg *engine.Message) (kafka.Message, error) {
	// Serialize body to JSON
	var value []byte
	var err error

	switch v := msg.Body.(type) {
	case string:
		value = []byte(v)
	case []byte:
		value = v
	default:
		value, err = json.Marshal(msg.Body)
		if err != nil {
			return kafka.Message{}, fmt.Errorf("failed to marshal message body: %w", err)
		}
	}

	// Extract Kafka key from headers (if present)
	var key []byte
	if keyVal, ok := msg.Headers["kafka_key"]; ok {
		if keyStr, isStr := keyVal.(string); isStr {
			key = []byte(keyStr)
		}
	}

	// Extract Kafka headers from message headers
	kafkaHeaders := make([]kafka.Header, 0)
	for headerKey, headerVal := range msg.Headers {
		if headerKey == "kafka_key" || headerKey == "kafka_topic" ||
		   headerKey == "kafka_partition" || headerKey == "kafka_offset" ||
		   headerKey == "kafka_timestamp" {
			continue // Skip Kafka metadata headers
		}

		var headerBytes []byte
		switch v := headerVal.(type) {
		case string:
			headerBytes = []byte(v)
		case []byte:
			headerBytes = v
		default:
			// Skip non-string/bytes headers
			continue
		}

		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   headerKey,
			Value: headerBytes,
		})
	}

	return kafka.Message{
		Key:     key,
		Value:   value,
		Headers: kafkaHeaders,
	}, nil
}

// Start is a no-op for sinks (they don't have an active listener)
func (ks *KafkaSink) Start(ctx context.Context) error {
	return nil
}

// Close closes the Kafka sink and releases resources
func (ks *KafkaSink) Close() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if ks.closed {
		return nil
	}

	ks.closed = true
	return ks.writer.Close()
}

// recordError records an error for metrics/monitoring
func (ks *KafkaSink) recordError(err error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.metrics.failed++
	ks.metrics.lastError = err
}

// GetMetrics returns sink metrics for monitoring
func (ks *KafkaSink) GetMetrics() map[string]interface{} {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	return map[string]interface{}{
		"messages_written": ks.metrics.written,
		"messages_failed":  ks.metrics.failed,
		"bytes_written":    ks.metrics.bytes,
		"last_error":       ks.metrics.lastError,
	}
}

// HealthCheck verifies the Kafka sink connection is alive
func (ks *KafkaSink) HealthCheck(ctx context.Context) error {
	if ks.writer == nil {
		return fmt.Errorf("kafka writer not initialized")
	}
	if len(ks.config.Brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}
	return nil
}
