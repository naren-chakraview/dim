package expr

import (
	"github.com/blues/jsonata-go"
)

// Evaluator wraps a JSONata expression for evaluation
type Evaluator struct {
	expr *jsonata.Expr
}

// CompileExpression compiles a JSONata string into an Evaluator
func CompileExpression(exprStr string) (*Evaluator, error) {
	expr, err := jsonata.Compile(exprStr)
	if err != nil {
		return nil, err
	}
	return &Evaluator{expr: expr}, nil
}

// Eval evaluates the expression against input data
func (e *Evaluator) Eval(input interface{}) (interface{}, error) {
	return e.expr.Eval(input)
}
