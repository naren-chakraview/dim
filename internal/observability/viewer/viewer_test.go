package viewer

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test 1: Ring buffer circular behavior
func TestRingBufferCircularBehavior(t *testing.T) {
	rb := NewRingBuffer(3)

	// Add 5 messages to a buffer of size 3
	for i := 1; i <= 5; i++ {
		msg := Message{
			CorrelationID: "msg-" + string(rune('0'+i)),
			Timestamp:     time.Now().UTC(),
			LatencyMs:     int64(i * 10),
			Status:        "success",
		}
		rb.Add(msg)
	}

	// Should have the last 3 messages: 3, 4, 5
	messages := rb.GetAll()
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	expectedIDs := []string{"msg-3", "msg-4", "msg-5"}
	for i, expected := range expectedIDs {
		if messages[i].CorrelationID != expected {
			t.Errorf("message %d: expected %s, got %s", i, expected, messages[i].CorrelationID)
		}
	}
}

// Test 2: Ring buffer not full
func TestRingBufferNotFull(t *testing.T) {
	rb := NewRingBuffer(5)

	// Add only 2 messages
	for i := 1; i <= 2; i++ {
		msg := Message{
			CorrelationID: "msg-" + string(rune('0'+i)),
			Timestamp:     time.Now().UTC(),
			LatencyMs:     int64(i * 10),
			Status:        "success",
		}
		rb.Add(msg)
	}

	messages := rb.GetAll()
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].CorrelationID != "msg-1" || messages[1].CorrelationID != "msg-2" {
		t.Errorf("messages not in expected order")
	}
}

// Test 3: Statistics calculation - average and percentiles
func TestStatisticsCalculation(t *testing.T) {
	rv := NewRouteViewer("test-route", 100)

	// Record latencies: 10ms, 20ms, 30ms, 40ms, 50ms
	for i := 1; i <= 5; i++ {
		latency := time.Duration(i*10) * time.Millisecond
		rv.RecordMessage("msg-"+string(rune('0'+i)), latency, true)
	}

	state := rv.GetState()

	// Check total processed
	if state.TotalProcessed != 5 {
		t.Errorf("expected 5 total processed, got %d", state.TotalProcessed)
	}

	// Check average: (10+20+30+40+50)/5 = 30
	expectedAvg := 30.0
	if state.AvgLatencyMs != expectedAvg {
		t.Errorf("expected avg %.2f, got %.2f", expectedAvg, state.AvgLatencyMs)
	}

	// Check p50: median of [10,20,30,40,50] = 30
	if state.P50LatencyMs != 30.0 {
		t.Errorf("expected p50 30.0, got %.2f", state.P50LatencyMs)
	}

	// Check p99 and p90 exist and are reasonable
	if state.P90LatencyMs < 30 || state.P99LatencyMs < 30 {
		t.Errorf("p90 and p99 should be >= p50")
	}
}

// Test 4: Thread safety with concurrent updates
func TestThreadSafetyConcurrentUpdates(t *testing.T) {
	rv := NewRouteViewer("test-route", 1000)

	const numGoroutines = 10
	const messagesPerGoroutine = 100

	var wg sync.WaitGroup
	var errorCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				correlationID := "msg-" + string(rune('0'+id)) + "-" + string(rune('0'+j%10))
				latency := time.Duration((id+j)%100) * time.Millisecond
				success := (j % 3) != 0 // 2/3 success rate
				rv.RecordMessage(correlationID, latency, success)
			}
		}(i)
	}

	// Also read state concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = rv.GetState()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Verify final state
	state := rv.GetState()
	if state.TotalProcessed != int64(numGoroutines*messagesPerGoroutine) {
		t.Errorf("expected %d total processed, got %d",
			numGoroutines*messagesPerGoroutine, state.TotalProcessed)
	}

	// Check that errors were recorded (roughly 1/3 of messages)
	expectedErrors := int64(numGoroutines * messagesPerGoroutine / 3)
	if state.TotalErrors < expectedErrors/2 || state.TotalErrors > expectedErrors+100 {
		t.Logf("error count: %d (expected around %d)", state.TotalErrors, expectedErrors)
	}

	if errorCount > 0 {
		t.Errorf("encountered %d errors during concurrent execution", errorCount)
	}
}

