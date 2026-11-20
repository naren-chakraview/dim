package kafka

import (
	"errors"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
)

func scopedResolver(t *testing.T) *secrets.Resolver {
	t.Helper()
	store := secrets.NewInMemoryStore()
	if err := store.Register(secrets.SecretEntry{Name: "kafka-user", Value: "svc-payments", Domain: "payments"}); err != nil {
		t.Fatal(err)
	}
	return secrets.NewResolver(store)
}

func TestKafkaSourceSecretsAreDomainScoped(t *testing.T) {
	r := scopedResolver(t)
	out := engine.NewChannel("test", 1)

	// Explicit cross-domain syntax is denied outright.
	if _, err := NewKafkaSourceWithDomain([]string{"b:9092"}, "${SECRET:payments.kafka-user}", "g", out, "fraud", r); !errors.Is(err, secrets.ErrCrossDomainDenied) {
		t.Fatalf("explicit cross-domain reference: got %v, want ErrCrossDomainDenied", err)
	}
	// An unqualified name only resolves in the requesting domain, so it is refused too.
	if _, err := NewKafkaSourceWithDomain([]string{"b:9092"}, "${SECRET:kafka-user}", "g", out, "fraud", r); err == nil {
		t.Fatal("unqualified name from another domain was not refused")
	}
	if _, err := NewKafkaSourceWithDomain([]string{"b:9092"}, "${SECRET:kafka-user}", "g", out, "", r); !errors.Is(err, secrets.ErrNoSecretContext) {
		t.Fatalf("no domain: got %v, want ErrNoSecretContext", err)
	}
	src, err := NewKafkaSourceWithDomain([]string{"b:9092"}, "${SECRET:kafka-user}", "g", out, "payments", r)
	if err != nil {
		t.Fatalf("own domain: %v", err)
	}
	if src.config.Topic != "svc-payments" {
		t.Fatalf("topic not resolved in own domain: %q", src.config.Topic)
	}
}
