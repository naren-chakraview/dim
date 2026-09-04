package steps

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestAuthorizeStepNoPrincipal verifies that no principal results in denial
func TestAuthorizeStepNoPrincipal(t *testing.T) {
	// Create an RBAC step
	step, err := NewAuthorizeStep("rbac", []string{"admin"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "read"}, "test-route", "v1")
	// Principal is nil by default

	result, err := step.Execute(ctx, msg)

	if err == nil {
		t.Error("Expected error when principal is nil, but got none")
	}

	if result != nil {
		t.Error("Expected nil message when principal is nil, but got a message")
	}

	// Verify error is permanent (contains "authorization_denied")
	if errMsg := err.Error(); errMsg != "authorization_denied: no principal" {
		t.Errorf("Expected error message 'authorization_denied: no principal', got %q", errMsg)
	}
}

// TestAuthorizeStepRBACRoleMatch verifies RBAC with matching role
func TestAuthorizeStepRBACRoleMatch(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin", "editor"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "write"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
		Roles:   []string{"editor", "viewer"},
	}

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute should succeed with matching role, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through with matching role, but got nil")
	}
}

// TestAuthorizeStepRBACRoleMissing verifies RBAC with no matching role
func TestAuthorizeStepRBACRoleMissing(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin", "editor"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "delete"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
		Roles:   []string{"viewer"},
	}

	result, err := step.Execute(ctx, msg)

	if err == nil {
		t.Error("Expected error when role is missing, but got none")
	}

	if result != nil {
		t.Error("Expected nil message when role is missing, but got a message")
	}

	if errMsg := err.Error(); errMsg != "authorization_denied: insufficient roles" {
		t.Errorf("Expected error message 'authorization_denied: insufficient roles', got %q", errMsg)
	}
}

// TestAuthorizeStepRBACEmptyRoles verifies RBAC with principal having no roles
func TestAuthorizeStepRBACEmptyRoles(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "read"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
		Roles:   []string{}, // No roles
	}

	result, err := step.Execute(ctx, msg)

	if err == nil {
		t.Error("Expected error when principal has no roles, but got none")
	}

	if result != nil {
		t.Error("Expected nil message when principal has no roles, but got a message")
	}
}

// TestAuthorizeStepABACExpressionTrue verifies ABAC with truthy expression
func TestAuthorizeStepABACExpressionTrue(t *testing.T) {
	step, err := NewAuthorizeStep("abac", nil, "principal.subject = 'admin-user'", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "write"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "admin-user",
		Roles:   []string{"viewer"},
	}

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute should succeed with true predicate, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through with true predicate, but got nil")
	}
}

// TestAuthorizeStepABACExpressionFalse verifies ABAC with falsy expression
func TestAuthorizeStepABACExpressionFalse(t *testing.T) {
	step, err := NewAuthorizeStep("abac", nil, "principal.subject = 'superadmin'", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "write"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "regular-user",
		Roles:   []string{"editor"},
	}

	result, err := step.Execute(ctx, msg)

	if err == nil {
		t.Error("Expected error when predicate is false, but got none")
	}

	if result != nil {
		t.Error("Expected nil message when predicate is false, but got a message")
	}

	if errMsg := err.Error(); errMsg != "authorization_denied: policy evaluation failed" {
		t.Errorf("Expected error message 'authorization_denied: policy evaluation failed', got %q", errMsg)
	}
}

// TestAuthorizeStepABACComplexExpression verifies ABAC with complex expressions
func TestAuthorizeStepABACComplexExpression(t *testing.T) {
	// Expression checks for specific subject OR specific body condition
	step, err := NewAuthorizeStep("abac", nil, `principal.subject = 'admin' or body.privileged = true`, "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "write", "privileged": true}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "regular-user",
		Roles:   []string{"viewer"},
	}

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute should succeed with complex predicate matching second condition, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through with true predicate, but got nil")
	}
}

// TestAuthorizeStepNewAuthorizeStepInvalidMode verifies invalid mode rejected
func TestAuthorizeStepNewAuthorizeStepInvalidMode(t *testing.T) {
	_, err := NewAuthorizeStep("invalid", []string{"admin"}, "", "", 0)
	if err == nil {
		t.Error("Expected error for invalid mode, but got none")
	}

	if errMsg := err.Error(); errMsg != `invalid mode: "invalid" (must be 'rbac', 'abac', or 'pbac')` {
		t.Errorf("Expected error about invalid mode, got %q", errMsg)
	}
}

// TestAuthorizeStepNewAuthorizeStepRBACNoRoles verifies RBAC requires roles
func TestAuthorizeStepNewAuthorizeStepRBACNoRoles(t *testing.T) {
	_, err := NewAuthorizeStep("rbac", []string{}, "", "", 0)
	if err == nil {
		t.Error("Expected error when RBAC has no require_roles, but got none")
	}

	if errMsg := err.Error(); errMsg != "rbac mode requires at least one role in require_roles" {
		t.Errorf("Expected error about empty roles, got %q", errMsg)
	}
}

