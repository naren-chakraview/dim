package amqp

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestAMQPSourceBasic tests basic AMQP source consumption
// NOTE: Requires a running AMQP broker (RabbitMQ) at localhost:5672
// Skipped if broker is not available
func TestAMQPSourceBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	url := "amqp://guest:guest@localhost:5672/"
	queueName := "test-queue-" + time.Now().Format("20060102150405")

	outChan := engine.NewChannel("amqp-test", 10)
	source, err := NewAMQPSource(url, queueName, outChan)
	if err != nil {
		t.Skipf("AMQP broker not available: %v", err)
	}
	defer source.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start source in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- source.Start(ctx)
	}()

	// Create sink and publish a test message
	sink, err := NewAMQPSink(url, "test-exchange")
	if err != nil {
		t.Skipf("AMQP broker not available: %v", err)
	}
	defer sink.Close()

	// Publish test message
	testMsg := engine.NewMessage(`{"test": "data"}`, "test", "v1")
	if err := sink.Write(ctx, testMsg); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	// Consume message from source
	receivedMsg, err := outChan.Recv(ctx)
	if err != nil {
		t.Fatalf("failed to receive message: %v", err)
	}

	if receivedMsg == nil {
		t.Fatal("received nil message")
	}

	// Verify message content
	if receivedMsg.Body != `{"test": "data"}` {
		t.Errorf("message body mismatch: got %v, want %v", receivedMsg.Body, `{"test": "data"}`)
	}

	cancel()
	<-errChan
}

// TestAMQPSinkMetrics tests sink metrics collection
func TestAMQPSinkMetrics(t *testing.T) {
	sink, err := NewAMQPSinkWithConfig(SinkConfig{
		URL:          "amqp://guest:guest@localhost:5672/",
		ExchangeName: "metrics-test-exchange",
		ExchangeType: "direct",
	})
	if err != nil {
		t.Skipf("AMQP broker not available: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to write a message
	msg := engine.NewMessage(`{"key": "value"}`, "test", "v1")
	sink.Write(ctx, msg)

	// Get metrics
	metrics := sink.GetMetrics()
	if metrics == nil {
		t.Fatal("metrics are nil")
	}

	// Verify metric fields exist
	if _, ok := metrics["messages_published"]; !ok {
		t.Errorf("messages_published metric missing")
	}
	if _, ok := metrics["messages_failed"]; !ok {
		t.Errorf("messages_failed metric missing")
	}

	cancel()
}

// TestAMQPSourceClose tests graceful shutdown
func TestAMQPSourceClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	url := "amqp://guest:guest@localhost:5672/"
	queueName := "close-test-queue-" + time.Now().Format("20060102150405")

	outChan := engine.NewChannel("amqp-close-test", 10)
	source, err := NewAMQPSource(url, queueName, outChan)
	if err != nil {
		t.Skipf("AMQP broker not available: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start source
	errChan := make(chan error, 1)
	go func() {
		errChan <- source.Start(ctx)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for source to finish
	select {
	case err := <-errChan:
		if err != context.Canceled {
			t.Logf("source stopped with: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("source did not stop after context cancellation")
	}

	// Close should not panic
	if err := source.Close(); err != nil {
		t.Logf("close error (may be expected): %v", err)
	}
}

// BenchmarkAMQPMessageConversion benchmarks message conversion performance
func BenchmarkAMQPMessageConversion(b *testing.B) {
	sink, err := NewAMQPSink("amqp://guest:guest@localhost:5672/", "bench-exchange")
	if err != nil {
		b.Skipf("AMQP broker not available: %v", err)
	}
	defer sink.Close()

	msg := engine.NewMessage(`{"id": "test-123", "data": "benchmark"}`, "bench", "v1")
	msg.Headers["amqp_routing_key"] = "bench.key"
	msg.Headers["custom_header"] = "value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sink.Write(context.Background(), msg)
	}
}
