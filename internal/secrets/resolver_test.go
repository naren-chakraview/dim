package secrets

import (
	"strings"
	"testing"
)

func TestResolveInRouteSameDomain(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "db-url", Value: "postgresql://payments-db", Domain: "payments"})

	resolver := NewResolver(store)

	text := "connection: ${SECRET:db-url}"
	resolved, err := resolver.ResolveInRoute("payments", text)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if !strings.Contains(resolved, "postgresql://payments-db") {
		t.Fatalf("secret not resolved in output: %q", resolved)
	}
}

func TestResolveInRouteGlobalFallback(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "shared-key", Value: "global-key-value", Domain: ""})

	resolver := NewResolver(store)

	text := "api_key: ${SECRET:shared-key}"
	resolved, err := resolver.ResolveInRoute("payments", text)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if !strings.Contains(resolved, "global-key-value") {
		t.Fatalf("global secret not resolved: %q", resolved)
	}
}

func TestResolveInRouteExplicitCrossDomain(t *testing.T) {
	// T1.13 Fix: Cross-domain secret references are blocked by authorization
	// Even with explicit syntax (domain.name), the resolver cannot access them
	// Cross-domain patterns remain unresolved in the output

	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "shared-secret", Value: "platform-secret", Domain: "platform"})

	resolver := NewResolver(store)

	text := "shared: ${SECRET:platform.shared-secret}"
	resolved, err := resolver.ResolveInRoute("payments", text)
	if err != nil {
		t.Fatalf("failed to call resolve: %v", err)
	}

	// Cross-domain secrets remain unresolved (pattern is kept as-is)
	// This prevents silent failures and makes it obvious the secret couldn't be accessed
	if !strings.Contains(resolved, "${SECRET:platform.shared-secret}") {
		t.Fatalf("cross-domain secret should remain unresolved in output: %q", resolved)
	}
}

func TestResolveMultipleSecrets(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "db-user", Value: "admin", Domain: "payments"})
	store.Register(SecretEntry{Name: "db-pass", Value: "secret123", Domain: "payments"})

	resolver := NewResolver(store)

	text := "user: ${SECRET:db-user}\npass: ${SECRET:db-pass}"
	resolved, err := resolver.ResolveInRoute("payments", text)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if !strings.Contains(resolved, "admin") || !strings.Contains(resolved, "secret123") {
		t.Fatalf("not all secrets resolved: %q", resolved)
	}
}

func TestValidateSecretReferencesPass(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "api-key", Value: "key123", Domain: "payments"})

	resolver := NewResolver(store)

	text := "header: Authorization: Bearer ${SECRET:api-key}"
	err := resolver.ValidateSecretReferences("payments", text)
	if err != nil {
		t.Fatalf("validation should pass: %v", err)
	}
}

func TestValidateSecretReferencesFail(t *testing.T) {
	store := NewInMemoryStore()
	// Register secret in orders domain, but trying to access from payments

	resolver := NewResolver(store)

	text := "header: Authorization: Bearer ${SECRET:api-key}"
	err := resolver.ValidateSecretReferences("payments", text)
	if err == nil {
		t.Fatal("validation should fail for unresolvable secret")
	}
}

func TestValidateSecretReferencesExplicitCrossDomain(t *testing.T) {
	// T1.13 Fix: Cross-domain access is rejected even with explicit syntax
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "shared-key", Value: "value", Domain: "platform"})

	resolver := NewResolver(store)

	// Explicit cross-domain syntax must be rejected by validation
	// Routes cannot access secrets from other domains, period
	text := "key: ${SECRET:platform.shared-key}"
	err := resolver.ValidateSecretReferences("payments", text)
	if err == nil {
		t.Fatalf("T1.13 BUG: explicit cross-domain should be rejected")
	}
	if !strings.Contains(err.Error(), "cross-domain") {
		t.Fatalf("error should mention cross-domain denial: %v", err)
	}
}

func TestListUnresolvedSecrets(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "found", Value: "val", Domain: "payments"})

	resolver := NewResolver(store)

	text := "resolved: ${SECRET:found}\nunresolved: ${SECRET:missing}"
	unresolved, err := resolver.ListUnresolvedSecrets("payments", text)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(unresolved) != 1 || unresolved[0] != "missing" {
		t.Fatalf("got %v, want [missing]", unresolved)
	}
}

func TestResolveEmptyText(t *testing.T) {
	store := NewInMemoryStore()
	resolver := NewResolver(store)

	resolved, err := resolver.ResolveInRoute("payments", "")
	if err != nil {
		t.Fatalf("should handle empty text: %v", err)
	}
	if resolved != "" {
		t.Fatalf("empty text should return empty, got %q", resolved)
	}
}

func TestValidateEmptyText(t *testing.T) {
	store := NewInMemoryStore()
	resolver := NewResolver(store)

	err := resolver.ValidateSecretReferences("payments", "")
	if err != nil {
		t.Fatalf("should handle empty text: %v", err)
	}
}

func TestResolveImplicitCrossDomainAccessBlocked(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "db-pass", Value: "secret", Domain: "orders"})

	resolver := NewResolver(store)

	// Trying to access orders' secret from payments without explicit syntax
	text := "pass: ${SECRET:db-pass}"
	err := resolver.ValidateSecretReferences("payments", text)
	if err == nil {
		t.Fatal("should fail for implicit cross-domain access")
	}

	if !strings.Contains(err.Error(), "route domain") {
		t.Fatalf("error should mention domain: %v", err)
	}
}
