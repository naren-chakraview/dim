package viewer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Integration test 1: Full viewer request flow with multiple routes
func TestViewerIntegrationFullFlow(t *testing.T) {
	// Create viewer server
	server := NewViewerServer()

	// Add test data for payment route
	payment := server.GetOrCreateViewer("payment-processing", 200)
	payment.SetInFlight(3)
	payment.RecordMessage("msg-001", 10*time.Millisecond, true)
	payment.RecordMessage("msg-002", 20*time.Millisecond, false)
	payment.RecordMessage("msg-003", 15*time.Millisecond, true)

	// Test HTTP handler directly
	req := httptest.NewRequest("GET", "/debug/routes", nil)
	w := httptest.NewRecorder()
	server.HandleRoutes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	// Verify response structure
	routes, ok := response["routes"].([]interface{})
	if !ok {
		t.Fatalf("'routes' field is not an array")
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	route := routes[0].(map[string]interface{})
	if route["name"] != "payment-processing" {
		t.Fatalf("expected route name 'payment-processing', got %v", route["name"])
	}

	if int32(route["inFlight"].(float64)) != 3 {
		t.Fatalf("expected 3 in-flight messages, got %d", int32(route["inFlight"].(float64)))
	}

	if int64(route["totalProcessed"].(float64)) != 3 {
		t.Fatalf("expected 3 total processed, got %d", int64(route["totalProcessed"].(float64)))
	}

	if int64(route["totalErrors"].(float64)) != 1 {
		t.Fatalf("expected 1 error, got %d", int64(route["totalErrors"].(float64)))
	}

	recentMessages, ok := route["recentMessages"].([]interface{})
	if !ok {
		t.Fatalf("'recentMessages' field is not an array")
	}

	if len(recentMessages) != 3 {
		t.Fatalf("expected 3 recent messages, got %d", len(recentMessages))
	}

	// Verify message data
	msg1 := recentMessages[0].(map[string]interface{})
	if msg1["correlationId"] != "msg-001" {
		t.Fatalf("expected msg-001, got %v", msg1["correlationId"])
	}
	if msg1["status"] != "success" {
		t.Fatalf("expected status 'success', got %v", msg1["status"])
	}

	msg2 := recentMessages[1].(map[string]interface{})
	if msg2["status"] != "error" {
		t.Fatalf("expected status 'error' for msg-002, got %v", msg2["status"])
	}
}

// Integration test 2: Multiple routes scenario
func TestViewerIntegrationMultipleRoutes(t *testing.T) {
	server := NewViewerServer()

	// Simulate three different routes
	payment := server.GetOrCreateViewer("payment-processing", 200)
	payment.SetInFlight(5)
	for i := 1; i <= 100; i++ {
		latency := time.Duration((i % 50)) * time.Millisecond
		success := i%10 != 0
		payment.RecordMessage(fmt.Sprintf("pay-%d", i), latency, success)
	}

	inventory := server.GetOrCreateViewer("inventory-update", 200)
	inventory.SetInFlight(2)
	for i := 1; i <= 50; i++ {
		latency := time.Duration((i % 30)) * time.Millisecond
		success := i%15 != 0
		inventory.RecordMessage(fmt.Sprintf("inv-%d", i), latency, success)
	}

	notify := server.GetOrCreateViewer("notifications", 200)
	notify.SetInFlight(8)
	for i := 1; i <= 200; i++ {
		latency := time.Duration((i % 20)) * time.Millisecond
		success := i%5 != 0
		notify.RecordMessage(fmt.Sprintf("notif-%d", i), latency, success)
	}

	// Make request
	req := httptest.NewRequest("GET", "/debug/routes", nil)
	w := httptest.NewRecorder()
	server.HandleRoutes(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	routes := response["routes"].([]interface{})
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	// Routes should be sorted by name
	route1 := routes[0].(map[string]interface{})
	route2 := routes[1].(map[string]interface{})
	route3 := routes[2].(map[string]interface{})

	names := []string{
		route1["name"].(string),
		route2["name"].(string),
		route3["name"].(string),
	}

	expectedNames := []string{"inventory-update", "notifications", "payment-processing"}
	for i, expected := range expectedNames {
		if names[i] != expected {
			t.Fatalf("route %d: expected %s, got %s", i, expected, names[i])
		}
	}

	// Verify statistics
	paymentRoute := routes[2].(map[string]interface{})
	if int32(paymentRoute["inFlight"].(float64)) != 5 {
		t.Fatalf("expected 5 in-flight for payment, got %d", int32(paymentRoute["inFlight"].(float64)))
	}
	if int64(paymentRoute["totalProcessed"].(float64)) != 100 {
		t.Fatalf("expected 100 processed for payment, got %d", int64(paymentRoute["totalProcessed"].(float64)))
	}

	// Verify latency statistics exist and are reasonable
	avgLatency := paymentRoute["avgLatencyMs"].(float64)
	p99Latency := paymentRoute["p99LatencyMs"].(float64)

	if avgLatency <= 0 || avgLatency >= 100 {
		t.Fatalf("average latency out of range: %.2f", avgLatency)
	}
	if p99Latency < avgLatency {
		t.Fatalf("p99 latency should be >= average: avg=%.2f, p99=%.2f", avgLatency, p99Latency)
	}
}

// Integration test 3: Response includes timestamp
func TestViewerIntegrationTimestamp(t *testing.T) {
	server := NewViewerServer()

	req := httptest.NewRequest("GET", "/debug/routes", nil)
	w := httptest.NewRecorder()
	server.HandleRoutes(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	timestamp, ok := response["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp is not a string: %v", response["timestamp"])
	}

	// Verify it's a valid RFC3339 timestamp
	_, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Fatalf("timestamp is not valid RFC3339: %s (error: %v)", timestamp, err)
	}
}

// Integration test 4: High-load concurrent message recording
func TestViewerIntegrationHighLoad(t *testing.T) {
	server := NewViewerServer()
	route := server.GetOrCreateViewer("high-load-route", 500)

	// Simulate concurrent message recording from 10 workers
	done := make(chan bool, 10)
	for w := 0; w < 10; w++ {
		go func(workerID int) {
			for i := 0; i < 100; i++ {
				msgID := fmt.Sprintf("worker-%d-msg-%d", workerID, i)
				latency := time.Duration((workerID*100+i)%200) * time.Millisecond
				success := (workerID+i)%7 != 0
				route.RecordMessage(msgID, latency, success)
			}
			done <- true
		}(w)
	}

	// Wait for all workers to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Make request and verify results
	req := httptest.NewRequest("GET", "/debug/routes", nil)
	w := httptest.NewRecorder()
	server.HandleRoutes(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	routes := response["routes"].([]interface{})
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	route1 := routes[0].(map[string]interface{})
	totalProcessed := int64(route1["totalProcessed"].(float64))

	if totalProcessed != 1000 {
		t.Fatalf("expected 1000 total processed, got %d", totalProcessed)
	}

	// Verify ring buffer only keeps the last 500 messages
	recentMessages := route1["recentMessages"].([]interface{})
	if len(recentMessages) > 500 {
		t.Fatalf("expected at most 500 recent messages, got %d", len(recentMessages))
	}
}
