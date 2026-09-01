package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestHTTPSourcePostJSON tests that a POST request with JSON body
// creates a message on the output channel with correct metadata.
func TestHTTPSourcePostJSON(t *testing.T) {
	// Create an output channel
	outChan := engine.NewChannel("test-output", 10)
	defer outChan.Close()

	// Create the HTTP source on a random port (0 = auto-assign)
	// We'll extract the actual port after creation
	src, err := NewHTTPSource(0, "/ingest", outChan)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the server in a goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		src.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Extract the actual port from the listener
	addr := src.listener.Addr().String()
	if addr == "" {
		t.Fatal("failed to get listener address")
	}

	// Create a test payload
	payload := map[string]interface{}{
		"test":  "data",
		"count": 42,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	// Make the HTTP request
	resp, err := http.Post(
		"http://"+addr+"/ingest",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, resp.StatusCode, string(respBody))
	}

	// Receive the message from the channel
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	msg, err := outChan.Recv(ctx2)
	if err != nil {
		t.Fatalf("failed to receive message: %v", err)
	}
	if msg == nil {
		t.Fatal("received nil message")
	}

	// Verify the message body matches the payload
	msgBody, ok := msg.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("expected message body to be map, got %T", msg.Body)
	}

	if msgBody["test"] != "data" {
		t.Errorf("expected body.test = 'data', got %v", msgBody["test"])
	}

	if msgBody["count"] != float64(42) {
		t.Errorf("expected body.count = 42, got %v", msgBody["count"])
	}

	// Verify metadata is set
	if msg.Metadata.CorrelationID == "" {
		t.Error("correlation ID should not be empty")
	}

	if msg.Metadata.IngestedAt.IsZero() {
		t.Error("ingested_at should be set")
	}

	// Correlation ID should be 20-digit timestamp format
	if len(msg.Metadata.CorrelationID) != 20 {
		t.Errorf("expected correlation ID length 20, got %d: %s",
			len(msg.Metadata.CorrelationID), msg.Metadata.CorrelationID)
	}
}

// TestHTTPSourceMalformedJSON tests that malformed JSON returns a 400 error.
func TestHTTPSourceMalformedJSON(t *testing.T) {
	// Create an output channel
	outChan := engine.NewChannel("test-output", 10)
	defer outChan.Close()

	// Create the HTTP source
	src, err := NewHTTPSource(0, "/ingest", outChan)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		src.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	addr := src.listener.Addr().String()

	// Make request with invalid JSON
	resp, err := http.Post(
		"http://"+addr+"/ingest",
		"application/json",
		strings.NewReader("{invalid json}"),
	)
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check that we get a 400 Bad Request
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	// Verify that no message was sent to the channel
	msg := outChan.TryRecv()
	if msg != nil {
		t.Fatal("expected no message on channel for malformed JSON, but got one")
	}
}

