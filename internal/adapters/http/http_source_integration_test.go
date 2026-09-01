package http

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"net/http"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestHTTPSourceIntegration is an integration test that verifies an HTTP source
// can accept a POST request and deliver it through the engine's executor pipeline.
func TestHTTPSourceIntegration(t *testing.T) {
	// Create channels: HTTP source output -> executor input -> executor output
	sourceOut := engine.NewChannel("http-source-output", 10)
	defer sourceOut.Close()

	executorOut := engine.NewChannel("executor-output", 10)
	defer executorOut.Close()

	// Create the HTTP source
	src, err := NewHTTPSource(0, "/ingest", sourceOut)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the HTTP source
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		src.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create a simple passthrough executor (no steps, just forwards messages)
	executor := engine.NewExecutor("passthrough", sourceOut, executorOut, []engine.Step{})

	// Run the executor in a goroutine
	executorDone := make(chan error, 1)
	go func() {
		executorDone <- executor.Run(ctx)
	}()

	// Give the executor a moment to start
	time.Sleep(50 * time.Millisecond)

	addr := src.listener.Addr().String()

	// Send a test payload via HTTP
	testPayload := map[string]interface{}{
		"order_id":    "ORD-12345",
		"customer":    "Alice",
		"total":       99.99,
		"items_count": 3,
	}
	payloadJSON, err := json.Marshal(testPayload)
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}

	resp, err := http.Post(
		"http://"+addr+"/ingest",
		"application/json",
		strings.NewReader(string(payloadJSON)),
	)
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
	}

	// Receive the processed message from the executor's output
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	processedMsg, err := executorOut.Recv(ctx2)
	if err != nil {
		t.Fatalf("failed to receive message from executor output: %v", err)
	}
	if processedMsg == nil {
		t.Fatal("received nil message from executor output")
	}

	// Verify the payload was correctly transmitted
	msgBody, ok := processedMsg.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("expected message body to be map, got %T", processedMsg.Body)
	}

	if msgBody["order_id"] != "ORD-12345" {
		t.Errorf("expected order_id = 'ORD-12345', got %v", msgBody["order_id"])
	}

	if msgBody["customer"] != "Alice" {
		t.Errorf("expected customer = 'Alice', got %v", msgBody["customer"])
	}

	if msgBody["total"] != 99.99 {
		t.Errorf("expected total = 99.99, got %v", msgBody["total"])
	}

	// Verify metadata is preserved through the executor
	if processedMsg.Metadata.CorrelationID == "" {
		t.Error("correlation ID should be preserved through executor")
	}

	if processedMsg.Metadata.IngestedAt.IsZero() {
		t.Error("ingested_at should be preserved through executor")
	}

	t.Logf("Integration test passed: message payload transmitted through HTTP source -> executor pipeline")
	t.Logf("Correlation ID: %s", processedMsg.Metadata.CorrelationID)
	t.Logf("Ingested at: %s", processedMsg.Metadata.IngestedAt)
}

// TestHTTPSourceMultipleMessages tests that multiple messages can be
// sent through the HTTP source and processed sequentially.
func TestHTTPSourceMultipleMessages(t *testing.T) {
	// Create channels
	sourceOut := engine.NewChannel("http-source-output", 10)
	defer sourceOut.Close()

	executorOut := engine.NewChannel("executor-output", 10)
	defer executorOut.Close()

	// Create the HTTP source
	src, err := NewHTTPSource(0, "/ingest", sourceOut)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the HTTP source
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		src.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create executor
	executor := engine.NewExecutor("passthrough", sourceOut, executorOut, []engine.Step{})

	go func() {
		executor.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	addr := src.listener.Addr().String()

	// Send multiple messages
	numMessages := 5
	for i := 0; i < numMessages; i++ {
		payload := map[string]interface{}{
			"id":    i + 1,
			"value": (i + 1) * 10,
		}
		payloadJSON, _ := json.Marshal(payload)

		resp, err := http.Post(
			"http://"+addr+"/ingest",
			"application/json",
			strings.NewReader(string(payloadJSON)),
		)
		if err != nil {
			t.Fatalf("failed to send message %d: %v", i+1, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("message %d: expected status %d, got %d", i+1, http.StatusAccepted, resp.StatusCode)
		}
	}

	// Verify all messages were received in order
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	for i := 0; i < numMessages; i++ {
		msg, err := executorOut.Recv(ctx2)
		if err != nil {
			t.Fatalf("failed to receive message %d: %v", i+1, err)
		}
		if msg == nil {
			t.Fatalf("received nil message at position %d", i+1)
		}

		msgBody, ok := msg.Body.(map[string]interface{})
		if !ok {
			t.Fatalf("message %d: expected body to be map, got %T", i+1, msg.Body)
		}

		expectedID := float64(i + 1)
		if msgBody["id"] != expectedID {
			t.Errorf("message %d: expected id = %v, got %v", i+1, expectedID, msgBody["id"])
		}

		expectedValue := float64((i + 1) * 10)
		if msgBody["value"] != expectedValue {
			t.Errorf("message %d: expected value = %v, got %v", i+1, expectedValue, msgBody["value"])
		}
	}

	t.Logf("Successfully processed %d messages through HTTP source -> executor pipeline", numMessages)
}
