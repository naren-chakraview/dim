package steps

import (
	"fmt"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// BuildStepsFromSpec creates step instances from a slice of step specifications.
// Returns (steps, stepNames, error).
// stepNames is a parallel slice mapping step indices to their type names (e.g., "filter", "translate").
//
// Supported step types in Phase 0:
// - filter: Boolean predicate to drop messages
// - translate: JSONata expression to transform message body
//
// Unsupported step types return error "not implemented in Phase 0":
// - route, wiretap, idempotent, authorize
func BuildStepsFromSpec(stepSpecs []config.StepSpec) ([]engine.Step, []string, error) {
	var steps []engine.Step
	var stepNames []string

	for i, spec := range stepSpecs {
		// Count how many step types are defined (should be exactly 1)
		stepTypeCount := 0

		if spec.Filter != nil {
			stepTypeCount++
		}
		if spec.Translate != nil {
			stepTypeCount++
		}
		if spec.Route != nil {
			stepTypeCount++
		}
		if spec.Wiretap != nil {
			stepTypeCount++
		}
		if spec.Idempotent != nil {
			stepTypeCount++
		}
		if spec.Authorize != nil {
			stepTypeCount++
		}

		if stepTypeCount == 0 {
			return nil, nil, fmt.Errorf("step %d: no step type specified", i)
		}
		if stepTypeCount > 1 {
			return nil, nil, fmt.Errorf("step %d: multiple step types specified (exactly one required)", i)
		}

		// Build the step based on type
		var step engine.Step
		var err error

		switch {
		case spec.Filter != nil:
			step, err = NewFilterStep(spec.Filter.Expr)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (filter): %w", i, err)
			}
			stepNames = append(stepNames, "filter")

		case spec.Translate != nil:
			step, err = NewTranslateStep(spec.Translate.Expr)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (translate): %w", i, err)
			}
			stepNames = append(stepNames, "translate")

		case spec.Route != nil:
			return nil, nil, fmt.Errorf("step %d: route step not implemented in Phase 0", i)
		case spec.Wiretap != nil:
			return nil, nil, fmt.Errorf("step %d: wiretap step not implemented in Phase 0", i)
		case spec.Idempotent != nil:
			return nil, nil, fmt.Errorf("step %d: idempotent step not implemented in Phase 0", i)
		case spec.Authorize != nil:
			return nil, nil, fmt.Errorf("step %d: authorize step not implemented in Phase 0", i)
		}

		steps = append(steps, step)
	}

	return steps, stepNames, nil
}
