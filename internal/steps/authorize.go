package steps

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// AuthorizeStep implements authorization enforcement with RBAC or ABAC policies.
// It validates that the principal has the required permissions before allowing
// the message to proceed. Authorization denials are permanent errors (don't retry).
type AuthorizeStep struct {
	mode         string          // "rbac" or "abac"
	requireRoles []string        // for RBAC mode
	evaluator    *expr.Evaluator // for ABAC mode (compiled expression)
}

// NewAuthorizeStep creates a new authorization step.
// For RBAC mode: requireRoles must not be empty.
// For ABAC mode: abacExpr is compiled at construction time to catch syntax errors early.
func NewAuthorizeStep(mode string, requireRoles []string, abacExpr string) (*AuthorizeStep, error) {
	// Validate mode
	if mode != "rbac" && mode != "abac" {
		return nil, fmt.Errorf("invalid mode: %q (must be 'rbac' or 'abac')", mode)
	}

	// Mode-specific validation
	if mode == "rbac" {
		if len(requireRoles) == 0 {
			return nil, fmt.Errorf("rbac mode requires at least one role in require_roles")
		}
	}

	if mode == "abac" {
		if abacExpr == "" {
			return nil, fmt.Errorf("abac mode requires an expression")
		}
	}

	// For ABAC: compile expression at construction time
	var evaluator *expr.Evaluator
	if mode == "abac" {
		var err error
		evaluator, err = expr.CompileExpression(abacExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to compile abac expression: %w", err)
		}
	}

	return &AuthorizeStep{
		mode:         mode,
		requireRoles: requireRoles,
		evaluator:    evaluator,
	}, nil
}

// Execute evaluates the authorization policy and returns an error if denied.
// Returns:
// - (msg, nil) if the principal is authorized
// - (nil, error) if the principal is not authorized or principal is nil
// Error type is permanent (don't retry authorization failures)
func (as *AuthorizeStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, nil
	}

	// Extract principal from metadata
	principal := msg.Metadata.Principal
	if principal == nil {
		return nil, &PermanentError{
			Msg: "authorization_denied: no principal",
		}
	}

	// Evaluate policy based on mode
	switch as.mode {
	case "rbac":
		return as.executeRBAC(principal, msg)
	case "abac":
		return as.executeABAC(principal, msg)
	}

	// Should not reach here (mode is validated in constructor)
	return nil, &PermanentError{
		Msg: "authorization_denied: invalid authorization mode",
	}
}

// executeRBAC checks if the principal has any of the required roles
func (as *AuthorizeStep) executeRBAC(principal *engine.Principal, msg *engine.Message) (*engine.Message, error) {
	// Check if any principal role matches any required role
	for _, principalRole := range principal.Roles {
		for _, requiredRole := range as.requireRoles {
			if principalRole == requiredRole {
				// Match found - authorized
				return msg, nil
			}
		}
	}

	// No matching role found - denied
	return nil, &PermanentError{
		Msg: "authorization_denied: insufficient roles",
	}
}

// executeABAC evaluates a JSONata predicate with the principal in scope
func (as *AuthorizeStep) executeABAC(principal *engine.Principal, msg *engine.Message) (*engine.Message, error) {
	// Convert principal to a map for proper JSONata access
	principalMap := map[string]interface{}{
		"subject": principal.Subject,
		"roles":   principal.Roles,
		"claims":  principal.Claims,
		"token":   principal.Token,
	}

	// Build the context object with principal, body, headers, and metadata
	context := map[string]interface{}{
		"principal": principalMap,
		"body":      msg.Body,
		"headers":   msg.Headers,
		"metadata":  msg.Metadata,
	}

	// Evaluate the predicate
	result, err := as.evaluator.Eval(context)
	if err != nil {
		return nil, fmt.Errorf("abac expression evaluation failed: %w", err)
	}

	// Convert the result to a boolean
	allowed := isTruthy(result)

	if !allowed {
		return nil, &PermanentError{
			Msg: "authorization_denied: policy evaluation failed",
		}
	}

	return msg, nil
}

// isTruthy converts a value to a boolean following JSONata truthiness rules
func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case float64:
		// In JSONata, numeric 0 is falsy, other numbers are truthy
		return v != 0
	case string:
		// In JSONata, empty string is falsy, non-empty is truthy
		return v != ""
	default:
		// Other types (arrays, objects) are truthy
		return true
	}
}

// PermanentError represents an authorization error that should not be retried
type PermanentError struct {
	Msg string
}

func (e *PermanentError) Error() string {
	return e.Msg
}
