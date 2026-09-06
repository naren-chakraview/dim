package sdk

import (
	"fmt"
	"testing"
	"time"
)

// MockPlugin is a simple plugin for testing conformance
type MockPlugin struct {
	callCount int
}

func (m *MockPlugin) Call(args ...interface{}) (interface{}, error) {
	m.callCount++

	// Handle specific test cases
	if len(args) > 0 {
		if str, ok := args[0].(string); ok {
			if str == "invalid_input" {
				return nil, fmt.Errorf("invalid input")
			}
			if str == "test" {
				return map[string]interface{}{
					"input":      str,
					"call_count": m.callCount,
				}, nil
			}
		}
	}

	// Echo back all arguments
	return map[string]interface{}{
		"args":  args,
		"count": len(args),
	}, nil
}

// TestConformanceSuiteStructure verifies the suite is well-formed
func TestConformanceSuiteStructure(t *testing.T) {
	suite := DefaultConformanceSuite()

	if suite.Name == "" {
		t.Error("Suite name should not be empty")
	}

	if len(suite.Tests) == 0 {
		t.Error("Suite should have at least one test")
	}

	for _, test := range suite.Tests {
		if test.Name == "" {
			t.Error("Test name should not be empty")
		}

		if len(test.Meta) == 0 {
			t.Errorf("Test %q should have metadata", test.Name)
		}

		if _, exists := test.Meta["description"]; !exists {
			t.Errorf("Test %q should have a description in metadata", test.Name)
		}
	}
}

// TestPluginConformance verifies a plugin against the conformance suite
func TestPluginConformance(t *testing.T) {
	plugin := &MockPlugin{}
	suite := DefaultConformanceSuite()

	results := TestPlugin(plugin, suite)

	if len(results) != len(suite.Tests) {
		t.Errorf("Expected %d results, got %d", len(suite.Tests), len(results))
	}

	// Count passes and failures
	passed, failed, summary := SummarizeResults(results)

	t.Logf("Conformance results: %s", summary)

	if failed > 0 {
		for _, r := range results {
			if !r.Passed {
				t.Logf("  ❌ %s: %s", r.Test, r.Message)
			}
		}
	}

	// For this mock plugin, we expect it to pass basic tests
	if passed < 3 {
		t.Errorf("Expected at least 3 passing tests, got %d", passed)
	}
}

// TestDeterminismCheck verifies that the same inputs produce same outputs
func TestDeterminismCheck(t *testing.T) {
	plugin := &MockPlugin{}

	// Make two identical calls
	result1, err1 := plugin.Call("test")
	if err1 != nil {
		t.Fatalf("First call failed: %v", err1)
	}

	result2, err2 := plugin.Call("test")
	if err2 != nil {
		t.Fatalf("Second call failed: %v", err2)
	}

	// Results should be identical in structure (but call_count will differ)
	m1, ok1 := result1.(map[string]interface{})
	m2, ok2 := result2.(map[string]interface{})

	if !ok1 || !ok2 {
		t.Fatalf("Expected map results, got %T and %T", result1, result2)
	}

	// Both should have same input
	if m1["input"] != m2["input"] {
		t.Errorf("Input mismatch: %v vs %v", m1["input"], m2["input"])
	}
}

// TestErrorHandling verifies error cases are handled correctly
func TestErrorHandling(t *testing.T) {
	plugin := &MockPlugin{}

	_, err := plugin.Call("invalid_input")
	if err == nil {
		t.Error("Expected error for invalid_input")
	}

	if err.Error() != "invalid input" {
		t.Errorf("Expected 'invalid input' error, got: %v", err)
	}
}

// TestConformanceTimeout verifies the timeout handling in conformance tests
func TestConformanceTimeout(t *testing.T) {
	// SlowPlugin that delays returns
	slowPlugin := &SlowPlugin{delay: 2 * time.Second}

	// Test with short timeout
	suite := &ConformanceSuite{
		Name: "Timeout Test",
		Tests: []ConformanceTest{
			{
				Name:    "slow_call",
				Inputs:  []interface{}{"hello"},
				Timeout: 500 * time.Millisecond,
				Meta: map[string]interface{}{
					"description": "This call should timeout",
				},
			},
		},
	}

	results := TestPlugin(slowPlugin, suite)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Passed {
		t.Error("Expected test to fail due to timeout")
	}

	if result.Error == nil {
		t.Error("Expected timeout error")
	}
}

// SlowPlugin delays its response for testing timeouts
type SlowPlugin struct {
	delay time.Duration
}

func (s *SlowPlugin) Call(args ...interface{}) (interface{}, error) {
	time.Sleep(s.delay)
	return map[string]interface{}{"delayed": true}, nil
}

// TestSummarizeResults verifies the summary formatting
func TestSummarizeResults(t *testing.T) {
	results := []ConformanceResult{
		{Test: "test1", Passed: true},
		{Test: "test2", Passed: true},
		{Test: "test3", Passed: false, Message: "failed"},
	}

	passed, failed, summary := SummarizeResults(results)

	if passed != 2 {
		t.Errorf("Expected 2 passed, got %d", passed)
	}

	if failed != 1 {
		t.Errorf("Expected 1 failed, got %d", failed)
	}

	if len(summary) == 0 {
		t.Error("Summary should not be empty")
	}

	t.Logf("Summary: %s", summary)
}

// TestMultipleArguments verifies plugins handle multiple argument types
func TestMultipleArguments(t *testing.T) {
	plugin := &MockPlugin{}

	result, err := plugin.Call("hello", 42, 3.14, true, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}

	count, ok := m["count"].(int)
	if !ok {
		t.Fatalf("Expected int count, got %T", m["count"])
	}

	if count != 5 {
		t.Errorf("Expected 5 arguments, got %d", count)
	}
}
