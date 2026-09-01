package steps

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// FilterStep implements the filter step in the pipeline.
// It evaluates a JSONata boolean predicate against the message and drops
// messages that don't match the predicate.
type FilterStep struct {
	evaluator *expr.Evaluator
}

// NewFilterStep creates a new filter step from a JSONata boolean expression.
// The expression is compiled at construction time to catch syntax errors early.
func NewFilterStep(exprStr string) (*FilterStep, error) {
	evaluator, err := expr.CompileExpression(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile filter expression: %w", err)
	}

	return &FilterStep{
		evaluator: evaluator,
	}, nil
}

// Execute evaluates the filter predicate against the message.
// Returns:
// - (msg, nil) if the predicate evaluates to true
// - (nil, nil) if the predicate evaluates to false (signals message drop)
// - (nil, error) if the predicate evaluation fails
func (f *FilterStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, nil
	}

	// Build the context object with body, headers, and metadata.
	// This is what the JSONata expression will evaluate against.
	context := map[string]interface{}{
		"body":     msg.Body,
		"headers":  msg.Headers,
		"metadata": msg.Metadata,
	}

	// Evaluate the predicate
	result, err := f.evaluator.Eval(context)
	if err != nil {
		return nil, fmt.Errorf("filter predicate evaluation failed: %w", err)
	}

	// Convert the result to a boolean.
	// JSONata may return different types (bool, float64, etc.), so we need to handle type assertions.
	predicate, ok := result.(bool)
	if !ok {
		// Try to interpret the result as truthy/falsy
		switch v := result.(type) {
		case float64:
			// In JSONata, numeric 0 is falsy, other numbers are truthy
			predicate = v != 0
		case string:
			// In JSONata, empty string is falsy, non-empty is truthy
			predicate = v != ""
		case nil:
			// null is falsy
			predicate = false
		default:
			// Other types (arrays, objects) are truthy
			predicate = true
		}
	}

	// If the predicate is false, return nil to signal that this message should be dropped
	if !predicate {
		return nil, nil
	}

	// If the predicate is true, return the message unchanged
	return msg, nil
}
