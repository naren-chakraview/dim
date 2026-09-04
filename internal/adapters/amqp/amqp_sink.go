package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPSink is an AMQP producer sink adapter that writes messages to an AMQP broker
type AMQPSink struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	config    SinkConfig
	closed    bool
	mu        sync.Mutex
	metrics   struct {
		published int64
		failed    int64
		bytes     int64
		lastError error
	}
}

// SinkConfig defines configuration for an AMQP producer sink
type SinkConfig struct {
	URL            string        // AMQP connection URL
	ExchangeName   string        // Exchange to publish to
	ExchangeType   string        // Exchange type: "direct", "fanout", "topic", "headers"
	RoutingKey     string        // Default routing key
	Durable        bool          // Exchange durability
	AutoDelete     bool          // Exchange auto-delete
	Mandatory      bool          // Fail if no queue bound
	Immediate      bool          // Fail if no consumer
	DeliveryMode   uint8         // 1=Transient, 2=Persistent (default)
	ContentType    string        // Content type (default: "application/json")
	ConnectionName string        // Connection name for logging
}

// NewAMQPSink creates a new AMQP producer sink
func NewAMQPSink(url string, exchangeName string) (*AMQPSink, error) {
	return NewAMQPSinkWithConfig(SinkConfig{
		URL:            url,
		ExchangeName:   exchangeName,
		ExchangeType:   "topic",
		Durable:        true,
		AutoDelete:     false,
		DeliveryMode:   amqp.Persistent,
		ContentType:    "application/json",
		ConnectionName: "dim-amqp-sink",
	})
}

// NewAMQPSinkWithConfig creates an AMQP producer sink with custom configuration
func NewAMQPSinkWithConfig(config SinkConfig) (*AMQPSink, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("AMQP URL cannot be empty")
	}

	if config.ExchangeName == "" {
		return nil, fmt.Errorf("exchange name cannot be empty")
	}

	// Connect to AMQP broker
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AMQP broker: %w", err)
	}

	// Create channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Declare exchange
	err = channel.ExchangeDeclare(
		config.ExchangeName,
		config.ExchangeType,
		config.Durable,
		config.AutoDelete,
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	sink := &AMQPSink{
		conn:    conn,
		channel: channel,
		config:  config,
	}

	log.Printf("[INFO] AMQP sink created: url=%s exchange=%s type=%s", config.URL, config.ExchangeName, config.ExchangeType)

	return sink, nil
}

// Write writes messages to AMQP
// Implements the sink interface: Write(ctx context.Context, messages ...*engine.Message) error
func (as *AMQPSink) Write(ctx context.Context, messages ...*engine.Message) error {
	as.mu.Lock()
	if as.closed {
		as.mu.Unlock()
		return fmt.Errorf("sink is closed")
	}
	as.mu.Unlock()

	if len(messages) == 0 {
		return nil
	}

	for _, msg := range messages {
		// Serialize body to JSON
		var body []byte
		var err error

		switch v := msg.Body.(type) {
		case string:
			body = []byte(v)
		case []byte:
			body = v
		default:
			body, err = json.Marshal(msg.Body)
			if err != nil {
				as.recordError(err)
				return fmt.Errorf("failed to marshal message: %w", err)
			}
		}

		// Extract routing key from headers (if present)
		routingKey := as.config.RoutingKey
		if rkVal, ok := msg.Headers["amqp_routing_key"]; ok {
			if rkStr, isStr := rkVal.(string); isStr {
				routingKey = rkStr
			}
		}

		// Extract AMQP headers from message headers
		headers := make(amqp.Table)
		for headerKey, headerVal := range msg.Headers {
			if headerKey == "amqp_routing_key" || headerKey == "amqp_exchange" ||
			   headerKey == "amqp_delivery_tag" {
				continue // Skip AMQP metadata
			}
			headers[headerKey] = headerVal
		}

		// Create AMQP publishing object
		publishing := amqp.Publishing{
			ContentType:   as.config.ContentType,
			DeliveryMode:  as.config.DeliveryMode,
			Body:          body,
			Headers:       headers,
			Timestamp:     time.Now(),
			MessageId:     fmt.Sprintf("%v", msg.Headers["correlation_id"]),
			CorrelationId: fmt.Sprintf("%v", msg.Headers["correlation_id"]),
		}

		// Publish message
		err = as.channel.PublishWithContext(ctx,
			as.config.ExchangeName,
			routingKey,
			as.config.Mandatory,
			as.config.Immediate,
			publishing,
		)
		if err != nil {
			as.recordError(err)
			return fmt.Errorf("failed to publish message: %w", err)
		}

		// Update metrics
		as.mu.Lock()
		as.metrics.published++
		as.metrics.bytes += int64(len(body))
		as.mu.Unlock()
	}

	return nil
}

// Start is a no-op for sinks (they don't have an active listener)
func (as *AMQPSink) Start(ctx context.Context) error {
	return nil
}

// Close closes the AMQP sink and releases resources
func (as *AMQPSink) Close() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.closed {
		return nil
	}

	as.closed = true
	as.channel.Close()
	return as.conn.Close()
}

// recordError records an error for metrics
func (as *AMQPSink) recordError(err error) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.metrics.failed++
	as.metrics.lastError = err
}

// GetMetrics returns sink metrics for monitoring
func (as *AMQPSink) GetMetrics() map[string]interface{} {
	as.mu.Lock()
	defer as.mu.Unlock()

	return map[string]interface{}{
		"messages_published": as.metrics.published,
		"messages_failed":    as.metrics.failed,
		"bytes_published":    as.metrics.bytes,
		"last_error":         as.metrics.lastError,
	}
}
