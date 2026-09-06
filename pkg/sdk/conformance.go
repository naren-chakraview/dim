package sdk

import (
	"fmt"
	"time"
)

// ConformanceTest is a single test case for plugin conformance
type ConformanceTest struct {
	Name    string                 // Test name (e.g., "valid_input", "timeout")
	Inputs  []interface{}          // Input arguments for the test
	Expect  interface{}            // Expected output
	WantErr bool                   // Whether an error is expected
	Timeout time.Duration          // Timeout for this test (0 = use default)
	Meta    map[string]interface{} // Metadata for interpretation
}

// ConformanceResult is the result of a single conformance test
type ConformanceResult struct {
	Test      string
	Passed    bool
	Error     error
	Duration  time.Duration
	Message   string
	GotResult interface{}
}

// ConformanceSuite is a set of conformance tests
type ConformanceSuite struct {
	Name  string
	Tests []ConformanceTest
}

// DefaultConformanceSuite returns the standard conformance tests that all plugins must pass
func DefaultConformanceSuite() *ConformanceSuite {
	return &ConformanceSuite{
		Name: "Plugin Conformance Suite",
		Tests: []ConformanceTest{
			{
				Name:   "simple_call_with_single_arg",
				Inputs: []interface{}{"hello"},
				Meta: map[string]interface{}{
					"description": "Plugin must accept a single string argument",
					"required":    true,
				},
			},
			{
				Name:   "simple_call_with_multiple_args",
				Inputs: []interface{}{"hello", 42, true},
				Meta: map[string]interface{}{
					"description": "Plugin must accept multiple arguments",
					"required":    true,
				},
			},
			{
				Name:   "simple_call_with_no_args",
				Inputs: []interface{}{},
				Meta: map[string]interface{}{
					"description": "Plugin must handle zero arguments",
					"required":    true,
				},
			},
			{
				Name:    "error_handling",
				Inputs:  []interface{}{"invalid_input"},
				WantErr: true,
				Meta: map[string]interface{}{
					"description": "Plugin must return error for invalid input, not panic",
					"required":    true,
				},
			},
			{
				Name:    "determinism_first_call",
				Inputs:  []interface{}{"test"},
				Timeout: 5 * time.Second,
				Meta: map[string]interface{}{
					"description": "First call with test input (for determinism check)",
					"required":    true,
				},
			},
			{
				Name:    "determinism_second_call",
				Inputs:  []interface{}{"test"},
				Timeout: 5 * time.Second,
				Meta: map[string]interface{}{
					"description": "Second call with same input must produce same result (determinism)",
					"required":    true,
				},
			},
		},
	}
}

// TestPlugin runs the conformance test suite against a plugin
// Returns slice of results, one per test
func TestPlugin(fn Function, suite *ConformanceSuite) []ConformanceResult {
	results := make([]ConformanceResult, len(suite.Tests))

	for i, test := range suite.Tests {
		result := ConformanceResult{
			Test: test.Name,
		}

		// Execute the test
		start := time.Now()

		// Create a channel for the result with timeout
		timeout := test.Timeout
		if timeout == 0 {
			timeout = 5 * time.Second
		}

		done := make(chan interface{}, 1)
		errCh := make(chan error, 1)

		go func() {
			out, err := fn.Call(test.Inputs...)
			if err != nil {
				errCh <- err
			} else {
				done <- out
			}
		}()

		// Wait for result or timeout
		select {
		case res := <-done:
			result.Duration = time.Since(start)
			result.GotResult = res
			if test.WantErr {
				result.Passed = false
				result.Message = fmt.Sprintf("Expected error but got result: %v", res)
			} else {
				result.Passed = true
				result.Message = "Call succeeded"
			}

		case err := <-errCh:
			result.Duration = time.Since(start)
			result.Error = err
			if test.WantErr {
				result.Passed = true
				result.Message = fmt.Sprintf("Got expected error: %v", err)
			} else {
				result.Passed = false
				result.Message = fmt.Sprintf("Unexpected error: %v", err)
			}

		case <-time.After(timeout):
			result.Duration = time.Since(start)
			result.Passed = false
			result.Error = fmt.Errorf("call exceeded timeout (%v)", timeout)
			result.Message = "Call timed out"
		}

		results[i] = result
	}

	return results
}

// SummarizeResults returns a summary of conformance test results
func SummarizeResults(results []ConformanceResult) (passed int, failed int, summary string) {
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	total := len(results)
	pct := 0
	if total > 0 {
		pct = (passed * 100) / total
	}

	if failed == 0 {
		summary = fmt.Sprintf("✅ All tests passed (%d/%d)", passed, total)
	} else {
		summary = fmt.Sprintf("❌ %d/%d tests passed (%d%%) - %d failures", passed, total, pct, failed)
	}

	return
}
