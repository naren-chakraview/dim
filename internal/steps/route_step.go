package steps

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// RouteStep implements the content-based router in the pipeline.
// It evaluates a JSONata expression against the message to determine
// which target route/sink the message should be sent to.
// The expression result is mapped to a case name, which is then looked up
// in a cases map to get the target route/sink name.
// If the case is not found, an optional default target is used.
// The message metadata.Route is updated with the target route name.
type RouteStep struct {
	evaluator      *expr.Evaluator
	cases          map[string]string // case name → target route/sink name
	defaultTarget  string             // fallback target if no case matches
}

// NewRouteStep creates a new route step from a JSONata expression and cases map.
// The expression is compiled during initialization; compilation errors are returned immediately.
// The cases map must not be empty.
// Routing errors are permanent (not retryable).
func NewRouteStep(exprStr string, cases map[string]string, defaultTarget string) (*RouteStep, error) {
	// Validate inputs
	if exprStr == "" {
		return nil, fmt.Errorf("route expression cannot be empty")
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("route cases cannot be empty")
	}

	// Compile the expression
	evaluator, err := expr.CompileExpression(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile route expression: %w", err)
	}

	return &RouteStep{
		evaluator:     evaluator,
		cases:         cases,
		defaultTarget: defaultTarget,
	}, nil
}

// Execute evaluates the route expression against the message and routes it to the appropriate target.
// Returns:
// - (msg with updated Route metadata, nil) if routing succeeds
// - (nil, error) if routing fails or no matching case with no default
func (r *RouteStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, fmt.Errorf("route step: message is nil")
	}

	// Build the context object for JSONata evaluation
	// Contains body, headers, and metadata from the message
	context := map[string]interface{}{
		"body":     msg.Body,
		"headers":  msg.Headers,
		"metadata": msg.Metadata,
	}

	// Evaluate the expression to get the case name
	result, err := r.evaluator.Eval(context)
	if err != nil {
		return nil, fmt.Errorf("route step: expression evaluation failed: %w", err)
	}

	// Convert result to string (case name)
	caseName := resultToString(result)

	// Look up the case in the cases map
	target, exists := r.cases[caseName]
	if !exists {
		// Case not found; use default target if available
		if r.defaultTarget != "" {
			target = r.defaultTarget
		} else {
			// No matching case and no default
			return nil, fmt.Errorf("route step: no matching case %q and no default target", caseName)
		}
	}

	// Update the message metadata with the target route
	newMsg := msg.Copy()
	newMsg.Metadata.Route = target

	return newMsg, nil
}

// resultToString converts a JSONata evaluation result to a string.
// Handles various types: string, number, boolean, nil, etc.
func resultToString(result interface{}) string {
	switch v := result.(type) {
	case string:
		return v
	case float64:
		// Format number as integer if it's a whole number, otherwise as float
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		// For complex types, use string conversion
		return fmt.Sprintf("%v", v)
	}
}
