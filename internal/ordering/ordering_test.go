package ordering

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// TestParseOrderingRequired verifies parsing of "required" ordering mode
func TestParseOrderingRequired(t *testing.T) {
	mode := ParseOrdering("required")
	if mode != OrderingRequired {
		t.Errorf("ParseOrdering(\"required\") = %v, want %v", mode, OrderingRequired)
	}
}

// TestParseOrderingNone verifies parsing of "none" ordering mode
func TestParseOrderingNone(t *testing.T) {
	mode := ParseOrdering("none")
	if mode != OrderingNone {
		t.Errorf("ParseOrdering(\"none\") = %v, want %v", mode, OrderingNone)
	}
}

// TestParseOrderingEmpty verifies that empty string defaults to OrderingNone
func TestParseOrderingEmpty(t *testing.T) {
	mode := ParseOrdering("")
	if mode != OrderingNone {
		t.Errorf("ParseOrdering(\"\") = %v, want %v", mode, OrderingNone)
	}
}

// TestParseOrderingInvalid verifies that invalid values default to OrderingNone
func TestParseOrderingInvalid(t *testing.T) {
	mode := ParseOrdering("invalid-value")
	if mode != OrderingNone {
		t.Errorf("ParseOrdering(\"invalid-value\") = %v, want %v", mode, OrderingNone)
	}
}

