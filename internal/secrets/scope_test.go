package secrets

import (
	"errors"
	"testing"
)

func TestResolveScopedRefusesReferenceWithoutContext(t *testing.T) {
	if _, err := ResolveScoped(nil, "", "${SECRET:db-pass}"); !errors.Is(err, ErrNoSecretContext) {
		t.Fatalf("no resolver/domain: got %v, want ErrNoSecretContext", err)
	}
	r := NewResolver(NewInMemoryStore())
	if _, err := ResolveScoped(r, "", "${SECRET:db-pass}"); !errors.Is(err, ErrNoSecretContext) {
		t.Fatalf("no domain: got %v, want ErrNoSecretContext", err)
	}
}

func TestResolveScopedPassesPlainText(t *testing.T) {
	got, err := ResolveScoped(nil, "", "kafka:9092")
	if err != nil || got != "kafka:9092" {
		t.Fatalf("plain text: got %q, %v", got, err)
	}
}
