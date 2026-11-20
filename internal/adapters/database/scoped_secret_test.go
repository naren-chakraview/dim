package database

import (
	"errors"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
)

func TestDatabaseSourceDSNIsDomainScoped(t *testing.T) {
	store := secrets.NewInMemoryStore()
	if err := store.Register(secrets.SecretEntry{Name: "orders-dsn", Value: "postgres://svc:pw@db/orders", Domain: "payments"}); err != nil {
		t.Fatal(err)
	}
	r := secrets.NewResolver(store)
	out := engine.NewChannel("test", 1)
	cfg := &SourceConfig{Driver: "postgres", DSN: "${SECRET:orders-dsn}", Query: "SELECT 1", WatermarkColumn: "id", WatermarkType: "integer", Schedule: time.Second}

	explicit := &SourceConfig{Driver: "postgres", DSN: "${SECRET:payments.orders-dsn}", Query: "SELECT 1", WatermarkColumn: "id", WatermarkType: "integer", Schedule: time.Second}
	if _, err := NewDatabaseSourceWithDomain(explicit, out, "fraud", r); !errors.Is(err, secrets.ErrCrossDomainDenied) {
		t.Fatalf("explicit cross-domain DSN: got %v, want ErrCrossDomainDenied", err)
	}
	if _, err := NewDatabaseSourceWithDomain(cfg, out, "fraud", r); err == nil {
		t.Fatal("unqualified DSN name from another domain was not refused")
	}
	if _, err := NewDatabaseSourceWithDomain(cfg, out, "", r); !errors.Is(err, secrets.ErrNoSecretContext) {
		t.Fatalf("no domain: got %v, want ErrNoSecretContext", err)
	}
	// Opening the database needs a driver and a live server, neither of which this test
	// has. What it asserts is that the DSN resolved in its own domain: any error must
	// not be a secret error.
	_, err := NewDatabaseSourceWithDomain(cfg, out, "payments", r)
	if errors.Is(err, secrets.ErrCrossDomainDenied) || errors.Is(err, secrets.ErrSecretNotFound) || errors.Is(err, secrets.ErrNoSecretContext) {
		t.Fatalf("own-domain DSN was not resolved: %v", err)
	}
	if cfg.DSN != "${SECRET:orders-dsn}" {
		t.Fatal("caller's config was mutated; resolution must copy")
	}
}