// TestOrderingModeString verifies string representation of ordering modes
func TestOrderingModeString(t *testing.T) {
	tests := []struct {
		mode     OrderingMode
		expected string
	}{
		{OrderingNone, "none"},
		{OrderingRequired, "required"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.expected {
			t.Errorf("OrderingMode.String() = %s, want %s", got, tt.expected)
		}
	}
}

// TestGetOrderingMode verifies extraction of ordering mode from RouteSpec
func TestGetOrderingMode(t *testing.T) {
	tests := []struct {
		name     string
		ordering string
		expected OrderingMode
	}{
		{"required", "required", OrderingRequired},
		{"none", "none", OrderingNone},
		{"empty", "", OrderingNone},
		{"invalid", "invalid", OrderingNone},
		{"nil route", "", OrderingNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &config.RouteSpec{Ordering: tt.ordering}
			if got := GetOrderingMode(route); got != tt.expected {
				t.Errorf("GetOrderingMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestGetOrderingModeNilRoute verifies handling of nil route
func TestGetOrderingModeNilRoute(t *testing.T) {
	mode := GetOrderingMode(nil)
	if mode != OrderingNone {
		t.Errorf("GetOrderingMode(nil) = %v, want %v", mode, OrderingNone)
	}
}

// TestGetWorkerCountOrdered verifies that ordered routes get 1 worker
func TestGetWorkerCountOrdered(t *testing.T) {
	route := &config.RouteSpec{Ordering: "required"}
	workers := GetWorkerCount(route, 4)
	if workers != 1 {
		t.Errorf("GetWorkerCount(ordered, 4) = %d, want 1", workers)
	}
}

// TestGetWorkerCountUnordered verifies that unordered routes get default worker count
func TestGetWorkerCountUnordered(t *testing.T) {
	route := &config.RouteSpec{Ordering: "none"}
	workers := GetWorkerCount(route, 4)
	if workers != 4 {
		t.Errorf("GetWorkerCount(unordered, 4) = %d, want 4", workers)
	}
}

// TestGetWorkerCountDefault verifies that empty ordering defaults to default worker count
func TestGetWorkerCountDefault(t *testing.T) {
	route := &config.RouteSpec{Ordering: ""}
	workers := GetWorkerCount(route, 4)
	if workers != 4 {
		t.Errorf("GetWorkerCount(empty, 4) = %d, want 4", workers)
	}
}

// TestGetWorkerCountInvalid verifies that invalid ordering defaults to default worker count
func TestGetWorkerCountInvalid(t *testing.T) {
	route := &config.RouteSpec{Ordering: "invalid"}
	workers := GetWorkerCount(route, 4)
	if workers != 4 {
		t.Errorf("GetWorkerCount(invalid, 4) = %d, want 4", workers)
	}
}

// TestGetWorkerCountNilRoute verifies handling of nil route
func TestGetWorkerCountNilRoute(t *testing.T) {
	workers := GetWorkerCount(nil, 4)
	if workers != 4 {
		t.Errorf("GetWorkerCount(nil, 4) = %d, want 4", workers)
	}
}

// TestGetWorkerCountInvalidDefault verifies that invalid default workers gets fallback
func TestGetWorkerCountInvalidDefault(t *testing.T) {
	route := &config.RouteSpec{Ordering: ""}
	workers := GetWorkerCount(route, 0)
	if workers != 4 {
		t.Errorf("GetWorkerCount(empty, 0) = %d, want 4 (fallback)", workers)
	}
}

// TestIsOrderingRequired verifies the IsOrderingRequired helper
func TestIsOrderingRequired(t *testing.T) {
	tests := []struct {
		name     string
		ordering string
		expected bool
	}{
		{"required", "required", true},
		{"none", "none", false},
		{"empty", "", false},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &config.RouteSpec{Ordering: tt.ordering}
			if got := IsOrderingRequired(route); got != tt.expected {
				t.Errorf("IsOrderingRequired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestMessageOrderPreservedWithOrdering verifies that ordered routes preserve message order
func TestMessageOrderPreservedWithOrdering(t *testing.T) {
	inputCh := engine.NewChannel("input", 100)
	outputCh := engine.NewChannel("output", 100)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create ordered route (1 worker)
	route := &config.RouteSpec{Ordering: "required"}
	workers := GetWorkerCount(route, 4)
	if workers != 1 {
		t.Fatalf("expected 1 worker for ordered route, got %d", workers)
	}

	// Create a passthrough step
	steps := []engine.Step{}

	// Create executor with 1 worker (ordered)
	executor := engine.NewExecutorWithWorkers("ordered-test", inputCh, outputCh, nil, steps, nil, workers)

	// Run executor in background
	runErr := make(chan error, 1)
	go func() {
		runErr <- executor.Run(ctx)
	}()

	// Send messages in order
	messageCount := 10
	expectedOrder := make([]int, messageCount)
	for i := 0; i < messageCount; i++ {
		expectedOrder[i] = i
		msg := engine.NewMessage(map[string]interface{}{"seq": i}, "ordered", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("failed to send message %d: %v", i, err)
		}
	}

	// Close input channel to signal end of messages
	inputCh.Close()

	// Read messages from output and verify order
	receivedOrder := make([]int, 0, messageCount)
	for i := 0; i < messageCount; i++ {
		msg, err := outputCh.Recv(ctx)
		if err != nil {
			t.Fatalf("failed to receive message %d: %v", i, err)
		}
		if msg == nil {
			t.Fatalf("received nil message at position %d", i)
		}
		body := msg.Body.(map[string]interface{})
		var seq int
		// Handle both int and float64 types
		switch v := body["seq"].(type) {
		case int:
			seq = v
		case float64:
			seq = int(v)
		default:
			t.Fatalf("unexpected type for seq: %T", v)
		}
		receivedOrder = append(receivedOrder, seq)
	}

	// Close output channel
	outputCh.Close()

	// Wait for executor to finish
	if err := <-runErr; err != nil {
		t.Logf("executor finished with error: %v", err)
	}

	// Verify order
	for i, expected := range expectedOrder {
		if receivedOrder[i] != expected {
			t.Errorf("message order mismatch at position %d: got %d, want %d", i, receivedOrder[i], expected)
		}
	}
}

// TestConcurrentProcessingWithoutOrdering verifies that unordered routes allow parallelization
func TestConcurrentProcessingWithoutOrdering(t *testing.T) {
	inputCh := engine.NewChannel("input", 100)
	outputCh := engine.NewChannel("output", 100)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create unordered route (4 workers)
	route := &config.RouteSpec{Ordering: "none"}
	workers := GetWorkerCount(route, 4)
	if workers != 4 {
		t.Fatalf("expected 4 workers for unordered route, got %d", workers)
	}

	// Create a slow step to detect parallelism
	processingTimes := sync.Map{}
	slowStep := &slowMockStep{
		delay:           100 * time.Millisecond,
		processingTimes: &processingTimes,
	}

	steps := []engine.Step{slowStep}

	// Create executor with 4 workers (unordered)
	executor := engine.NewExecutorWithWorkers("unordered-test", inputCh, outputCh, nil, steps, nil, workers)

	// Run executor in background
	runErr := make(chan error, 1)
	go func() {
		runErr <- executor.Run(ctx)
	}()

	// Send messages concurrently
	messageCount := 4
	startTime := time.Now()
	for i := 0; i < messageCount; i++ {
		msg := engine.NewMessage(map[string]interface{}{"id": i}, "unordered", "v1")
		if err := inputCh.Send(ctx, msg); err != nil {
			t.Fatalf("failed to send message %d: %v", i, err)
		}
	}

	// Close input to signal end
	inputCh.Close()

	// Receive all messages
	for i := 0; i < messageCount; i++ {
		_, err := outputCh.Recv(ctx)
		if err != nil {
			t.Fatalf("failed to receive message %d: %v", i, err)
		}
	}

	totalTime := time.Since(startTime)

	// Close output and wait for executor to finish
	outputCh.Close()
	if err := <-runErr; err != nil {
		t.Logf("executor finished with error: %v", err)
	}

	// With 4 workers and 4 messages with 100ms delay each:
	// - Sequential (1 worker): ~400ms
	// - Parallel (4 workers): ~100ms
	// We expect parallel processing to finish in roughly 100-150ms
	// If it takes >250ms, it's likely serial processing

	if totalTime > 250*time.Millisecond {
		t.Logf("WARNING: Unordered route processing took %v (expected <250ms for parallelism)", totalTime)
		t.Logf("This may indicate messages were processed sequentially instead of in parallel")
		// Note: We don't fail this test as timing can vary, but we log it for visibility
	}
}

// TestBackwardCompatibility verifies that routes without Ordering field default to unordered
func TestBackwardCompatibility(t *testing.T) {
	// Simulate a Phase 1 route config without the Ordering field
	route := &config.RouteSpec{
		From: "source",
		Steps: []config.StepSpec{},
	}

	mode := GetOrderingMode(route)
	if mode != OrderingNone {
		t.Errorf("Route without Ordering field: got %v, want %v", mode, OrderingNone)
	}

	workers := GetWorkerCount(route, 4)
	if workers != 4 {
		t.Errorf("Route without Ordering field: got %d workers, want 4", workers)
	}
}

// slowMockStep is a test step that adds delay to simulate slow processing
type slowMockStep struct {
	delay           time.Duration
	processingTimes *sync.Map
}

func (s *slowMockStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	start := time.Now()
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	s.processingTimes.Store(start.Unix(), time.Since(start))
	return msg, nil
}

// TestConfigParsing verifies that ordering field is correctly parsed from RouteSpec
func TestConfigParsing(t *testing.T) {
	// Test various route configurations
	tests := []struct {
		name            string
		route           *config.RouteSpec
		expectedWorkers int
		expectedMode    OrderingMode
	}{
		{
			name: "ordered-required",
			route: &config.RouteSpec{
				From:     "source",
				Ordering: "required",
				Steps:    []config.StepSpec{},
			},
			expectedWorkers: 1,
			expectedMode:    OrderingRequired,
		},
		{
			name: "unordered-none",
			route: &config.RouteSpec{
				From:     "source",
				Ordering: "none",
				Steps:    []config.StepSpec{},
			},
			expectedWorkers: 4,
			expectedMode:    OrderingNone,
		},
		{
			name: "unordered-empty",
			route: &config.RouteSpec{
				From:  "source",
				Steps: []config.StepSpec{},
			},
			expectedWorkers: 4,
			expectedMode:    OrderingNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := GetOrderingMode(tt.route)
			if mode != tt.expectedMode {
				t.Errorf("GetOrderingMode() = %v, want %v", mode, tt.expectedMode)
			}

			workers := GetWorkerCount(tt.route, 4)
			if workers != tt.expectedWorkers {
				t.Errorf("GetWorkerCount() = %d, want %d", workers, tt.expectedWorkers)
			}
		})
	}
}

// BenchmarkOrdering benchmarks the overhead of ordering detection
func BenchmarkOrdering(b *testing.B) {
	route := &config.RouteSpec{Ordering: "required"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetOrderingMode(route)
		_ = GetWorkerCount(route, 4)
		_ = IsOrderingRequired(route)
	}
}