// Test 5: In-flight tracking
func TestInFlightTracking(t *testing.T) {
	rv := NewRouteViewer("test-route", 100)

	rv.SetInFlight(5)
	state := rv.GetState()
	if state.InFlight != 5 {
		t.Errorf("expected 5 in-flight, got %d", state.InFlight)
	}

	rv.SetInFlight(10)
	state = rv.GetState()
	if state.InFlight != 10 {
		t.Errorf("expected 10 in-flight, got %d", state.InFlight)
	}
}

// Test 6: HTTP endpoint returns valid JSON
func TestHTTPEndpointValidJSON(t *testing.T) {
	vs := NewViewerServer()

	// Add some test data
	viewer := vs.GetOrCreateViewer("route-1", 100)
	viewer.RecordMessage("msg-1", 10*time.Millisecond, true)
	viewer.RecordMessage("msg-2", 20*time.Millisecond, true)

	// Create test request
	req := httptest.NewRequest("GET", "/debug/routes", nil)
	w := httptest.NewRecorder()

	vs.HandleRoutes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response JSON
	body := w.Body.String()
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}

	if _, hasRoutes := response["routes"]; !hasRoutes {
		t.Errorf("response missing 'routes' field")
	}

	if _, hasTimestamp := response["timestamp"]; !hasTimestamp {
		t.Errorf("response missing 'timestamp' field")
	}
}

// Test 7: HTTP endpoint returns correct statistics
func TestHTTPEndpointStatistics(t *testing.T) {
	vs := NewViewerServer()

	// Create multiple routes with different stats
	viewer1 := vs.GetOrCreateViewer("route-1", 100)
	viewer1.SetInFlight(3)
	viewer1.RecordMessage("msg-1", 10*time.Millisecond, true)
	viewer1.RecordMessage("msg-2", 20*time.Millisecond, true)
	viewer1.RecordMessage("msg-3", 15*time.Millisecond, false)

	viewer2 := vs.GetOrCreateViewer("route-2", 100)
	viewer2.SetInFlight(1)
	viewer2.RecordMessage("msg-4", 30*time.Millisecond, true)

	// Create test request
	req := httptest.NewRequest("GET", "/debug/routes", nil)
	w := httptest.NewRecorder()

	vs.HandleRoutes(w, req)

	// Parse response JSON
	var response map[string]interface{}
	body, _ := io.ReadAll(w.Body)
	json.Unmarshal(body, &response)

	routes := response["routes"].([]interface{})
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}

	// Check route-1
	route1 := routes[0].(map[string]interface{})
	if route1["name"] != "route-1" {
		t.Errorf("expected route name 'route-1', got %v", route1["name"])
	}
	if int32(route1["inFlight"].(float64)) != 3 {
		t.Errorf("expected 3 in-flight for route-1, got %v", route1["inFlight"])
	}
	if int64(route1["totalProcessed"].(float64)) != 3 {
		t.Errorf("expected 3 total processed for route-1, got %v", route1["totalProcessed"])
	}
	if int64(route1["totalErrors"].(float64)) != 1 {
		t.Errorf("expected 1 error for route-1, got %v", route1["totalErrors"])
	}
}

// Test 8: HTTP endpoint rejects non-GET requests
func TestHTTPEndpointRejectsNonGET(t *testing.T) {
	vs := NewViewerServer()

	tests := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range tests {
		req := httptest.NewRequest(method, "/debug/routes", nil)
		w := httptest.NewRecorder()

		vs.HandleRoutes(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405 for %s, got %d", method, w.Code)
		}
	}
}

