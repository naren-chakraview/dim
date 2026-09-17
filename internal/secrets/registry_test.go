package secrets

import (
	"testing"
)

func TestSecretRegistrationGlobal(t *testing.T) {
	store := NewInMemoryStore()

	// Register a global secret
	err := store.Register(SecretEntry{
		Name:   "db-password",
		Value:  "secret123",
		Domain: "",
	})
	if err != nil {
		t.Fatalf("failed to register global secret: %v", err)
	}

	// Resolve from any domain should work
	val, err := store.Resolve("payments", "db-password")
	if err != nil {
		t.Fatalf("failed to resolve global secret: %v", err)
	}
	if val != "secret123" {
		t.Fatalf("got %q, want %q", val, "secret123")
	}
}

func TestSecretRegistrationDomainScoped(t *testing.T) {
	store := NewInMemoryStore()

	// Register domain-scoped secrets
	err := store.Register(SecretEntry{
		Name:   "api-key",
		Value:  "payments-key-123",
		Domain: "payments",
	})
	if err != nil {
		t.Fatalf("failed to register domain secret: %v", err)
	}

	err = store.Register(SecretEntry{
		Name:   "api-key",
		Value:  "orders-key-456",
		Domain: "orders",
	})
	if err != nil {
		t.Fatalf("failed to register domain secret: %v", err)
	}

	// Resolve from same domain should work
	val, err := store.Resolve("payments", "api-key")
	if err != nil {
		t.Fatalf("failed to resolve domain secret: %v", err)
	}
	if val != "payments-key-123" {
		t.Fatalf("got %q, want %q", val, "payments-key-123")
	}

	// Cross-domain access must be DENIED (T1.13 authorization check)
	// Even with explicit syntax (domain.name), cross-domain is blocked
	_, err = store.Resolve("orders", "payments.api-key")
	if err == nil {
		t.Fatalf("T1.13 BUG: Cross-domain access was allowed")
	}
	if !containsErrorString(err.Error(), "cross-domain") {
		t.Fatalf("Expected cross-domain error, got: %v", err)
	}
}

func TestSecretAccessControlImplicit(t *testing.T) {
	store := NewInMemoryStore()

	// Register secret in payments domain
	err := store.Register(SecretEntry{
		Name:   "api-key",
		Value:  "payments-key",
		Domain: "payments",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Try to resolve from different domain without explicit syntax — should fail
	_, err = store.Resolve("orders", "api-key")
	if err == nil {
		t.Fatal("expected error when accessing domain-scoped secret from different domain without explicit syntax")
	}
}

func TestSecretAccessControlExplicit(t *testing.T) {
	// T1.13 Fix: Cross-domain access is explicitly blocked, even with explicit syntax
	// The explicit syntax documents intent but does NOT grant access.
	// This enforces domain-scoping isolation required by governance model.

	store := NewInMemoryStore()

	// Register secret in payments domain
	err := store.Register(SecretEntry{
		Name:   "api-key",
		Value:  "payments-key",
		Domain: "payments",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// CRITICAL: Cross-domain resolution must be DENIED, not allowed
	// Even with explicit syntax (domain.name), a route in "orders" domain
	// cannot access secrets from "payments" domain
	val, err := store.Resolve("orders", "payments.api-key")
	if err == nil {
		t.Fatalf("T1.13 BUG: Cross-domain access was allowed! Got value: %q. "+
			"Domain-scoping authorization check failed.", val)
	}
	if !containsErrorString(err.Error(), "cross-domain") {
		t.Fatalf("Expected 'cross-domain' error, got: %v", err)
	}
}

func TestSecretResolutionPriority(t *testing.T) {
	store := NewInMemoryStore()

	// Register global secret
	err := store.Register(SecretEntry{
		Name:   "db-password",
		Value:  "global-password",
		Domain: "",
	})
	if err != nil {
		t.Fatalf("failed to register global: %v", err)
	}

	// Register domain-scoped secret with same name
	err = store.Register(SecretEntry{
		Name:   "db-password",
		Value:  "domain-password",
		Domain: "payments",
	})
	if err != nil {
		t.Fatalf("failed to register domain: %v", err)
	}

	// Resolve from payments — should get domain-scoped version first
	val, err := store.Resolve("payments", "db-password")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}
	if val != "domain-password" {
		t.Fatalf("got %q, want %q (domain-scoped should take priority)", val, "domain-password")
	}

	// Resolve from orders — should get global fallback
	val, err = store.Resolve("orders", "db-password")
	if err != nil {
		t.Fatalf("failed to resolve from different domain: %v", err)
	}
	if val != "global-password" {
		t.Fatalf("got %q, want %q (should fall back to global)", val, "global-password")
	}
}

func TestSecretNotFound(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.Resolve("payments", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent secret")
	}
}

func TestListByDomain(t *testing.T) {
	store := NewInMemoryStore()

	// Register multiple secrets
	store.Register(SecretEntry{Name: "key1", Value: "val1", Domain: "payments"})
	store.Register(SecretEntry{Name: "key2", Value: "val2", Domain: "payments"})
	store.Register(SecretEntry{Name: "key3", Value: "val3", Domain: "orders"})
	store.Register(SecretEntry{Name: "key4", Value: "val4", Domain: ""})

	// List payments domain
	entries, err := store.ListByDomain("payments")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	// Verify entries
	nameSet := make(map[string]bool)
	for _, e := range entries {
		nameSet[e.Name] = true
	}
	if !nameSet["key1"] || !nameSet["key2"] {
		t.Fatal("unexpected entries in payments domain")
	}
}

func TestSecretNameValidation(t *testing.T) {
	store := NewInMemoryStore()

	err := store.Register(SecretEntry{
		Name:   "",
		Value:  "val",
		Domain: "test",
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestParseSecretRef(t *testing.T) {
	tests := []struct {
		ref          string
		wantDomain   string
		wantName     string
	}{
		{"api-key", "", "api-key"},
		{"payments.api-key", "payments", "api-key"},
		{"orders.db.password", "orders", "db.password"},
		{"global-key", "", "global-key"},
	}

	for _, tt := range tests {
		gotDomain, gotName := parseSecretRef(tt.ref)
		if gotDomain != tt.wantDomain || gotName != tt.wantName {
			t.Errorf("parseSecretRef(%q) = (%q, %q), want (%q, %q)",
				tt.ref, gotDomain, gotName, tt.wantDomain, tt.wantName)
		}
	}
}

// Helper: check if a string contains a substring
func containsErrorString(err, substr string) bool {
	for i := 0; i+len(substr) <= len(err); i++ {
		if err[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
