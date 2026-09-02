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
// Supported step types:
// - filter: Boolean predicate to drop messages (Phase 0+)
// - translate: JSONata expression to transform message body (Phase 0+)
// - route: Content-based router (M0.2.3+)
// - wiretap: Copy to secondary sink (M0.2.4+)
// - authorize: RBAC/ABAC policy enforcement (M0.2.6+)
//
// Unsupported step types return error "not implemented in Phase 0":
// - idempotent (M0.2.5)
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
			// Convert RouteCase objects to a map[string]string
			casesMap := make(map[string]string)
			for caseName, routeCase := range spec.Route.Cases {
				casesMap[caseName] = routeCase.Target
			}

			step, err = NewRouteStep(spec.Route.Expr, casesMap, spec.Route.Default)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (route): %w", i, err)
			}
			stepNames = append(stepNames, "route")
		case spec.Wiretap != nil:
			step, err = NewWiretapStep(spec.Wiretap.Sink)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (wiretap): %w", i, err)
			}
			stepNames = append(stepNames, "wiretap")
		case spec.Idempotent != nil:
			return nil, nil, fmt.Errorf("step %d: idempotent step not implemented in Phase 0", i)
		case spec.Authorize != nil:
			step, err = NewAuthorizeStep(spec.Authorize.Mode, spec.Authorize.RequireRoles, spec.Authorize.Expr)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (authorize): %w", i, err)
			}
			stepNames = append(stepNames, "authorize")
		}

		steps = append(steps, step)
	}

	return steps, stepNames, nil
}
