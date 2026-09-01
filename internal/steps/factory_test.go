package steps

import (
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
)

// TestBuildStepsFromSpecEmptySteps tests building with no steps
func TestBuildStepsFromSpecEmptySteps(t *testing.T) {
	steps, stepNames, err := BuildStepsFromSpec([]config.StepSpec{})
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 0 {
		t.Fatalf("expected 0 steps, got %d", len(steps))
	}
	if len(stepNames) != 0 {
		t.Fatalf("expected 0 stepNames, got %d", len(stepNames))
	}
}

// TestBuildStepsFromSpecFilter tests building with filter step
func TestBuildStepsFromSpecFilter(t *testing.T) {
	specs := []config.StepSpec{
		{
			Filter: &config.FilterSpec{
				Expr: "body.amount > 100",
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 {
		t.Fatalf("expected 1 stepName, got %d", len(stepNames))
	}
	if stepNames[0] != "filter" {
		t.Fatalf("expected step name 'filter', got '%s'", stepNames[0])
	}

	// Verify the step is a FilterStep
	_, ok := steps[0].(*FilterStep)
	if !ok {
		t.Fatalf("expected FilterStep, got %T", steps[0])
	}
}

// TestBuildStepsFromSpecTranslate tests building with translate step
func TestBuildStepsFromSpecTranslate(t *testing.T) {
	specs := []config.StepSpec{
		{
			Translate: &config.TranslateSpec{
				Expr: "body.amount * 2",
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 {
		t.Fatalf("expected 1 stepName, got %d", len(stepNames))
	}
	if stepNames[0] != "translate" {
		t.Fatalf("expected step name 'translate', got '%s'", stepNames[0])
	}

	// Verify the step is a TranslateStep
	_, ok := steps[0].(*TranslateStep)
	if !ok {
		t.Fatalf("expected TranslateStep, got %T", steps[0])
	}
}

// TestBuildStepsFromSpecMultiple tests building with multiple steps in sequence
func TestBuildStepsFromSpecMultiple(t *testing.T) {
	specs := []config.StepSpec{
		{
			Filter: &config.FilterSpec{
				Expr: "body.amount > 100",
			},
		},
		{
			Translate: &config.TranslateSpec{
				Expr: "body.amount * 2",
			},
		},
		{
			Filter: &config.FilterSpec{
				Expr: "body.amount < 1000",
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	if len(stepNames) != 3 {
		t.Fatalf("expected 3 stepNames, got %d", len(stepNames))
	}

	expectedNames := []string{"filter", "translate", "filter"}
	for i, expected := range expectedNames {
		if stepNames[i] != expected {
			t.Fatalf("expected step %d name '%s', got '%s'", i, expected, stepNames[i])
		}
	}
}

// TestBuildStepsFromSpecNoStepTypeError tests error when no step type is specified
func TestBuildStepsFromSpecNoStepTypeError(t *testing.T) {
	specs := []config.StepSpec{
		{}, // No step type specified
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error when no step type specified")
	}
}

// TestBuildStepsFromSpecMultipleStepTypesError tests error when multiple step types specified
func TestBuildStepsFromSpecMultipleStepTypesError(t *testing.T) {
	specs := []config.StepSpec{
		{
			Filter: &config.FilterSpec{
				Expr: "body.amount > 100",
			},
			Translate: &config.TranslateSpec{
				Expr: "body.amount * 2",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error when multiple step types specified")
	}
}

// TestBuildStepsFromSpecRouteStepNotImplemented tests error for route step
func TestBuildStepsFromSpecRouteStepNotImplemented(t *testing.T) {
	specs := []config.StepSpec{
		{
			Route: &config.RouteStepSpec{
				Expr: "body.type",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error for route step not implemented")
	}
}

// TestBuildStepsFromSpecWiretapStepNotImplemented tests error for wiretap step
func TestBuildStepsFromSpecWiretapStepNotImplemented(t *testing.T) {
	specs := []config.StepSpec{
		{
			Wiretap: &config.WiretapSpec{
				Sink: "other",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error for wiretap step not implemented")
	}
}

// TestBuildStepsFromSpecIdempotentStepNotImplemented tests error for idempotent step
func TestBuildStepsFromSpecIdempotentStepNotImplemented(t *testing.T) {
	specs := []config.StepSpec{
		{
			Idempotent: &config.IdempotentSpec{
				KeyExpr: "body.id",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error for idempotent step not implemented")
	}
}

// TestBuildStepsFromSpecAuthorizeStepNotImplemented tests error for authorize step
func TestBuildStepsFromSpecAuthorizeStepNotImplemented(t *testing.T) {
	specs := []config.StepSpec{
		{
			Authorize: &config.AuthorizeSpec{
				Mode: "rbac",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error for authorize step not implemented")
	}
}

// TestBuildStepsFromSpecInvalidFilterExpression tests error for invalid filter expression
func TestBuildStepsFromSpecInvalidFilterExpression(t *testing.T) {
	specs := []config.StepSpec{
		{
			Filter: &config.FilterSpec{
				Expr: "invalid [ syntax",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error for invalid filter expression")
	}
}

// TestBuildStepsFromSpecInvalidTranslateExpression tests error for invalid translate expression
func TestBuildStepsFromSpecInvalidTranslateExpression(t *testing.T) {
	specs := []config.StepSpec{
		{
			Translate: &config.TranslateSpec{
				Expr: "invalid [ syntax",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs)
	if err == nil {
		t.Fatal("expected error for invalid translate expression")
	}
}
