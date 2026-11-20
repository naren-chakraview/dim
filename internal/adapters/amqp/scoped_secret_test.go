package amqp

import (
	"errors"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
)

func TestAMQPSourceSecretsAreDomainScoped(t *testing.T) {
	store := secrets.NewInMemoryStore()
	if err := store.Register(secrets.SecretEntry{Name: "amqp-url", Value: "amqp://svc:pw@broker:5672/", Domain: "payments"}); err != nil {
		t.Fatal(err)
	}
	r := secrets.NewResolver(store)
	out := engine.NewChannel("test", 1)

	if _, err := NewAMQPSourceWithDomain("${SECRET:payments.amqp-url}", "q", out, "fraud", r); !errors.Is(err, secrets.ErrCrossDomainDenied) {
		t.Fatalf("explicit cross-domain: got %v, want ErrCrossDomainDenied", err)
	}
	if _, err := NewAMQPSourceWithDomain("${SECRET:amqp-url}", "q", out, "fraud", r); err == nil {
		t.Fatal("unqualified name from another domain was not refused")
	}
	if _, err := NewAMQPSourceWithDomain("${SECRET:amqp-url}", "q", out, "", r); !errors.Is(err, secrets.ErrNoSecretContext) {
		t.Fatalf("no domain: got %v, want ErrNoSecretContext", err)
	}
	// Constructing an AMQP source dials the broker, so with no broker running the
	// constructor fails on the dial. That is acceptable here: what this test asserts is
	// that the reference resolved in its own domain (no secret error). Live dialing is
	// covered by the live source test.
	_, err := NewAMQPSourceWithDomain("${SECRET:amqp-url}", "q", out, "payments", r)
	if errors.Is(err, secrets.ErrCrossDomainDenied) || errors.Is(err, secrets.ErrSecretNotFound) || errors.Is(err, secrets.ErrNoSecretContext) {
		t.Fatalf("own-domain reference was not resolved: %v", err)
	}
}
