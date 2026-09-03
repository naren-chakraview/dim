package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAuthValidationMissingAuthWarnMode tests that missing auth logs warning in warn mode
func TestAuthValidationMissingAuthWarnMode(t *testing.T) {
	yaml := `
version: 1
sources:
  in:
    type: http
sinks:
  out:
    type: file
    path: /tmp/out
routes:
  test:
    from: in
    error_path:
      target: out
    steps:
      - translate:
          expr: '{ message: body }'
`

	tmpFile := filepath.Join(t.TempDir(), "test_no_auth.yaml")
	if err := os.WriteFile(tmpFile, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Create config manually to bypass schema validation
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{
			"test": {
				From: "in",
				Auth: nil, // Missing auth
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{
					{
						Translate: &TranslateSpec{
							Expr: "{ message: body }",
						},
					},
				},
			},
		},
	}

	// Should not error in warn mode
	err := ValidateAuthDeclarations(config, AuthValidationWarn)
	if err != nil {
		t.Errorf("expected no error in warn mode, got %v", err)
	}
}

// TestAuthValidationMissingAuthStrictMode tests that missing auth fails in strict mode
func TestAuthValidationMissingAuthStrictMode(t *testing.T) {
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{
			"test": {
				From: "in",
				Auth: nil, // Missing auth
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
		},
	}

	// Should error in strict mode
	err := ValidateAuthDeclarations(config, AuthValidationStrict)
	if err == nil {
		t.Errorf("expected error in strict mode, got nil")
	}

	// Check error message contains route name
	if err != nil && err.Error() != "Route \"test\" missing auth: declaration. Declare auth: none if this route is intentionally unauthenticated" {
		t.Errorf("expected specific error message, got %v", err)
	}
}

// TestAuthValidationAuthNonePassesValidation tests that auth: none passes validation
func TestAuthValidationAuthNonePassesValidation(t *testing.T) {
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{
			"test": {
				From: "in",
				Auth: "none",
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
		},
	}

	// Should pass in both modes
	err := ValidateAuthDeclarations(config, AuthValidationWarn)
	if err != nil {
		t.Errorf("expected no error with auth: none in warn mode, got %v", err)
	}

	err = ValidateAuthDeclarations(config, AuthValidationStrict)
	if err != nil {
		t.Errorf("expected no error with auth: none in strict mode, got %v", err)
	}
}

// TestAuthValidationAuthorizeStepPassesValidation tests that auth with authorize steps passes
func TestAuthValidationAuthorizeStepPassesValidation(t *testing.T) {
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{
			"test": {
				From: "in",
				Auth: map[string]interface{}{
					"mode": "rbac",
				},
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
		},
	}

	// Should pass in both modes
	err := ValidateAuthDeclarations(config, AuthValidationWarn)
	if err != nil {
		t.Errorf("expected no error with auth object in warn mode, got %v", err)
	}

	err = ValidateAuthDeclarations(config, AuthValidationStrict)
	if err != nil {
		t.Errorf("expected no error with auth object in strict mode, got %v", err)
	}
}

// TestAuthValidationMultipleRoutesMixed tests mixed auth declarations with multiple routes
func TestAuthValidationMultipleRoutesMixed(t *testing.T) {
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{
			"route1": {
				From: "in",
				Auth: "none",
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
			"route2": {
				From: "in",
				Auth: map[string]interface{}{
					"mode": "abac",
				},
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
			"route3": {
				From: "in",
				Auth: nil, // Missing auth
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
		},
	}

	// Should pass in warn mode
	err := ValidateAuthDeclarations(config, AuthValidationWarn)
	if err != nil {
		t.Errorf("expected no error in warn mode, got %v", err)
	}

	// Should fail in strict mode
	err = ValidateAuthDeclarations(config, AuthValidationStrict)
	if err == nil {
		t.Errorf("expected error in strict mode, got nil")
	}

	// Check error message mentions the missing route
	if err != nil && err.Error() != "Route \"route3\" missing auth: declaration. Declare auth: none if this route is intentionally unauthenticated" {
		t.Errorf("expected error for route3, got %v", err)
	}
}

// TestAuthValidationErrorMessageContainsGuidance tests that error message includes guidance
func TestAuthValidationErrorMessageContainsGuidance(t *testing.T) {
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{
			"my_route": {
				From: "in",
				Auth: nil,
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
		},
	}

	err := ValidateAuthDeclarations(config, AuthValidationStrict)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	errorMsg := err.Error()
	if errorMsg != "Route \"my_route\" missing auth: declaration. Declare auth: none if this route is intentionally unauthenticated" {
		t.Errorf("expected guidance in error message, got: %s", errorMsg)
	}
}

// TestAuthValidationAllRoutesHaveAuth tests that all routes with auth pass
func TestAuthValidationAllRoutesHaveAuth(t *testing.T) {
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{
			"route1": {
				From: "in",
				Auth: "none",
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
			"route2": {
				From: "in",
				Auth: "none",
				ErrorPath: &ErrorPathSpec{
					Target: "out",
				},
				Steps: []StepSpec{},
			},
		},
	}

	// Should pass in both modes
	err := ValidateAuthDeclarations(config, AuthValidationWarn)
	if err != nil {
		t.Errorf("expected no error in warn mode, got %v", err)
	}

	err = ValidateAuthDeclarations(config, AuthValidationStrict)
	if err != nil {
		t.Errorf("expected no error in strict mode, got %v", err)
	}
}

// TestAuthValidationEmptyRoutes tests that empty routes map passes validation
func TestAuthValidationEmptyRoutes(t *testing.T) {
	config := &RouteConfig{
		Version: 1,
		Sources: map[string]SourceSpec{
			"in": {Type: "http"},
		},
		Sinks: map[string]SinkSpec{
			"out": {Type: "file", Path: "/tmp/out"},
		},
		Routes: map[string]RouteSpec{},
	}

	// Should pass in both modes
	err := ValidateAuthDeclarations(config, AuthValidationWarn)
	if err != nil {
		t.Errorf("expected no error with empty routes, got %v", err)
	}

	err = ValidateAuthDeclarations(config, AuthValidationStrict)
	if err != nil {
		t.Errorf("expected no error with empty routes, got %v", err)
	}
}