// TestAuthorizeStepNewAuthorizeStepABACNoExpr verifies ABAC requires expression
func TestAuthorizeStepNewAuthorizeStepABACNoExpr(t *testing.T) {
	_, err := NewAuthorizeStep("abac", nil, "", "", 0)
	if err == nil {
		t.Error("Expected error when ABAC has no expression, but got none")
	}

	if errMsg := err.Error(); errMsg != "abac mode requires an expression" {
		t.Errorf("Expected error about missing expression, got %q", errMsg)
	}
}

// TestAuthorizeStepNewAuthorizeStepInvalidABACExpr verifies invalid ABAC expression rejected
func TestAuthorizeStepNewAuthorizeStepInvalidABACExpr(t *testing.T) {
	_, err := NewAuthorizeStep("abac", nil, "invalid syntax }{", "", 0)
	if err == nil {
		t.Error("Expected error for invalid ABAC expression, but got none")
	}
}

// TestAuthorizeStepNilMessage verifies nil messages are handled gracefully
func TestAuthorizeStepNilMessage(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, nil)

	if err != nil {
		t.Fatalf("Execute should not error on nil message, got %v", err)
	}

	if result != nil {
		t.Error("Expected nil message to pass through as nil")
	}
}

// TestAuthorizeStepMetadataPreservation verifies metadata is preserved on authorization
func TestAuthorizeStepMetadataPreservation(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "write"}, "special-route", "v123")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
		Roles:   []string{"admin"},
	}
	msg.Metadata.Stage = "authorizing"

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through, but got nil")
	}

	// Verify metadata is preserved
	if result.Metadata.Route != "special-route" {
		t.Errorf("Route not preserved: got %s, want special-route", result.Metadata.Route)
	}

	if result.Metadata.RouteVersion != "v123" {
		t.Errorf("RouteVersion not preserved: got %s, want v123", result.Metadata.RouteVersion)
	}

	if result.Metadata.Stage != "authorizing" {
		t.Errorf("Stage not preserved: got %s, want authorizing", result.Metadata.Stage)
	}

	if result.Metadata.Principal.Subject != "user123" {
		t.Errorf("Principal not preserved: got %s, want user123", result.Metadata.Principal.Subject)
	}
}

// TestAuthorizeStepRBACMultipleRoles verifies RBAC with multiple principal roles
func TestAuthorizeStepRBACMultipleRoles(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin", "editor"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"action": "read"}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
		Roles:   []string{"viewer", "contributor", "editor"},
	}

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute should succeed with matching role in multiple roles, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through with matching role, but got nil")
	}
}

// TestAuthorizeStepABACWithSubject verifies ABAC can access principal subject
func TestAuthorizeStepABACWithSubject(t *testing.T) {
	// Use subject which is a simple string field
	step, err := NewAuthorizeStep("abac", nil, `principal.subject = 'authorized-user'`, "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"amount": 1000}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "authorized-user",
		Roles:   []string{"viewer"},
	}

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute should succeed when subject matches, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through with matching subject, but got nil")
	}
}

// TestAuthorizeStepABACTruthyFalsy verifies ABAC truthy/falsy handling
func TestAuthorizeStepABACTruthyFalsy(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		principal *engine.Principal
		shouldAllow bool
	}{
		{
			name:      "numeric 1 is truthy",
			expr:      "1",
			principal: &engine.Principal{Subject: "user1"},
			shouldAllow: true,
		},
		{
			name:      "numeric 0 is falsy",
			expr:      "0",
			principal: &engine.Principal{Subject: "user1"},
			shouldAllow: false,
		},
		{
			name:      "non-empty string is truthy",
			expr:      `"allowed"`,
			principal: &engine.Principal{Subject: "user1"},
			shouldAllow: true,
		},
		{
			name:      "empty string is falsy",
			expr:      `""`,
			principal: &engine.Principal{Subject: "user1"},
			shouldAllow: false,
		},
		{
			name:      "boolean true",
			expr:      "true",
			principal: &engine.Principal{Subject: "user1"},
			shouldAllow: true,
		},
		{
			name:      "boolean false",
			expr:      "false",
			principal: &engine.Principal{Subject: "user1"},
			shouldAllow: false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := NewAuthorizeStep("abac", nil, tt.expr, "", 0)
			if err != nil {
				t.Fatalf("NewAuthorizeStep failed: %v", err)
			}

			msg := engine.NewMessage(map[string]interface{}{}, "test-route", "v1")
			msg.Metadata.Principal = tt.principal

			result, err := step.Execute(ctx, msg)

			if tt.shouldAllow {
				if err != nil {
					t.Errorf("Expected success with truthy value, got error: %v", err)
				}
				if result == nil {
					t.Error("Expected message to pass through with truthy value")
				}
			} else {
				if err == nil {
					t.Error("Expected error with falsy value")
				}
				if result != nil {
					t.Error("Expected nil message with falsy value")
				}
			}
		})
	}
}

// TestAuthorizeStepABACBodyContextAvailable verifies body is available in ABAC expression
func TestAuthorizeStepABACBodyContextAvailable(t *testing.T) {
	step, err := NewAuthorizeStep("abac", nil, `body.amount > 100`, "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"amount": 150}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
	}

	result, err := step.Execute(ctx, msg)

	if err != nil {
		t.Fatalf("Execute should succeed when body condition is true, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected message to pass through when body condition is true")
	}
}

