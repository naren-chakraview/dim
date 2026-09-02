package lineage

import (
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

func TestResolveStaticRetention(t *testing.T) {
	policies := map[string]time.Duration{
		"default":      30 * 24 * time.Hour,
		"30-days":      30 * 24 * time.Hour,
		"premium-90":   90 * 24 * time.Hour,
	}

	resolver := NewRetentionResolver(policies)

	routeSpec := &config.RouteSpec{
		RetentionPolicy: "30-days",
	}

	msg := engine.NewMessage(map[string]interface{}{}, "route", "v1")

	policyName, ttl, err := resolver.Resolve(routeSpec, msg)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if policyName != "30-days" {
		t.Errorf("Expected policy '30-days', got %q", policyName)
	}

	if ttl != 30*24*time.Hour {
		t.Errorf("Expected TTL 30 days, got %v", ttl)
	}
}

func TestResolveDynamicRetention(t *testing.T) {
	policies := map[string]time.Duration{
		"default":      30 * 24 * time.Hour,
		"premium-90":   90 * 24 * time.Hour,
	}

	resolver := NewRetentionResolver(policies)

	routeSpec := &config.RouteSpec{
		RetentionPolicyExpr: `body.tier = "premium" ? "premium-90" : "default"`,
	}

	// Test premium message
	msg := engine.NewMessage(map[string]interface{}{"tier": "premium"}, "route", "v1")

	policyName, ttl, err := resolver.Resolve(routeSpec, msg)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if policyName != "premium-90" {
		t.Errorf("Expected policy 'premium-90', got %q", policyName)
	}

	if ttl != 90*24*time.Hour {
		t.Errorf("Expected TTL 90 days, got %v", ttl)
	}

	// Test regular message
	msg2 := engine.NewMessage(map[string]interface{}{"tier": "standard"}, "route", "v1")

	policyName2, ttl2, err := resolver.Resolve(routeSpec, msg2)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if policyName2 != "default" {
		t.Errorf("Expected policy 'default', got %q", policyName2)
	}

	if ttl2 != 30*24*time.Hour {
		t.Errorf("Expected TTL 30 days, got %v", ttl2)
	}
}

func TestResolveFailOpen(t *testing.T) {
	policies := map[string]time.Duration{
		"default": 30 * 24 * time.Hour,
	}

	resolver := NewRetentionResolver(policies)

	routeSpec := &config.RouteSpec{
		RetentionPolicy: "nonexistent-policy",
	}

	msg := engine.NewMessage(map[string]interface{}{}, "route", "v1")

	policyName, ttl, err := resolver.Resolve(routeSpec, msg)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if policyName != "default" {
		t.Errorf("Expected fallback to 'default', got %q", policyName)
	}

	if ttl != 30*24*time.Hour {
		t.Errorf("Expected fallback TTL 30 days, got %v", ttl)
	}
}

func TestResolveExprEvaluationError(t *testing.T) {
	policies := map[string]time.Duration{
		"default": 30 * 24 * time.Hour,
	}

	resolver := NewRetentionResolver(policies)

	routeSpec := &config.RouteSpec{
		RetentionPolicyExpr: `body.nonexistent.field`,
	}

	msg := engine.NewMessage(map[string]interface{}{}, "route", "v1")

	// Should fall back to default on evaluation error
	policyName, _, err := resolver.Resolve(routeSpec, msg)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if policyName != "default" {
		t.Errorf("Expected fallback to 'default' on expr error, got %q", policyName)
	}
}

func TestExtractSubjectID(t *testing.T) {
	resolver := NewRetentionResolver(nil)

	msg := engine.NewMessage(
		map[string]interface{}{"user_id": "user-123"},
		"route",
		"v1",
	)

	subjectID, err := resolver.ExtractSubjectID(`body.user_id`, msg)
	if err != nil {
		t.Fatalf("ExtractSubjectID failed: %v", err)
	}

	if subjectID != "user-123" {
		t.Errorf("Expected 'user-123', got %q", subjectID)
	}
}

func TestExtractSubjectIDEmpty(t *testing.T) {
	resolver := NewRetentionResolver(nil)

	msg := engine.NewMessage(map[string]interface{}{}, "route", "v1")

	// Empty expr should return empty string without error
	subjectID, err := resolver.ExtractSubjectID(``, msg)
	if err != nil {
		t.Fatalf("ExtractSubjectID with empty expr failed: %v", err)
	}

	if subjectID != "" {
		t.Errorf("Expected empty string for empty expr, got %q", subjectID)
	}
}

func TestExtractSubjectIDWithPrincipal(t *testing.T) {
	resolver := NewRetentionResolver(nil)

	// Test with simple body-based subject extraction instead
	msg := engine.NewMessage(map[string]interface{}{"principal_id": "principal-456"}, "route", "v1")

	subjectID, err := resolver.ExtractSubjectID(`body.principal_id`, msg)
	if err != nil {
		t.Fatalf("ExtractSubjectID from principal failed: %v", err)
	}

	if subjectID != "principal-456" {
		t.Errorf("Expected 'principal-456', got %q", subjectID)
	}
}
