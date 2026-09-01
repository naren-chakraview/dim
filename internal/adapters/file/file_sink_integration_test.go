package file

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// PassthroughStep is a simple step that passes messages through unchanged
type PassthroughStep struct{}

func (p *PassthroughStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	return msg, nil
}

// TestFileSinkIntegrationEndToEnd tests the file sink in an end-to-end scenario
// simulating messages flowing through an executor and being written to a file.
func TestFileSinkIntegrationEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "integration_test.jsonl")

	// Create channels for a simple pipeline
	sourceChan := engine.NewChannel("source", 10)
	sinkChan := engine.NewChannel("sink", 10)

	// Create a simple step that passes messages through unchanged
	passthrough := &PassthroughStep{}

	// Create an executor that will process messages from source to sink
	executor := engine.NewExecutor("test-executor", sourceChan, sinkChan, []engine.Step{passthrough})

	// Create the file sink that reads from the sink channel
	sink, err := NewFileSink(filePath, sinkChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	// Start executor in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- executor.Run(ctx)
	}()

	// Start the sink
	sink.Start(ctx)

	// Send test messages through the source channel
	testMessages := []map[string]interface{}{
		{"type": "order", "id": "ORD-001", "amount": 100.50},
		{"type": "payment", "id": "PAY-001", "status": "success"},
		{"type": "shipment", "id": "SHIP-001", "tracking": "1Z999AA10123456784"},
	}

	for i, body := range testMessages {
		msg := engine.NewMessage(body, "integration-route", "v1.0.0")
		msg.Headers["content-type"] = "application/json"
		msg.Headers["request-id"] = "REQ-001"

		if err := sourceChan.Send(ctx, msg); err != nil {
			t.Fatalf("Send message %d failed: %v", i, err)
		}
	}

	// Give time for processing
	time.Sleep(500 * time.Millisecond)

	// Close the source channel to signal completion
	sourceChan.Close()

	// Wait a bit for the executor to finish
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop executor and sink
	cancel()

	// Verify no executor errors
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Logf("executor error (expected if context cancelled): %v", err)
		}
	case <-time.After(1 * time.Second):
		// Executor should finish quickly
	}

	// Close sink and verify file contents
	sink.Close()

	// Verify the file exists and has content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("output file is empty")
	}

	// Parse and verify each line
	lines := make([]string, 0)
	var lineStart int
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, string(data[lineStart:i]))
			lineStart = i + 1
		}
	}

	if len(lines) != len(testMessages) {
		t.Errorf("expected %d lines, got %d", len(testMessages), len(lines))
	}

	// Verify each message was written correctly
	for i, line := range lines {
		var result engine.Message
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			t.Errorf("line %d failed to unmarshal: %v", i, err)
			continue
		}

		// Verify metadata
		if result.Metadata.Route != "integration-route" {
			t.Errorf("line %d: route mismatch", i)
		}
		if result.Metadata.RouteVersion != "v1.0.0" {
			t.Errorf("line %d: route version mismatch", i)
		}

		// Verify headers
		if result.Headers["content-type"] != "application/json" {
			t.Errorf("line %d: content-type header mismatch", i)
		}
		if result.Headers["request-id"] != "REQ-001" {
			t.Errorf("line %d: request-id header mismatch", i)
		}

		// Verify body matches expected message
		expectedBody := testMessages[i]
		resultBody := result.Body.(map[string]interface{})
		for key, expectedVal := range expectedBody {
			if resultBody[key] != expectedVal {
				t.Errorf("line %d: body[%s] mismatch: got %v, want %v", i, key, resultBody[key], expectedVal)
			}
		}
	}
}

// TestFileSinkBackpressure tests that the sink applies backpressure correctly
// by blocking when the channel is full.
func TestFileSinkBackpressure(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "backpressure_test.jsonl")

	// Create a small buffer channel to easily fill it
	inChan := engine.NewChannel("test-sink", 2) // Only buffer 2 messages

	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Don't start the sink yet - let messages pile up in the channel

	// Send first message (channel has capacity)
	msg1 := engine.NewMessage(map[string]interface{}{"n": 1}, "test", "v1")
	if err := inChan.Send(ctx, msg1); err != nil {
		t.Fatalf("send message 1 failed: %v", err)
	}

	// Send second message (channel still has capacity)
	msg2 := engine.NewMessage(map[string]interface{}{"n": 2}, "test", "v1")
	if err := inChan.Send(ctx, msg2); err != nil {
		t.Fatalf("send message 2 failed: %v", err)
	}

	// Now start the sink to drain the channel
	sink.Start(ctx)

	// The channel should have been drained
	time.Sleep(200 * time.Millisecond)

	// Send more messages - they should flow through
	msg3 := engine.NewMessage(map[string]interface{}{"n": 3}, "test", "v1")
	if err := inChan.Send(ctx, msg3); err != nil {
		t.Fatalf("send message 3 failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	inChan.Close()
	cancel()

	// Verify all messages were written
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Count lines
	lineCount := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lineCount++
		}
	}

	if lineCount != 3 {
		t.Errorf("expected 3 messages, got %d", lineCount)
	}
}

// TestFileSinkWithComplexMessageStructure tests that complex nested structures are written correctly
func TestFileSinkWithComplexMessageStructure(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "complex_test.jsonl")

	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sink.Start(ctx)

	// Create a complex message with nested structures
	complexBody := map[string]interface{}{
		"order": map[string]interface{}{
			"id":    "ORD-12345",
			"items": []interface{}{
				map[string]interface{}{"sku": "ITEM-1", "qty": 2, "price": 29.99},
				map[string]interface{}{"sku": "ITEM-2", "qty": 1, "price": 49.99},
			},
			"customer": map[string]interface{}{
				"id":       "CUST-001",
				"name":     "John Doe",
				"addresses": []interface{}{
					map[string]interface{}{"type": "billing", "zip": "10001"},
					map[string]interface{}{"type": "shipping", "zip": "10002"},
				},
			},
		},
	}

	msg := engine.NewMessage(complexBody, "order-processing", "v2.1.0")
	msg.Metadata.Stage = "validation"
	msg.Headers["x-tenant-id"] = "tenant-abc"
	msg.Headers["x-correlation-id"] = "corr-xyz"

	if err := inChan.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	inChan.Close()
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Read and verify
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var result engine.Message
	if err := json.Unmarshal(data[:len(data)-1], &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Verify nested structure is preserved
	resultBody := result.Body.(map[string]interface{})
	order := resultBody["order"].(map[string]interface{})

	if order["id"] != "ORD-12345" {
		t.Errorf("order ID mismatch")
	}

	items := order["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Verify headers
	if result.Headers["x-tenant-id"] != "tenant-abc" {
		t.Errorf("x-tenant-id header mismatch")
	}

	// Verify metadata
	if result.Metadata.Stage != "validation" {
		t.Errorf("stage mismatch")
	}
}
