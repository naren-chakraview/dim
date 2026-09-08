package steps

import (
	"testing"

	"github.com/naren-chakraview/dim/internal/config"
)

// TestBuildStepsFromSpecEmptySteps tests building with no steps
func TestBuildStepsFromSpecEmptySteps(t *testing.T) {
	steps, stepNames, err := BuildStepsFromSpec([]config.StepSpec{}, nil, "", nil)
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

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
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

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
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

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
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

	_, _, err := BuildStepsFromSpec(specs, nil, "", nil)
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

	_, _, err := BuildStepsFromSpec(specs, nil, "", nil)
	if err == nil {
		t.Fatal("expected error when multiple step types specified")
	}
}

// TestBuildStepsFromSpecRoute tests building route step successfully
func TestBuildStepsFromSpecRoute(t *testing.T) {
	specs := []config.StepSpec{
		{
			Route: &config.RouteStepSpec{
				Expr: "body.type",
				Cases: map[string]config.RouteCase{
					"premium": {Target: "sink-premium"},
					"standard": {Target: "sink-standard"},
				},
				Default: "sink-default",
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 {
		t.Fatalf("expected 1 stepName, got %d", len(stepNames))
	}
	if stepNames[0] != "route" {
		t.Fatalf("expected step name 'route', got '%s'", stepNames[0])
	}

	// Verify the step is a RouteStep
	_, ok := steps[0].(*RouteStep)
	if !ok {
		t.Fatalf("expected RouteStep, got %T", steps[0])
	}
}

// TestBuildStepsFromSpecWiretapStep tests wiretap step (M0.2.4+)
func TestBuildStepsFromSpecWiretapStep(t *testing.T) {
	specs := []config.StepSpec{
		{
			Wiretap: &config.WiretapSpec{
				Sink: "audit_log",
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 {
		t.Fatalf("expected 1 stepName, got %d", len(stepNames))
	}
	if stepNames[0] != "wiretap" {
		t.Fatalf("expected step name 'wiretap', got '%s'", stepNames[0])
	}

	// Verify the step is a WiretapStep
	wiretapStep, ok := steps[0].(*WiretapStep)
	if !ok {
		t.Fatalf("expected WiretapStep, got %T", steps[0])
	}

	if wiretapStep.sinkName != "audit_log" {
		t.Fatalf("expected sink name 'audit_log', got %q", wiretapStep.sinkName)
	}
}

// TestBuildStepsFromSpecIdempotentStep tests building idempotent step (M3.1)
func TestBuildStepsFromSpecIdempotentStep(t *testing.T) {
	specs := []config.StepSpec{
		{
			Idempotent: &config.IdempotentSpec{
				KeyExpr: "body.id",
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 || stepNames[0] != "idempotent" {
		t.Fatalf("expected stepNames=['idempotent'], got %v", stepNames)
	}
}

// TestBuildStepsFromSpecAuthorizeStepRBAC tests building authorize step with RBAC mode
func TestBuildStepsFromSpecAuthorizeStepRBAC(t *testing.T) {
	specs := []config.StepSpec{
		{
			Authorize: &config.AuthorizeSpec{
				Mode:         "rbac",
				RequireRoles: []string{"admin", "editor"},
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 {
		t.Fatalf("expected 1 stepName, got %d", len(stepNames))
	}
	if stepNames[0] != "authorize" {
		t.Fatalf("expected step name 'authorize', got '%s'", stepNames[0])
	}

	// Verify the step is an AuthorizeStep
	authorizeStep, ok := steps[0].(*AuthorizeStep)
	if !ok {
		t.Fatalf("expected AuthorizeStep, got %T", steps[0])
	}

	if authorizeStep.mode != "rbac" {
		t.Fatalf("expected mode 'rbac', got '%s'", authorizeStep.mode)
	}
}

// TestBuildStepsFromSpecAuthorizeStepABAC tests building authorize step with ABAC mode
func TestBuildStepsFromSpecAuthorizeStepABAC(t *testing.T) {
	specs := []config.StepSpec{
		{
			Authorize: &config.AuthorizeSpec{
				Mode: "abac",
				Expr: "principal.subject = 'admin'",
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs, nil, "", nil)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 {
		t.Fatalf("expected 1 stepName, got %d", len(stepNames))
	}
	if stepNames[0] != "authorize" {
		t.Fatalf("expected step name 'authorize', got '%s'", stepNames[0])
	}

	// Verify the step is an AuthorizeStep
	authorizeStep, ok := steps[0].(*AuthorizeStep)
	if !ok {
		t.Fatalf("expected AuthorizeStep, got %T", steps[0])
	}

	if authorizeStep.mode != "abac" {
		t.Fatalf("expected mode 'abac', got '%s'", authorizeStep.mode)
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

	_, _, err := BuildStepsFromSpec(specs, nil, "", nil)
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

	_, _, err := BuildStepsFromSpec(specs, nil, "", nil)
	if err == nil {
		t.Fatal("expected error for invalid translate expression")
	}
}

// TestBuildStepsFromSpecContractStep tests building contract step (M0.3.5)
func TestBuildStepsFromSpecContractStep(t *testing.T) {
	// Setup contract store with a contract
	contractStore := config.NewContractStore()
	schema := map[string]interface{}{"type": "object"}
	contract := config.ContractSpec{
		ID:      "test-contract",
		Version: "1.0.0",
		Schema:  schema,
	}
	contractStore.LoadContracts("test-route", []config.ContractSpec{contract})

	specs := []config.StepSpec{
		{
			Contract: &config.ContractStepSpec{
				ID:     "test-contract",
				Strict: false,
			},
		},
	}

	steps, stepNames, err := BuildStepsFromSpec(specs, contractStore, "test-route", nil)
	if err != nil {
		t.Fatalf("BuildStepsFromSpec failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if len(stepNames) != 1 {
		t.Fatalf("expected 1 stepName, got %d", len(stepNames))
	}
	if stepNames[0] != "contract" {
		t.Fatalf("expected step name 'contract', got '%s'", stepNames[0])
	}

	// Verify the step is a ContractStep
	_, ok := steps[0].(*ContractStep)
	if !ok {
		t.Fatalf("expected ContractStep, got %T", steps[0])
	}
}

// TestBuildStepsFromSpecContractStepMissingContract tests error when contract not found
func TestBuildStepsFromSpecContractStepMissingContract(t *testing.T) {
	contractStore := config.NewContractStore()

	specs := []config.StepSpec{
		{
			Contract: &config.ContractStepSpec{
				ID: "nonexistent-contract",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs, contractStore, "test-route", nil)
	if err == nil {
		t.Fatal("expected error when contract not found")
	}
}

// TestBuildStepsFromSpecContractStepNoStore tests error when contract store is nil
func TestBuildStepsFromSpecContractStepNoStore(t *testing.T) {
	specs := []config.StepSpec{
		{
			Contract: &config.ContractStepSpec{
				ID: "any-contract",
			},
		},
	}

	_, _, err := BuildStepsFromSpec(specs, nil, "test-route", nil)
	if err == nil {
		t.Fatal("expected error when contract store is nil")
	}
}
