package expr

import (
	"github.com/blues/jsonata-go"
)

// Evaluator wraps a JSONata expression for evaluation
type Evaluator struct {
	expr     *jsonata.Expr
	registry *Registry
}

// CompileExpression compiles a JSONata string into an Evaluator
func CompileExpression(exprStr string) (*Evaluator, error) {
	expr, err := jsonata.Compile(exprStr)
	if err != nil {
		return nil, err
	}
	return &Evaluator{
		expr:     expr,
		registry: nil,
	}, nil
}

// CompileExpressionWithRegistry compiles a JSONata string with a custom functions registry
// Note: Direct JSONata integration of custom functions is deferred to R4/R5 (plugin/WASM runtimes)
// For now, the registry is available for steps to query and use
func CompileExpressionWithRegistry(exprStr string, registry *Registry) (*Evaluator, error) {
	expr, err := jsonata.Compile(exprStr)
	if err != nil {
		return nil, err
	}

	return &Evaluator{
		expr:     expr,
		registry: registry,
	}, nil
}

// GetRegistry returns the functions registry for this evaluator (if any)
func (e *Evaluator) GetRegistry() *Registry {
	return e.registry
}

// Eval evaluates the expression against input data
func (e *Evaluator) Eval(input interface{}) (interface{}, error) {
	return e.expr.Eval(input)
}
