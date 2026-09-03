package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/authz"
	"github.com/naren-chakraview/dim/internal/engine"
)

const (
	testJWTSecret = "test-jwt-secret-for-integration"
)

func TestHTTPSourceWithJWTValidToken(t *testing.T) {
	// Create a message channel
	msgChan := engine.NewChannel("test-jwt-channel", 10)
	defer msgChan.Close()

	// Create an HTTP source with JWT validation
	jwtValidator := authz.NewJWTValidatorWithSecret(testJWTSecret)

	source, err := NewHTTPSourceWithAuth(0, "/ingest", msgChan, jwtValidator, false)
	if err != nil {
		t.Fatalf("NewHTTPSourceWithAuth failed: %v", err)
	}
	defer source.Close()

	// Get the actual port from the listener
	port := source.listener.Addr().(*net.TCPAddr).Port

	// Start the server in a goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = source.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create a valid JWT token
	token, err := authz.CreateTestToken("user-123", []string{"admin"}, map[string]string{
		"org": "test-org",
	}, testJWTSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	// Create test body
	testBody := map[string]interface{}{
		"action": "test",
		"data":   "payload",
	}
	bodyBytes, _ := json.Marshal(testBody)

	// Make request with valid JWT
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/ingest", port), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Route", "test-route")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	// Verify message was received with principal
	msgCtx, msgCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer msgCancel()

	msg, err := msgChan.Recv(msgCtx)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	if msg.Metadata.Principal == nil {
		t.Fatal("Principal should not be nil")
	}

	if msg.Metadata.Principal.Subject != "user-123" {
		t.Errorf("Principal.Subject = %s, want user-123", msg.Metadata.Principal.Subject)
	}

	if len(msg.Metadata.Principal.Roles) != 1 || msg.Metadata.Principal.Roles[0] != "admin" {
		t.Errorf("Principal.Roles = %v, want [admin]", msg.Metadata.Principal.Roles)
	}
}

func TestHTTPSourceWithJWTNoAuth(t *testing.T) {
	// Create a message channel
	msgChan := engine.NewChannel("test-jwt-no-auth", 10)
	defer msgChan.Close()

	// Create HTTP source with JWT validation but not requiring auth
	jwtValidator := authz.NewJWTValidatorWithSecret(testJWTSecret)

	source, err := NewHTTPSourceWithAuth(0, "/ingest", msgChan, jwtValidator, false)
	if err != nil {
		t.Fatalf("NewHTTPSourceWithAuth failed: %v", err)
	}
	defer source.Close()

	// Get the actual port
	port := source.listener.Addr().(*net.TCPAddr).Port

	// Start the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = source.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Make request WITHOUT Authorization header
	testBody := map[string]interface{}{"test": "data"}
	bodyBytes, _ := json.Marshal(testBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/ingest", port), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	// Verify message received with nil principal
	msgCtx, msgCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer msgCancel()

	msg, err := msgChan.Recv(msgCtx)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	if msg.Metadata.Principal != nil {
		t.Error("Principal should be nil when no auth header")
	}
}

func TestHTTPSourceWithJWTRequireAuth(t *testing.T) {
	// Create message channel
	msgChan := engine.NewChannel("test-jwt-require-auth", 10)
	defer msgChan.Close()

	// Create HTTP source with JWT validation REQUIRING auth
	jwtValidator := authz.NewJWTValidatorWithSecret(testJWTSecret)

	source, err := NewHTTPSourceWithAuth(0, "/ingest", msgChan, jwtValidator, true)
	if err != nil {
		t.Fatalf("NewHTTPSourceWithAuth failed: %v", err)
	}
	defer source.Close()

	port := source.listener.Addr().(*net.TCPAddr).Port

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = source.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Request without auth header should be rejected
	testBody := map[string]interface{}{"test": "data"}
	bodyBytes, _ := json.Marshal(testBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/ingest", port), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d (Unauthorized)", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHTTPSourceWithJWTInvalidToken(t *testing.T) {
	// Create message channel
	msgChan := engine.NewChannel("test-jwt-invalid", 10)
	defer msgChan.Close()

	jwtValidator := authz.NewJWTValidatorWithSecret(testJWTSecret)

	source, err := NewHTTPSourceWithAuth(0, "/ingest", msgChan, jwtValidator, false)
	if err != nil {
		t.Fatalf("NewHTTPSourceWithAuth failed: %v", err)
	}
	defer source.Close()

	port := source.listener.Addr().(*net.TCPAddr).Port

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = source.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Request with invalid token
	testBody := map[string]interface{}{"test": "data"}
	bodyBytes, _ := json.Marshal(testBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/ingest", port), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid.token.here")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d (Unauthorized)", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHTTPSourceWithoutJWTValidation(t *testing.T) {
	// Test that source works without JWT validator (backward compatibility)
	msgChan := engine.NewChannel("test-no-jwt", 10)
	defer msgChan.Close()

	source, err := NewHTTPSource(0, "/ingest", msgChan)
	if err != nil {
		t.Fatalf("NewHTTPSource failed: %v", err)
	}
	defer source.Close()

	port := source.listener.Addr().(*net.TCPAddr).Port

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = source.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Make request (should work without auth)
	testBody := map[string]interface{}{"test": "data"}
	bodyBytes, _ := json.Marshal(testBody)

	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/ingest", port), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	// Verify message received without principal
	msgCtx, msgCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer msgCancel()

	msg, err := msgChan.Recv(msgCtx)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	if msg.Metadata.Principal != nil {
		t.Error("Principal should be nil when JWT validator not configured")
	}
}