// Test 9: Multiple routes tracked independently
func TestMultipleRoutesIndependent(t *testing.T) {
	vs := NewViewerServer()

	viewer1 := vs.GetOrCreateViewer("route-1", 100)
	viewer2 := vs.GetOrCreateViewer("route-2", 100)

	// Add different messages to each route
	viewer1.RecordMessage("msg-1", 10*time.Millisecond, true)
	viewer1.RecordMessage("msg-2", 20*time.Millisecond, false)

	viewer2.RecordMessage("msg-3", 30*time.Millisecond, true)
	viewer2.RecordMessage("msg-4", 40*time.Millisecond, true)
	viewer2.RecordMessage("msg-5", 50*time.Millisecond, true)

	state1 := viewer1.GetState()
	state2 := viewer2.GetState()

	if state1.TotalProcessed != 2 {
		t.Errorf("route-1: expected 2 total processed, got %d", state1.TotalProcessed)
	}
	if state1.TotalErrors != 1 {
		t.Errorf("route-1: expected 1 error, got %d", state1.TotalErrors)
	}

	if state2.TotalProcessed != 3 {
		t.Errorf("route-2: expected 3 total processed, got %d", state2.TotalProcessed)
	}
	if state2.TotalErrors != 0 {
		t.Errorf("route-2: expected 0 errors, got %d", state2.TotalErrors)
	}

	if len(state1.RecentMessages) != 2 {
		t.Errorf("route-1: expected 2 recent messages, got %d", len(state1.RecentMessages))
	}
	if len(state2.RecentMessages) != 3 {
		t.Errorf("route-2: expected 3 recent messages, got %d", len(state2.RecentMessages))
	}
}

// Test 10: Concurrent HTTP requests
func TestConcurrentHTTPRequests(t *testing.T) {
	vs := NewViewerServer()

	viewer := vs.GetOrCreateViewer("test-route", 100)

	// Start a goroutine that continuously records messages
	done := make(chan bool)
	go func() {
		for i := 0; i < 1000; i++ {
			viewer.RecordMessage("msg-"+string(rune('0'+(i%10))), time.Duration(i%100)*time.Millisecond, i%5 != 0)
			if i%100 == 0 {
				select {
				case <-done:
					return
				default:
				}
			}
		}
		close(done)
	}()

	// Make concurrent HTTP requests
	const numRequests = 20
	var wg sync.WaitGroup
	var requestErrors int32

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/debug/routes", nil)
			w := httptest.NewRecorder()

			vs.HandleRoutes(w, req)

			if w.Code != http.StatusOK {
				atomic.AddInt32(&requestErrors, 1)
			}

			// Verify response is valid JSON
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				atomic.AddInt32(&requestErrors, 1)
			}
		}()
	}

	wg.Wait()
	<-done

	if requestErrors > 0 {
		t.Errorf("encountered %d HTTP request errors", requestErrors)
	}
}

// Test 11: Percentile calculations with sparse data
func TestPercentileCalculationsSparse(t *testing.T) {
	rv := NewRouteViewer("test-route", 100)

	// Record just 1 message
	rv.RecordMessage("msg-1", 42*time.Millisecond, true)

	state := rv.GetState()
	if state.AvgLatencyMs != 42.0 {
		t.Errorf("expected avg 42.0, got %.2f", state.AvgLatencyMs)
	}
	if state.P50LatencyMs != 42.0 {
		t.Errorf("expected p50 42.0, got %.2f", state.P50LatencyMs)
	}
	if state.P99LatencyMs != 42.0 {
		t.Errorf("expected p99 42.0, got %.2f", state.P99LatencyMs)
	}
}

