package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestKafkaSourceBasic tests basic Kafka source consumption
// NOTE: Requires a running Kafka broker at localhost:9092
// Skipped if broker is not available
func TestKafkaSourceBasic(t *testing.T) {
	t.Skip("TODO: Kafka integration test requires broker - skip for CI without services")
	// Skip if Kafka not available (integration test)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	brokers := []string{"localhost:9092"}
	topic := "test-topic"
	groupID := "test-group-" + time.Now().Format("20060102150405")

	outChan := engine.NewChannel("kafka-test", 10)
	source, err := NewKafkaSource(brokers, topic, groupID, outChan)
	if err != nil {
		t.Skipf("Kafka broker not available: %v", err)
	}
	defer source.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start source in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- source.Start(ctx)
	}()

	// Create sink and produce a test message
	sink, err := NewKafkaSink(brokers, topic)
	if err != nil {
		t.Skipf("Kafka broker not available: %v", err)
	}
	defer sink.Close()

	// Produce test message
	testMsg := engine.NewMessage(`{"test": "data"}`, "test", "v1")
	if err := sink.Write(ctx, testMsg); err != nil {
		t.Fatalf("failed to produce message: %v", err)
	}

	// Consume message from source
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	receivedMsg, err := outChan.Recv(ctx2)
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

	// Verify Kafka headers
	if topic, ok := receivedMsg.Headers["kafka_topic"]; !ok || topic != topic {
		t.Errorf("kafka_topic header missing or incorrect")
	}

	cancel()
	<-errChan
}

// TestKafkaSinkMetrics tests sink metrics collection
func TestKafkaSinkMetrics(t *testing.T) {
	sink, err := NewKafkaSinkWithConfig(SinkConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       "metrics-test",
		Timeout:     5 * time.Second,
		MaxAttempts: 1,
	})
	if err != nil {
		t.Skipf("Kafka broker not available: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to write a message
	msg := engine.NewMessage(`{"key": "value"}`, "test", "v1")
	sink.Write(ctx, msg) // May fail if broker not available, but we're testing metrics collection

	// Get metrics
	metrics := sink.GetMetrics()
	if metrics == nil {
		t.Fatal("metrics are nil")
	}

	// Verify metric fields exist
	if _, ok := metrics["messages_written"]; !ok {
		t.Errorf("messages_written metric missing")
	}
	if _, ok := metrics["messages_failed"]; !ok {
		t.Errorf("messages_failed metric missing")
	}

	cancel()
}

// TestKafkaSourceClose tests graceful shutdown
func TestKafkaSourceClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	brokers := []string{"localhost:9092"}
	outChan := engine.NewChannel("kafka-close-test", 10)

	source, err := NewKafkaSource(brokers, "close-test-topic", "close-test-group", outChan)
	if err != nil {
		t.Skipf("Kafka broker not available: %v", err)
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

// BenchmarkKafkaMessageConversion benchmarks message conversion performance
func BenchmarkKafkaMessageConversion(b *testing.B) {
	sink, err := NewKafkaSink([]string{"localhost:9092"}, "bench-topic")
	if err != nil {
		b.Skipf("Kafka broker not available: %v", err)
	}
	defer sink.Close()

	msg := engine.NewMessage(`{"id": "test-123", "data": "benchmark"}`, "bench", "v1")
	msg.Headers["kafka_key"] = "key-123"
	msg.Headers["custom_header"] = "value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sink.dimMessageToKafkaMessage(msg)
		if err != nil {
			b.Fatalf("conversion failed: %v", err)
		}
	}
}
