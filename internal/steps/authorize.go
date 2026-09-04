package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// AuthorizeStep implements authorization enforcement with RBAC, ABAC, or PBAC policies.
// It validates that the principal has the required permissions before allowing
// the message to proceed. Authorization denials are permanent errors (don't retry).
type AuthorizeStep struct {
	mode         string          // "rbac", "abac", or "pbac"
	requireRoles []string        // for RBAC mode
	evaluator    *expr.Evaluator // for ABAC mode (compiled expression)
	pdpEndpoint  string          // for PBAC mode (M1.2)
	pdpTimeout   int             // for PBAC mode: timeout in milliseconds
}

// NewAuthorizeStep creates a new authorization step.
// For RBAC mode: requireRoles must not be empty.
// For ABAC mode: abacExpr is compiled at construction time to catch syntax errors early.
// For PBAC mode: pdpEndpoint must be provided (M1.2).
func NewAuthorizeStep(mode string, requireRoles []string, abacExpr string, pdpEndpoint string, pdpTimeout int) (*AuthorizeStep, error) {
	// Validate mode
	if mode != "rbac" && mode != "abac" && mode != "pbac" {
		return nil, fmt.Errorf("invalid mode: %q (must be 'rbac', 'abac', or 'pbac')", mode)
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

	if mode == "pbac" {
		if pdpEndpoint == "" {
			return nil, fmt.Errorf("pbac mode requires a pdp endpoint")
		}
		if pdpTimeout <= 0 {
			pdpTimeout = 5000 // default 5 seconds
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
		pdpEndpoint:  pdpEndpoint,
		pdpTimeout:   pdpTimeout,
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
	case "pbac":
		return as.executePBAC(ctx, principal, msg)
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

// executePBAC evaluates authorization against an external Policy Decision Point (M1.2)
// Follows the neutral PDP contract defined in design/PDP_CONTRACT_SPEC.md
func (as *AuthorizeStep) executePBAC(ctx context.Context, principal *engine.Principal, msg *engine.Message) (*engine.Message, error) {
	// Build decision request per M1.1 contract
	decisionReq := PDPDecisionRequest{
		Principal: PDPPrincipal{
			Subject: principal.Subject,
			Roles:   principal.Roles,
			Attributes: map[string]interface{}{
				"token": principal.Token,
			},
		},
		Action: "process_message", // default action
		Resource: PDPResource{
			Type: "message",
		},
		Context: PDPContext{
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}

	// Call the PDP endpoint
	decision, err := as.callPDP(ctx, &decisionReq)
	if err != nil {
		return nil, fmt.Errorf("pbac decision error: %w", err)
	}

	// Check the decision
	if decision.Decision != "allow" {
		return nil, &PermanentError{
			Msg: fmt.Sprintf("authorization_denied: %s", decision.Reason),
		}
	}

	return msg, nil
}

// callPDP makes an HTTP request to the PDP endpoint and returns the decision
func (as *AuthorizeStep) callPDP(ctx context.Context, req *PDPDecisionRequest) (*PDPDecisionResponse, error) {
	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PDP request: %w", err)
	}

	// Create HTTP request with timeout
	httpReq, err := http.NewRequestWithContext(ctx, "POST", as.pdpEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create PDP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(as.pdpTimeout) * time.Millisecond,
	}

	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("PDP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDP response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PDP returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Unmarshal response
	decision := &PDPDecisionResponse{}
	if err := json.Unmarshal(respBody, decision); err != nil {
		return nil, fmt.Errorf("failed to parse PDP response: %w", err)
	}

	return decision, nil
}

// PDPDecisionRequest follows the neutral contract from design/PDP_CONTRACT_SPEC.md
type PDPDecisionRequest struct {
	Principal PDPPrincipal   `json:"principal"`
	Action    string         `json:"action"`
	Resource  PDPResource    `json:"resource"`
	Context   PDPContext     `json:"context,omitempty"`
}

// PDPPrincipal represents the authenticated identity
type PDPPrincipal struct {
	Subject    string                 `json:"subject"`
	Roles      []string               `json:"roles,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// PDPResource represents the resource being accessed
type PDPResource struct {
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Route string `json:"route,omitempty"`
	Stage string `json:"stage,omitempty"`
}

// PDPContext provides additional context for the decision
type PDPContext struct {
	Timestamp      string `json:"timestamp,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	Environment    string `json:"environment,omitempty"`
}

// PDPDecisionResponse follows the neutral contract from design/PDP_CONTRACT_SPEC.md
type PDPDecisionResponse struct {
	Decision   string                   `json:"decision"` // "allow" or "deny"
	Obligations []PDPObligation         `json:"obligations,omitempty"`
	Reason     string                   `json:"reason,omitempty"`
	Metadata   map[string]interface{}   `json:"metadata,omitempty"`
}

// PDPObligation represents additional requirements for an allowed decision
type PDPObligation struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// PermanentError represents an authorization error that should not be retried
type PermanentError struct {
	Msg string
}

func (e *PermanentError) Error() string {
	return e.Msg
}