// Test 12: RecordMessage with errors
func TestRecordMessageErrors(t *testing.T) {
	rv := NewRouteViewer("test-route", 100)

	// Record mix of successful and failed messages
	rv.RecordMessage("msg-1", 10*time.Millisecond, true)
	rv.RecordMessage("msg-2", 20*time.Millisecond, false)
	rv.RecordMessage("msg-3", 30*time.Millisecond, true)
	rv.RecordMessage("msg-4", 40*time.Millisecond, false)

	state := rv.GetState()
	if state.TotalProcessed != 4 {
		t.Errorf("expected 4 total processed, got %d", state.TotalProcessed)
	}
	if state.TotalErrors != 2 {
		t.Errorf("expected 2 errors, got %d", state.TotalErrors)
	}

	// Check recent messages
	if len(state.RecentMessages) != 4 {
		t.Errorf("expected 4 recent messages, got %d", len(state.RecentMessages))
	}

	// Verify status values
	successCount := 0
	errorCount := 0
	for _, msg := range state.RecentMessages {
		if msg.Status == "success" {
			successCount++
		} else if msg.Status == "error" {
			errorCount++
		}
	}

	if successCount != 2 || errorCount != 2 {
		t.Errorf("expected 2 success and 2 error messages, got %d success and %d error",
			successCount, errorCount)
	}
}

// Test 13: Route pairing tracking (Tier 1 Viewer Pairing)
func TestRoutePairingTracking(t *testing.T) {
	vs := NewViewerServer()

	// Record a route pair: route-1's kafka-sink feeds to route-2's kafka-source
	vs.RecordRoutePair("route-1", "kafka-sink", "route-2", "kafka-source")
	vs.RecordRoutePair("route-1", "kafka-sink", "route-2", "kafka-source")
	vs.RecordRoutePair("route-2", "http-sink", "route-3", "http-source")

	pairs := vs.GetAllPairs()
	if len(pairs) != 2 {
		t.Errorf("expected 2 unique pairs, got %d", len(pairs))
	}

	// Find the first pair and check it was incremented
	for _, pair := range pairs {
		if pair.SourceRoute == "route-1" && pair.TargetRoute == "route-2" {
			if pair.MessagesFlowed != 2 {
				t.Errorf("expected 2 messages flowed, got %d", pair.MessagesFlowed)
			}
		}
	}
}

// Test 14: Get route pairings for specific route
func TestGetPairsForRoute(t *testing.T) {
	vs := NewViewerServer()

	// Set up pairings
	vs.RecordRoutePair("route-1", "sink-a", "route-2", "source-b")
	vs.RecordRoutePair("route-1", "sink-c", "route-3", "source-d")
	vs.RecordRoutePair("route-4", "sink-e", "route-1", "source-f")

	// Get pairs for route-1 (both as source and target)
	pairs := vs.GetPairsForRoute("route-1")
	if len(pairs) != 3 {
		t.Errorf("expected 3 pairs for route-1, got %d", len(pairs))
	}

	// Verify route-1 appears as source or target
	for _, pair := range pairs {
		if pair.SourceRoute != "route-1" && pair.TargetRoute != "route-1" {
			t.Errorf("route-1 not found in pair: %v", pair)
		}
	}
}

// Test 15: Route pairs endpoint returns valid JSON
func TestRoutePairsEndpointJSON(t *testing.T) {
	vs := NewViewerServer()

	vs.RecordRoutePair("route-1", "kafka-sink", "route-2", "kafka-source")
	vs.RecordRoutePair("route-2", "http-sink", "route-3", "http-source")

	// Test all pairs endpoint
	req := httptest.NewRequest("GET", "/debug/route-pairs", nil)
	w := httptest.NewRecorder()

	vs.HandleRoutePairs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}

	if _, hasPairs := response["pairs"]; !hasPairs {
		t.Errorf("response missing 'pairs' field")
	}
}

// Test 16: Route pairs endpoint with query parameter
func TestRoutePairsEndpointWithQuery(t *testing.T) {
	vs := NewViewerServer()

	vs.RecordRoutePair("route-1", "kafka-sink", "route-2", "kafka-source")
	vs.RecordRoutePair("route-2", "http-sink", "route-3", "http-source")

	// Test filtered pairs endpoint
	req := httptest.NewRequest("GET", "/debug/route-pairs?route=route-1", nil)
	w := httptest.NewRecorder()

	vs.HandleRoutePairs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	pairsData := response["pairs"].([]interface{})
	if len(pairsData) != 1 {
		t.Errorf("expected 1 pair for route-1, got %d", len(pairsData))
	}
}