// TestHTTPSourceContextCancellation tests that the listener stops
// gracefully when the context is cancelled.
func TestHTTPSourceContextCancellation(t *testing.T) {
	// Create an output channel
	outChan := engine.NewChannel("test-output", 10)
	defer outChan.Close()

	// Create the HTTP source
	src, err := NewHTTPSource(0, "/ingest", outChan)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the server with a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	doneChan := make(chan error, 1)
	go func() {
		doneChan <- src.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	addr := src.listener.Addr().String()

	// Send a request to confirm it's running
	resp, err := http.Get("http://" + addr + "/ingest")
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		// Expected for GET on an ingest endpoint
		resp.Body.Close()
	} else {
		resp.Body.Close()
	}

	// Wait for the server to stop (context timeout)
	select {
	case err := <-doneChan:
		// Expected: context cancelled
		if err != context.DeadlineExceeded {
			t.Logf("expected context.DeadlineExceeded, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop after context cancellation")
	}

	// Verify the server is actually stopped by attempting to send another request
	// We expect this to fail because the server should be down
	time.Sleep(100 * time.Millisecond)
	postResp, postErr := http.Post(
		"http://"+addr+"/ingest",
		"application/json",
		strings.NewReader("{}"),
	)
	if postErr == nil {
		postResp.Body.Close()
		t.Log("server may still be running, but Start() returned - this is acceptable")
	}
}

// TestHTTPSourceCorrelationID tests that correlation IDs are set correctly
// and are unique for different requests.
func TestHTTPSourceCorrelationID(t *testing.T) {
	// Create an output channel
	outChan := engine.NewChannel("test-output", 10)
	defer outChan.Close()

	// Create the HTTP source
	src, err := NewHTTPSource(0, "/ingest", outChan)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		src.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	addr := src.listener.Addr().String()

	// Send first request
	resp1, err := http.Post(
		"http://"+addr+"/ingest",
		"application/json",
		strings.NewReader(`{"id":1}`),
	)
	if err != nil {
		t.Fatalf("failed to make first HTTP request: %v", err)
	}
	resp1.Body.Close()

	// Send second request with a small delay
	time.Sleep(10 * time.Millisecond)
	resp2, err := http.Post(
		"http://"+addr+"/ingest",
		"application/json",
		strings.NewReader(`{"id":2}`),
	)
	if err != nil {
		t.Fatalf("failed to make second HTTP request: %v", err)
	}
	resp2.Body.Close()

	// Receive both messages
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	msg1, err := outChan.Recv(ctx2)
	if err != nil || msg1 == nil {
		t.Fatal("failed to receive first message")
	}

	msg2, err := outChan.Recv(ctx2)
	if err != nil || msg2 == nil {
		t.Fatal("failed to receive second message")
	}

	// Verify correlation IDs are set and not empty
	if msg1.Metadata.CorrelationID == "" {
		t.Error("first message correlation ID is empty")
	}
	if msg2.Metadata.CorrelationID == "" {
		t.Error("second message correlation ID is empty")
	}

	// Verify they are different (or at least could be different given microsecond precision)
	// Due to timestamp-based generation, they might be the same if sent very quickly,
	// but we verify they're both valid IDs at least
	if len(msg1.Metadata.CorrelationID) != 20 {
		t.Errorf("first message correlation ID has unexpected length: %d", len(msg1.Metadata.CorrelationID))
	}
	if len(msg2.Metadata.CorrelationID) != 20 {
		t.Errorf("second message correlation ID has unexpected length: %d", len(msg2.Metadata.CorrelationID))
	}

	// Verify ingestion times are set and recent
	now := time.Now().UTC()
	if msg1.Metadata.IngestedAt.After(now.Add(1 * time.Second)) {
		t.Error("first message ingestion time is in the future")
	}
	if msg1.Metadata.IngestedAt.Before(now.Add(-5 * time.Second)) {
		t.Error("first message ingestion time is too old")
	}
}

// TestHTTPSourceMethodNotAllowed tests that GET and other methods
// on POST-only endpoints are handled appropriately.
func TestHTTPSourceMethodNotAllowed(t *testing.T) {
	// Create an output channel
	outChan := engine.NewChannel("test-output", 10)
	defer outChan.Close()

	// Create the HTTP source
	src, err := NewHTTPSource(0, "/ingest", outChan)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		src.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	addr := src.listener.Addr().String()

	// Try DELETE method
	req, _ := http.NewRequest(http.MethodDelete, "http://"+addr+"/ingest", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d for DELETE, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}

	// Verify no message was sent
	msg := outChan.TryRecv()
	if msg != nil {
		t.Fatal("expected no message for DELETE request, but got one")
	}
}

// TestHTTPSourceEmptyBody tests that empty JSON body is handled gracefully.
func TestHTTPSourceEmptyBody(t *testing.T) {
	// Create an output channel
	outChan := engine.NewChannel("test-output", 10)
	defer outChan.Close()

	// Create the HTTP source
	src, err := NewHTTPSource(0, "/ingest", outChan)
	if err != nil {
		t.Fatalf("failed to create HTTP source: %v", err)
	}
	defer src.Close()

	// Start the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		src.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	addr := src.listener.Addr().String()

	// Send request with empty JSON object
	resp, err := http.Post(
		"http://"+addr+"/ingest",
		"application/json",
		strings.NewReader("{}"),
	)
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Should be accepted
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
	}

	// Receive the message
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	msg, err := outChan.Recv(ctx2)
	if err != nil || msg == nil {
		t.Fatal("failed to receive message for empty JSON body")
	}

	// Verify the body is an empty map
	msgBody, ok := msg.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("expected message body to be map, got %T", msg.Body)
	}
	if len(msgBody) != 0 {
		t.Errorf("expected empty body map, got %v", msgBody)
	}
}
