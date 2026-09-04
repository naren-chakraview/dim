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

// AMQPSource is an AMQP consumer source adapter that reads messages from an AMQP broker
// (RabbitMQ, etc.) and sends them to an output channel.
type AMQPSource struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	queue        amqp.Queue
	deliveries   <-chan amqp.Delivery
	outChan      *engine.Channel
	closed       chan struct{}
	wg            sync.WaitGroup
	config       SourceConfig
	messageCount int64
	mu           sync.Mutex
}

// SourceConfig defines configuration for an AMQP consumer source
type SourceConfig struct {
	URL             string        // AMQP connection URL (e.g., "amqp://guest:guest@localhost:5672/")
	QueueName       string        // Queue to consume from
	ExchangeName    string        // Exchange to bind to (optional)
	ExchangeType    string        // Exchange type: "direct", "fanout", "topic", "headers"
	RoutingKey      string        // Routing key for bindings
	Durable         bool          // Queue durability
	AutoDelete      bool          // Auto-delete when no consumers
	AutoAck         bool          // Auto-acknowledge messages
	PrefetchCount   int           // QoS prefetch count
	ReconnectDelay  time.Duration // Delay before reconnection attempt
	ConnectionName  string        // Connection name for logging
}

// NewAMQPSource creates a new AMQP consumer source
func NewAMQPSource(url string, queueName string, outChan *engine.Channel) (*AMQPSource, error) {
	return NewAMQPSourceWithConfig(SourceConfig{
		URL:            url,
		QueueName:      queueName,
		Durable:        true,
		AutoDelete:     false,
		AutoAck:        false, // Manual ack for reliability
		PrefetchCount:  10,
		ReconnectDelay: 5 * time.Second,
		ConnectionName: "dim-amqp-source",
	}, outChan)
}

// NewAMQPSourceWithConfig creates an AMQP consumer source with custom configuration
func NewAMQPSourceWithConfig(config SourceConfig, outChan *engine.Channel) (*AMQPSource, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}

	if config.URL == "" {
		return nil, fmt.Errorf("AMQP URL cannot be empty")
	}

	if config.QueueName == "" {
		return nil, fmt.Errorf("queue name cannot be empty")
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

	// Declare queue
	queue, err := channel.QueueDeclare(
		config.QueueName,
		config.Durable,
		config.AutoDelete,
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind to exchange if specified
	if config.ExchangeName != "" {
		err = channel.QueueBind(
			queue.Name,
			config.RoutingKey,
			config.ExchangeName,
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to bind queue: %w", err)
		}
	}

	// Set QoS
	err = channel.Qos(config.PrefetchCount, 0, false)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	// Consume messages
	deliveries, err := channel.Consume(
		queue.Name,
		"",                // consumer tag
		config.AutoAck,    // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set up consumer: %w", err)
	}

	source := &AMQPSource{
		conn:       conn,
		channel:    channel,
		queue:      queue,
		deliveries: deliveries,
		outChan:    outChan,
		closed:     make(chan struct{}),
		config:     config,
	}

	log.Printf("[INFO] AMQP source created: url=%s queue=%s exchange=%s", config.URL, config.QueueName, config.ExchangeName)

	return source, nil
}

// Start begins consuming messages from AMQP
// It blocks until the context is cancelled
func (as *AMQPSource) Start(ctx context.Context) error {
	as.wg.Add(1)
	defer as.wg.Done()

	for {
		select {
		case <-ctx.Done():
			as.channel.Close()
			as.conn.Close()
			close(as.closed)
			return ctx.Err()
		case delivery, ok := <-as.deliveries:
			if !ok {
				// Channel closed
				as.channel.Close()
				as.conn.Close()
				close(as.closed)
				return nil
			}

			// Convert AMQP delivery to dim message
			dimMsg, err := as.amqpDeliveryToDimMessage(delivery)
			if err != nil {
				log.Printf("[WARN] failed to convert AMQP message: %v", err)
				delivery.Nack(false, true) // Requeue on error
				continue
			}

			// Send to output channel
			if err := as.outChan.Send(ctx, dimMsg); err != nil {
				log.Printf("[WARN] failed to send message to output channel: %v", err)
				delivery.Nack(false, true) // Requeue on send error
				continue
			}

			// Acknowledge message on success
			if !as.config.AutoAck {
				delivery.Ack(false) // Single ack, not multiple
			}

			as.mu.Lock()
			as.messageCount++
			as.mu.Unlock()
		}
	}
}

// amqpDeliveryToDimMessage converts an AMQP delivery to a dim engine message
func (as *AMQPSource) amqpDeliveryToDimMessage(delivery amqp.Delivery) (*engine.Message, error) {
	// Parse body as JSON if possible, else use as string
	var body interface{}
	if err := json.Unmarshal(delivery.Body, &body); err != nil {
		// If not JSON, use the raw value as string
		body = string(delivery.Body)
	}

	msg := engine.NewMessage(body, "amqp-source", "")

	// Add AMQP-specific headers
	msg.Headers["amqp_exchange"] = delivery.Exchange
	msg.Headers["amqp_routing_key"] = delivery.RoutingKey
	msg.Headers["amqp_delivery_tag"] = delivery.DeliveryTag
	msg.Headers["amqp_redelivered"] = delivery.Redelivered
	msg.Headers["amqp_message_id"] = delivery.MessageId
	msg.Headers["amqp_correlation_id"] = delivery.CorrelationId
	msg.Headers["amqp_timestamp"] = delivery.Timestamp.Unix()

	// Add AMQP headers if present
	if delivery.Headers != nil {
		for key, value := range delivery.Headers {
			msg.Headers[fmt.Sprintf("amqp_header_%s", key)] = value
		}
	}

	return msg, nil
}

// GetMessageCount returns the number of messages consumed
func (as *AMQPSource) GetMessageCount() int64 {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.messageCount
}

// Close closes the AMQP source and releases resources
func (as *AMQPSource) Close() error {
	as.channel.Close()
	as.conn.Close()
	<-as.closed
	as.wg.Wait()
	return nil
}

// HealthCheck verifies the AMQP source connection is alive
func (as *AMQPSource) HealthCheck(ctx context.Context) error {
	if as.conn == nil || as.channel == nil {
		return fmt.Errorf("AMQP connection or channel not initialized")
	}
	if as.conn.IsClosed() {
		return fmt.Errorf("AMQP connection is closed")
	}
	return nil
}

// Checkpoint returns the current message count as a checkpoint
func (as *AMQPSource) Checkpoint() (interface{}, error) {
	as.mu.Lock()
	defer as.mu.Unlock()
	return map[string]interface{}{
		"messages_consumed": as.messageCount,
	}, nil
}
