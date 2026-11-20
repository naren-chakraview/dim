package file

import (
	"context"
	"errors"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
)

func scopedStore() *secrets.Resolver {
	store := secrets.NewInMemoryStore()
	store.Register(secrets.SecretEntry{Name: "api-key", Value: "orders-secret", Domain: "orders"})
	store.Register(secrets.SecretEntry{Name: "api-key", Value: "payments-secret", Domain: "payments"})
	return secrets.NewResolver(store)
}

// A denied cross-domain reference must not fall back to an environment variable
// that happens to share the reference's name.
func TestResolveSecretDenialDoesNotFallBackToEnv(t *testing.T) {
	t.Setenv("payments.api-key", "LEAKED")

	src, err := NewFileSourceWithDomain("/tmp", "30s", engine.NewChannel("t", 1), "orders", scopedStore())
	if err != nil {
		t.Fatal(err)
	}

	value, err := src.resolveSecret("payments.api-key")
	if !errors.Is(err, secrets.ErrCrossDomainDenied) {
		t.Fatalf("denial must return ErrCrossDomainDenied, got %v", err)
	}
	if value != "" {
		t.Fatalf("SECURITY BUG: denied reference returned %q", value)
	}
}

// With no resolver or domain configured, the env fallback is the intended behaviour.
func TestResolveSecretUnscopedUsesEnv(t *testing.T) {
	t.Setenv("PLAIN_ENV_SECRET", "from-env")

	src, err := NewFileSourceWithDomain("/tmp", "30s", engine.NewChannel("t", 1), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	value, err := src.resolveSecret("PLAIN_ENV_SECRET")
	if err != nil || value != "from-env" {
		t.Fatalf("unscoped resolution should use env: got %q, err %v", value, err)
	}
}

// A resolver with a domain but a denied reference is a denial, not "unscoped".
func TestResolveSecretResolverWithoutDomainStillUsesEnv(t *testing.T) {
	t.Setenv("payments.api-key", "from-env")

	src, err := NewFileSourceWithDomain("/tmp", "30s", engine.NewChannel("t", 1), "", scopedStore())
	if err != nil {
		t.Fatal(err)
	}
	value, err := src.resolveSecret("payments.api-key")
	if err != nil || value != "from-env" {
		t.Fatalf("no domain configured means no domain scoping: got %q, err %v", value, err)
	}
}

func TestNewSFTPSourceRejectsDeniedPasswordRefAtConstruction(t *testing.T) {
	_, err := NewSFTPSourceWithDomain("localhost", 22, "user", "payments.api-key", "/up", "30s",
		engine.NewChannel("t", 1), "orders", scopedStore())
	if !errors.Is(err, secrets.ErrCrossDomainDenied) {
		t.Fatalf("construction must reject a denied password ref, got %v", err)
	}
}

func TestNewSFTPSourceAcceptsScopedPasswordRef(t *testing.T) {
	src, err := NewSFTPSourceWithDomain("localhost", 22, "user", "api-key", "/up", "30s",
		engine.NewChannel("t", 1), "orders", scopedStore())
	if err != nil {
		t.Fatalf("same-domain password ref should be accepted: %v", err)
	}
	if src.config.SFTPPassword != "api-key" {
		t.Fatalf("password ref not stored: %q", src.config.SFTPPassword)
	}
}

// pollSFTP must abort on a denied secret before attempting any network connection.
func TestPollSFTPAbortsOnDeniedSecret(t *testing.T) {
	src, err := NewSFTPSourceWithDomain("127.0.0.1", 1, "user", "", "/up", "30s",
		engine.NewChannel("t", 1), "orders", scopedStore())
	if err != nil {
		t.Fatal(err)
	}
	// Set directly to simulate a reference that was not validated at construction.
	src.config.SFTPPassword = "payments.api-key"

	err = src.pollSFTP(context.Background())
	if !errors.Is(err, secrets.ErrCrossDomainDenied) {
		t.Fatalf("poll must fail with the denial, not proceed; got %v", err)
	}
}
