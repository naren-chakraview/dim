package secrets

import (
	"errors"
	"testing"
)

func TestResolveInRouteNotFoundReturnsError(t *testing.T) {
	resolver := NewResolver(NewInMemoryStore())

	resolved, err := resolver.ResolveInRoute("payments", "k: ${SECRET:missing}")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("missing secret must return ErrSecretNotFound, got %v", err)
	}
	if resolved != "" {
		t.Fatalf("failed resolution must not return partial output: %q", resolved)
	}
}

func TestResolveInRouteDenialIsDistinguishableFromNotFound(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "api-key", Value: "real", Domain: "payments"})
	resolver := NewResolver(store)

	_, err := resolver.ResolveInRoute("orders", "k: ${SECRET:payments.api-key}")
	if !errors.Is(err, ErrCrossDomainDenied) {
		t.Fatalf("expected ErrCrossDomainDenied, got %v", err)
	}
	if errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("denial must not be reported as not-found: %v", err)
	}
}

func TestParseSecretRefRejectsEmptySegments(t *testing.T) {
	bad := []string{"", ".", ".payments.api-key", ".api-key", "payments."}
	for _, ref := range bad {
		if _, _, err := parseSecretRef(ref); err == nil {
			t.Errorf("parseSecretRef(%q) should be rejected", ref)
		}
	}
}

func TestParseSecretRefValidForms(t *testing.T) {
	cases := []struct{ ref, domain, name string }{
		{"api-key", "", "api-key"},
		{"payments.api-key", "payments", "api-key"},
	}
	for _, c := range cases {
		d, n, err := parseSecretRef(c.ref)
		if err != nil || d != c.domain || n != c.name {
			t.Errorf("parseSecretRef(%q) = (%q, %q, %v), want (%q, %q, nil)", c.ref, d, n, err, c.domain, c.name)
		}
	}
}

func TestLeadingDotReferenceFailsValidation(t *testing.T) {
	store := NewInMemoryStore()
	store.Register(SecretEntry{Name: "api-key", Value: "real", Domain: "payments"})
	resolver := NewResolver(store)

	if err := resolver.ValidateSecretReferences("orders", "k: ${SECRET:.payments.api-key}"); err == nil {
		t.Fatal("leading-dot reference must fail validation")
	}
}
