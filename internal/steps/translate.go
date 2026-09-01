package steps

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// TranslateStep evaluates a JSONata expression against a message and returns
// a new message with the evaluated result as its body.
// The JSONata context contains the message body, headers, and metadata.
type TranslateStep struct {
	evaluator *expr.Evaluator
}

// NewTranslateStep creates a new translate step from a JSONata expression string.
// The expression is compiled during initialization; compilation errors are returned immediately.
func NewTranslateStep(exprStr string) (*TranslateStep, error) {
	evaluator, err := expr.CompileExpression(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSONata expression: %w", err)
	}

	return &TranslateStep{
		evaluator: evaluator,
	}, nil
}

// Execute evaluates the JSONata expression against the message and returns
// a new message with the result as its body. The headers and metadata are
// preserved from the original message.
// If the expression evaluation fails, an error is returned.
func (t *TranslateStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, fmt.Errorf("translate step: message is nil")
	}

	// Build the context object for JSONata evaluation
	// Contains body, headers, and metadata from the message
	context := map[string]interface{}{
		"body":     msg.Body,
		"headers":  msg.Headers,
		"metadata": msg.Metadata,
	}

	// Evaluate the expression
	result, err := t.evaluator.Eval(context)
	if err != nil {
		return nil, fmt.Errorf("translate step: expression evaluation failed: %w", err)
	}

	// Create a new message with the result as the body
	// Copy the original message structure to preserve headers
	newMsg := msg.Copy()
	newMsg.Body = result

	return newMsg, nil
}
