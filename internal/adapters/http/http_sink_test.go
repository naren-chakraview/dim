package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestHTTPSinkBasic verifies basic HTTP sink functionality (M1.3.3)
func TestHTTPSinkBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["message"] != "test" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL:     server.URL,
		Timeout: 5000,
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"message": "test"}, "test-route", "v1")

	results := sink.Write(ctx, []*engine.Message{msg})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Expected success, got error: %v", results[0].Error)
	}
}

// TestHTTPSinkWithOBO verifies HTTP sink with OBO token exchange (M1.3.3)
func TestHTTPSinkWithOBO(t *testing.T) {
	// Mock downstream API that verifies scoped token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify scoped token (simplified check)
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" || len(token) < 10 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL:     server.URL,
		Timeout: 5000,
		OBO: &OBOConfig{
			Scope:    "read:orders",
			Audience: "test-api",
			TTL:      1800,
		},
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	sink.SetOBOSecret("test-secret")
	sink.SetRouteVersion("v1")

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"order_id": "123"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "alice@corp.com",
		Token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhbGljZUBjb3JwLmNvbSIsInNjb3BlIjoicmVhZDpvcmRlcnMgd3JpdGU6Y3VzdG9tZXJzIiwiaWF0IjoxNzI1NDU2MDAwLCJleHAiOjE3MjU0NTk2MDB9.test-sig",
	}

	results := sink.Write(ctx, []*engine.Message{msg})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Expected success with OBO token, got error: %v", results[0].Error)
	}
}

// TestHTTPSinkHTTPError verifies error handling for HTTP failures (M1.3.3)
func TestHTTPSinkHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL:     server.URL,
		Timeout: 5000,
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"test": true}, "test-route", "v1")

	results := sink.Write(ctx, []*engine.Message{msg})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Success {
		t.Error("Expected error for HTTP 500")
	}

	if !strings.Contains(results[0].Error.Error(), "HTTP 500") {
		t.Errorf("Expected HTTP 500 error, got: %v", results[0].Error)
	}
}

// TestHTTPSinkCorrelationID verifies correlation ID header (M1.3.3)
func TestHTTPSinkCorrelationID(t *testing.T) {
	var receivedCorrelationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCorrelationID = r.Header.Get("X-Correlation-ID")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL:     server.URL,
		Timeout: 5000,
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"test": true}, "test-route", "v1")
	msg.Metadata.CorrelationID = "corr-12345"

	sink.Write(ctx, []*engine.Message{msg})

	if receivedCorrelationID != "corr-12345" {
		t.Errorf("Expected correlation ID 'corr-12345', got '%s'", receivedCorrelationID)
	}
}

// TestHTTPSinkHealthCheck verifies health check (M1.3.3)
func TestHTTPSinkHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	err = sink.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

// TestHTTPSinkMultipleMessages verifies batch write (M1.3.3)
func TestHTTPSinkMultipleMessages(t *testing.T) {
	successCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL:     server.URL,
		Timeout: 5000,
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	ctx := context.Background()
	msgs := make([]*engine.Message, 3)
	for i := 0; i < 3; i++ {
		msgs[i] = engine.NewMessage(
			map[string]interface{}{"id": i},
			"test-route",
			"v1",
		)
	}

	results := sink.Write(ctx, msgs)

	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	if successCount != 3 {
		t.Errorf("Expected 3 successful writes, got %d", successCount)
	}
}

// TestHTTPSinkBadJSON verifies error handling for marshaling (M1.3.3)
func TestHTTPSinkBadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL:     server.URL,
		Timeout: 5000,
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	ctx := context.Background()

	// Create a message with unmarshalable body (channel)
	msg := engine.NewMessage(nil, "test-route", "v1")
	msg.Body = make(chan int) // Channels can't be marshaled to JSON

	results := sink.Write(ctx, []*engine.Message{msg})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Success {
		t.Error("Expected error for unmarshalable body")
	}

	if !strings.Contains(results[0].Error.Error(), "marshal") {
		t.Errorf("Expected marshaling error, got: %v", results[0].Error)
	}
}

// TestHTTPSinkNoAuthorization verifies request without auth header (M1.3.3)
func TestHTTPSinkNoAuthorization(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink, err := NewHTTPSink(&HTTPSinkConfig{
		URL:     server.URL,
		Timeout: 5000,
	})
	if err != nil {
		t.Fatalf("NewHTTPSink failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"test": true}, "test-route", "v1")
	msg.Metadata.Principal = nil // No principal

	sink.Write(ctx, []*engine.Message{msg})

	if receivedAuth != "" {
		t.Errorf("Expected no Authorization header, got: %s", receivedAuth)
	}
}