// TestAuthorizeStepABACBodyContextFail verifies body condition can fail in ABAC
func TestAuthorizeStepABACBodyContextFail(t *testing.T) {
	step, err := NewAuthorizeStep("abac", nil, `body.amount > 100`, "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{"amount": 50}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
	}

	result, err := step.Execute(ctx, msg)

	if err == nil {
		t.Error("Expected error when body condition is false")
	}

	if result != nil {
		t.Error("Expected nil message when body condition is false")
	}
}

// TestAuthorizeStepPermanentErrorType verifies authorization errors are permanent
func TestAuthorizeStepPermanentErrorType(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	ctx := context.Background()
	msg := engine.NewMessage(map[string]interface{}{}, "test-route", "v1")
	msg.Metadata.Principal = &engine.Principal{
		Subject: "user123",
		Roles:   []string{"viewer"},
	}

	_, err = step.Execute(ctx, msg)

	if err == nil {
		t.Fatal("Expected error from authorization denial")
	}

	// Check that error is a PermanentError
	_, isPermanent := err.(*PermanentError)
	if !isPermanent {
		t.Errorf("Expected error to be PermanentError, got %T", err)
	}

	// Verify ClassifyError treats it as permanent
	errorType := engine.ClassifyError(err)
	if errorType != engine.ErrorTypePermanent {
		t.Errorf("Expected ClassifyError to return ErrorTypePermanent, got %v", errorType)
	}
}

// TestAuthorizeStepRBACExactMatch verifies RBAC does exact string matching
func TestAuthorizeStepRBACExactMatch(t *testing.T) {
	step, err := NewAuthorizeStep("rbac", []string{"admin"}, "", "", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep failed: %v", err)
	}

	tests := []struct {
		name       string
		principalRoles []string
		shouldAllow bool
	}{
		{
			name:       "exact match",
			principalRoles: []string{"admin"},
			shouldAllow: true,
		},
		{
			name:       "substring no match",
			principalRoles: []string{"administrator"},
			shouldAllow: false,
		},
		{
			name:       "case sensitive no match",
			principalRoles: []string{"Admin"},
			shouldAllow: false,
		},
		{
			name:       "whitespace no match",
			principalRoles: []string{"admin "},
			shouldAllow: false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := engine.NewMessage(map[string]interface{}{}, "test-route", "v1")
			msg.Metadata.Principal = &engine.Principal{
				Subject: "user123",
				Roles:   tt.principalRoles,
			}

			result, err := step.Execute(ctx, msg)

			if tt.shouldAllow {
				if err != nil {
					t.Errorf("Expected success with exact role match, got error: %v", err)
				}
				if result == nil {
					t.Error("Expected message to pass through with exact role match")
				}
			} else {
				if err == nil {
					t.Error("Expected error when role does not exactly match")
				}
				if result != nil {
					t.Error("Expected nil message when role does not exactly match")
				}
			}
		})
	}
}

// TestAuthorizeStepNewAuthorizeStepPBACNoEndpoint verifies PBAC requires PDP endpoint (M1.2)
func TestAuthorizeStepNewAuthorizeStepPBACNoEndpoint(t *testing.T) {
	_, err := NewAuthorizeStep("pbac", nil, "", "", 0)
	if err == nil {
		t.Error("Expected error when PBAC has no pdp endpoint, but got none")
	}

	if errMsg := err.Error(); errMsg != "pbac mode requires a pdp endpoint" {
		t.Errorf("Expected error about missing PDP endpoint, got %q", errMsg)
	}
}

// TestAuthorizeStepPBACConfiguration verifies PBAC step configuration (M1.2)
func TestAuthorizeStepPBACConfiguration(t *testing.T) {
	// Valid PBAC configuration
	step, err := NewAuthorizeStep("pbac", nil, "", "http://localhost:8181", 5000)
	if err != nil {
		t.Fatalf("NewAuthorizeStep with valid PBAC config failed: %v", err)
	}

	if step.pdpEndpoint != "http://localhost:8181" {
		t.Errorf("Expected pdpEndpoint to be 'http://localhost:8181', got %q", step.pdpEndpoint)
	}

	if step.pdpTimeout != 5000 {
		t.Errorf("Expected pdpTimeout to be 5000, got %d", step.pdpTimeout)
	}
}

// TestAuthorizeStepPBACDefaultTimeout verifies PBAC default timeout (M1.2)
func TestAuthorizeStepPBACDefaultTimeout(t *testing.T) {
	// PBAC with no timeout should get default 5000ms
	step, err := NewAuthorizeStep("pbac", nil, "", "http://localhost:8181", 0)
	if err != nil {
		t.Fatalf("NewAuthorizeStep with PBAC config failed: %v", err)
	}

	if step.pdpTimeout != 5000 {
		t.Errorf("Expected default pdpTimeout to be 5000, got %d", step.pdpTimeout)
	}
}
