package testing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// Fixture represents a single test case for the fixture runner
type Fixture struct {
	// Name is the test name/description
	Name string `yaml:"name"`

	// Input is the message body to send to the pipeline
	Input map[string]interface{} `yaml:"input"`

	// ExpectedOutput is the expected message body after processing
	// If nil, no output check is performed (used for dropped/errored messages)
	ExpectedOutput map[string]interface{} `yaml:"expected_output"`

	// ExpectedRoute is the expected route name (set by route step)
	// If empty string, route check is skipped
	ExpectedRoute string `yaml:"expected_route"`

	// ExpectedDropped verifies the message was filtered out (returns nil)
	ExpectedDropped bool `yaml:"expected_dropped"`

	// ExpectedError expects an error to occur during processing
	ExpectedError bool `yaml:"expected_error"`

	// ExpectedErrorType expects a specific error type (optional string match)
	ExpectedErrorType string `yaml:"expected_error_type"`

	// Timeout in milliseconds for this fixture (default: 30000)
	TimeoutMs int `yaml:"timeout_ms"`
}

// FixtureResult contains the outcome of running a fixture
type FixtureResult struct {
	// Name of the fixture
	Name string

	// Passed indicates if the fixture passed all checks
	Passed bool

	// ActualOutput is the output message received (may be nil if dropped)
	ActualOutput map[string]interface{}

	// ActualRoute is the route metadata from output
	ActualRoute string

	// ActualError is any error that occurred
	ActualError error

	// ActualErrorType is the string representation of the error type
	ActualErrorType string

	// DurationMs is the time taken to run the fixture
	DurationMs int64

	// FailureReason explains why the fixture failed (if Passed is false)
	FailureReason string
}

// FixtureRunner executes fixtures against a middleware pipeline
type FixtureRunner struct {
	executor *engine.Executor
	timeout  time.Duration
}

// NewFixtureRunner creates a new fixture runner
// executor: the middleware executor to test
// timeoutMs: default timeout for fixtures (milliseconds)
func NewFixtureRunner(executor *engine.Executor, timeoutMs int) *FixtureRunner {
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second // Default 30 seconds
	}

	return &FixtureRunner{
		executor: executor,
		timeout:  timeout,
	}
}

// RunFixture executes a single fixture and returns the result
func (fr *FixtureRunner) RunFixture(ctx context.Context, fixture *Fixture) *FixtureResult {
	startTime := time.Now()
	result := &FixtureResult{
		Name: fixture.Name,
	}

	// Determine timeout for this fixture
	timeout := fr.timeout
	if fixture.TimeoutMs > 0 {
		timeout = time.Duration(fixture.TimeoutMs) * time.Millisecond
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create the input message
	msg := engine.NewMessage(fixture.Input, "", "v1")

	// Process the message
	output, err := fr.executor.ProcessMessage(ctx, msg)

	// Record duration
	result.DurationMs = time.Since(startTime).Milliseconds()

	// Handle timeout
	if errors.Is(err, context.DeadlineExceeded) {
		result.ActualError = fmt.Errorf("timeout after %v", timeout)
		result.ActualErrorType = "DeadlineExceeded"
		result.FailureReason = fmt.Sprintf("fixture timeout after %v", timeout)
		return result
	}

	// Handle errors
	if err != nil {
		result.ActualError = err
		result.ActualErrorType = fmt.Sprintf("%T", err)

		// Check if error was expected
		if fixture.ExpectedError {
			if fixture.ExpectedErrorType != "" {
				// Check error type match
				if errTypeMatches(err, fixture.ExpectedErrorType) {
					result.Passed = true
					return result
				}
				result.FailureReason = fmt.Sprintf("error type mismatch: expected %s, got %s",
					fixture.ExpectedErrorType, result.ActualErrorType)
				return result
			}
			// Any error is acceptable
			result.Passed = true
			return result
		}

		// Error was not expected
		result.FailureReason = fmt.Sprintf("unexpected error: %v", err)
		return result
	}

	// No error occurred
	if fixture.ExpectedError {
		result.FailureReason = fmt.Sprintf("expected error but got none")
		return result
	}

	// Check if message was dropped (output is nil)
	if output == nil {
		if fixture.ExpectedDropped {
			result.Passed = true
			return result
		}

		result.FailureReason = "message was dropped but expected_dropped is false"
		return result
	}

	// Message was not dropped but expected to be
	if fixture.ExpectedDropped {
		result.FailureReason = "message was not dropped but expected_dropped is true"
		return result
	}

	// Extract output body
	resultBody := output.Body
	if m, ok := resultBody.(map[string]interface{}); ok {
		result.ActualOutput = m
	} else {
		// Try to convert to map
		jsonBytes, err := json.Marshal(resultBody)
		if err == nil {
			var m map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &m); err == nil {
				result.ActualOutput = m
			}
		}
	}

	// Extract route metadata
	result.ActualRoute = output.Metadata.Route

	// Verify expected output if specified
	if fixture.ExpectedOutput != nil {
		if !deepEqual(fixture.ExpectedOutput, result.ActualOutput) {
			result.FailureReason = fmt.Sprintf("output mismatch: expected %v, got %v",
				fixture.ExpectedOutput, result.ActualOutput)
			return result
		}
	}

	// Verify expected route if specified
	if fixture.ExpectedRoute != "" {
		if fixture.ExpectedRoute != result.ActualRoute {
			result.FailureReason = fmt.Sprintf("route mismatch: expected %s, got %s",
				fixture.ExpectedRoute, result.ActualRoute)
			return result
		}
	}

	// All checks passed
	result.Passed = true
	return result
}

// RunFixtures runs all fixtures in batch and returns summary
func (fr *FixtureRunner) RunFixtures(ctx context.Context, fixtures []*Fixture) []*FixtureResult {
	var results []*FixtureResult

	for _, fixture := range fixtures {
		result := fr.RunFixture(ctx, fixture)
		results = append(results, result)
	}

	return results
}

// deepEqual compares two values recursively for deep equality
// Handles maps, slices, and basic types
func deepEqual(expected, actual interface{}) bool {
	if expected == nil && actual == nil {
		return true
	}

	if expected == nil || actual == nil {
		return false
	}

	// Convert to JSON and compare
	// This handles type coercion and deep comparison
	expJSON, err1 := json.Marshal(expected)
	actJSON, err2 := json.Marshal(actual)

	if err1 != nil || err2 != nil {
		// Fallback to reflect.DeepEqual
		return reflect.DeepEqual(expected, actual)
	}

	return string(expJSON) == string(actJSON)
}

// errTypeMatches checks if an error matches a type name
func errTypeMatches(err error, typeName string) bool {
	if err == nil {
		return false
	}

	actualType := fmt.Sprintf("%T", err)
	return actualType == typeName || actualType == "*"+typeName
}

// SummarizeResults returns a summary of test results
func SummarizeResults(results []*FixtureResult) (passed, failed, errors int, details string) {
	passed = 0
	failed = 0
	errors = 0

	for _, result := range results {
		if result.Passed {
			passed++
		} else if result.ActualError != nil {
			errors++
		} else {
			failed++
		}
	}

	summary := fmt.Sprintf("Total: %d fixtures | Passed: %d | Failed: %d | Errors: %d",
		len(results), passed, failed, errors)

	return passed, failed, errors, summary
}
