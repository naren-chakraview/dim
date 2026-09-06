package steps

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// SplitStep splits one message into many by evaluating an array expression (M2.2.2, Phase 2)
type SplitStep struct {
	splitEvaluator   *expr.Evaluator // Compiled split expression
	outputEvaluator  *expr.Evaluator // Compiled output transformation (optional)
	onNonArray       string          // "error_path" or "skip"
}

// SplitSpec defines splitter configuration
type SplitSpec struct {
	Expr        string `yaml:"expr" json:"expr"`                                 // JSONata expression producing array (required)
	OutputExpr  string `yaml:"output_expr,omitempty" json:"output_expr,omitempty"` // Optional output transformation
	OnNonArray  string `yaml:"on_non_array,omitempty" json:"on_non_array,omitempty"` // "error_path" (default) or "skip"
	OnError     string `yaml:"on_error,omitempty" json:"on_error,omitempty"`     // Error handling
}

// NewSplitStep creates a splitter step
func NewSplitStep(spec *SplitSpec) (*SplitStep, error) {
	if spec.Expr == "" {
		return nil, fmt.Errorf("expr is required")
	}

	if spec.OnNonArray == "" {
		spec.OnNonArray = "error_path"
	}

	if spec.OnNonArray != "error_path" && spec.OnNonArray != "skip" {
		return nil, fmt.Errorf("on_non_array must be 'error_path' or 'skip', got %s", spec.OnNonArray)
	}

	// Compile split expression
	splitEval, err := expr.CompileExpression(spec.Expr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile split expression: %w", err)
	}

	// Compile optional output expression
	var outputEval *expr.Evaluator
	if spec.OutputExpr != "" {
		var err error
		outputEval, err = expr.CompileExpression(spec.OutputExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to compile output_expr expression: %w", err)
		}
	}

	return &SplitStep{
		splitEvaluator:  splitEval,
		outputEvaluator: outputEval,
		onNonArray:      spec.OnNonArray,
	}, nil
}

// Execute processes a message through the splitter
// Returns a list of output messages (one per array element)
// Note: In a real pipeline, these would be queued to output channels
func (ss *SplitStep) Execute(ctx context.Context, msg *engine.Message) ([]*engine.Message, error) {
	// Build the context object for expression evaluation
	// Contains body, headers, and metadata from the message
	evalContext := map[string]interface{}{
		"body":     msg.Body,
		"headers":  msg.Headers,
		"metadata": msg.Metadata,
	}

	// Evaluate split expression
	result, err := ss.splitEvaluator.Eval(evalContext)
	if err != nil {
		return nil, fmt.Errorf("split expression evaluation failed: %w", err)
	}

	// Handle non-array results
	if result == nil {
		if ss.onNonArray == "error_path" {
			return nil, fmt.Errorf("split expression evaluated to null")
		}
		// "skip" — no outputs
		return []*engine.Message{}, nil
	}

	// Convert to slice if it's an array
	arrayElements, ok := result.([]interface{})
	if !ok {
		if ss.onNonArray == "error_path" {
			return nil, fmt.Errorf("split expression must produce array, got %T", result)
		}
		// "skip" — no outputs
		return []*engine.Message{}, nil
	}

	// Empty array produces no outputs
	if len(arrayElements) == 0 {
		return []*engine.Message{}, nil
	}

	// Create output message for each array element
	outputs := make([]*engine.Message, 0, len(arrayElements))

	for index, element := range arrayElements {
		// Apply optional output transformation
		outputValue := element
		if ss.outputEvaluator != nil {
			// Build context for output transformation (element is the body, original context preserved)
			outputContext := map[string]interface{}{
				"body":     element,
				"headers":  msg.Headers,
				"metadata": msg.Metadata,
			}
			transformed, err := ss.outputEvaluator.Eval(outputContext)
			if err != nil {
				return nil, fmt.Errorf("output_expr transformation failed at index %d: %w", index, err)
			}
			outputValue = transformed
		}

		// Create output message
		outMsg := engine.NewMessage(outputValue, msg.Metadata.Route, msg.Metadata.RouteVersion)

		// Preserve input metadata
		outMsg.Metadata.CorrelationID = msg.Metadata.CorrelationID // Share correlation ID
		outMsg.Metadata.Principal = msg.Metadata.Principal
		outMsg.Metadata.ContractVersion = msg.Metadata.ContractVersion

		// Add splitter facet to metadata
		outMsg.Metadata.SplitterFacet = &SplitterFacet{
			SplitExpr:     "expr", // In production, would store the actual expression string
			TotalElements: len(arrayElements),
			ElementIndex:  index,
			SplitTrigger:  "array_produced",
		}

		// Preserve input subject ID
		if msg.Metadata.Principal != nil {
			outMsg.Metadata.Principal = msg.Metadata.Principal
		}

		outputs = append(outputs, outMsg)
	}

	return outputs, nil
}

// SplitterFacet captures split metadata for lineage (inverse of AggregatorFacet)
type SplitterFacet struct {
	SplitExpr     string `json:"split_expr"`
	TotalElements int    `json:"total_elements"`
	ElementIndex  int    `json:"element_index"`
	SplitTrigger  string `json:"split_trigger"` // "array_produced" or "empty_array"
}
