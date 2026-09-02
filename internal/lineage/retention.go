package lineage

import (
	"fmt"
	"log"
	"time"

	"github.com/blues/jsonata-go"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// RetentionPolicy represents a named retention policy with TTL
type RetentionPolicy struct {
	Name string
	TTL  time.Duration
}

// RetentionResolver handles both static and dynamic retention policy resolution
type RetentionResolver struct {
	policies map[string]time.Duration
}

// NewRetentionResolver creates a new retention resolver from a map of policy names to durations
// Example: {"30-days": 720*time.Hour, "premium-90": 2160*time.Hour}
func NewRetentionResolver(policies map[string]time.Duration) *RetentionResolver {
	return &RetentionResolver{
		policies: policies,
	}
}

// Resolve determines the retention policy and TTL for a message
// 1. If RetentionPolicyExpr is set, evaluates it to get policy name
// 2. Otherwise uses RetentionPolicy (static)
// 3. Looks up policy name in configured policies
// 4. Falls back to "default" if not found (fail-open)
func (rr *RetentionResolver) Resolve(
	routeSpec *config.RouteSpec,
	msg *engine.Message,
) (string, time.Duration, error) {
	var policyName string
	var err error

	// If dynamic expr is configured, evaluate it
	if routeSpec.RetentionPolicyExpr != "" {
		policyName, err = rr.evaluateRetentionExpr(routeSpec.RetentionPolicyExpr, msg)
		if err != nil {
			log.Printf("[WARN] retention policy expr evaluation failed: %v, falling back to default", err)
			policyName = "default"
		}
	} else if routeSpec.RetentionPolicy != "" {
		// Use static policy
		policyName = routeSpec.RetentionPolicy
	} else {
		// Default to "default" policy
		policyName = "default"
	}

	// Look up TTL for policy name
	ttl, ok := rr.policies[policyName]
	if !ok {
		log.Printf("[WARN] retention policy %q not found, falling back to default", policyName)
		policyName = "default"
		ttl, _ = rr.policies["default"]
		if ttl == 0 {
			// Final fallback: 30 days
			ttl = 30 * 24 * time.Hour
		}
	}

	return policyName, ttl, nil
}

// evaluateRetentionExpr evaluates a JSONata expression to extract the policy name
func (rr *RetentionResolver) evaluateRetentionExpr(expr string, msg *engine.Message) (string, error) {
	evaluator := jsonata.MustCompile(expr)

	// Create evaluation context with message data
	result, err := evaluator.Eval(map[string]interface{}{
		"body":     msg.Body,
		"headers":  msg.Headers,
		"metadata": msg.Metadata,
	})
	if err != nil {
		return "", fmt.Errorf("expression evaluation error: %w", err)
	}

	// Result should be a string (policy name)
	policyName, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("retention policy expr must return a string, got %T", result)
	}

	return policyName, nil
}

// ExtractSubjectID evaluates the subject_id_expr to extract a subject identifier
// Returns empty string if expr is empty (optional)
// Returns error if expr evaluation fails
func (rr *RetentionResolver) ExtractSubjectID(
	expr string,
	msg *engine.Message,
) (string, error) {
	if expr == "" {
		return "", nil
	}

	evaluator := jsonata.MustCompile(expr)

	// Create evaluation context
	result, err := evaluator.Eval(map[string]interface{}{
		"body":     msg.Body,
		"headers":  msg.Headers,
		"metadata": msg.Metadata,
	})
	if err != nil {
		return "", fmt.Errorf("subject_id expr evaluation error: %w", err)
	}

	// Result should be a string (subject ID)
	if result == nil {
		return "", nil
	}

	subjectID, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("subject_id expr must return a string or null, got %T", result)
	}

	return subjectID, nil
}
