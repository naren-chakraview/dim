package s3

import (
	"errors"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
)

func TestS3SourceSecretsAreDomainScoped(t *testing.T) {
	store := secrets.NewInMemoryStore()
	if err := store.Register(secrets.SecretEntry{Name: "bucket", Value: "payments-landing", Domain: "payments"}); err != nil {
		t.Fatal(err)
	}
	r := secrets.NewResolver(store)
	out := engine.NewChannel("test", 1)
	cfg := SourceConfig{Bucket: "${SECRET:bucket}", Region: "eu-west-1"}

	explicit := SourceConfig{Bucket: "${SECRET:payments.bucket}", Region: "eu-west-1"}
	if _, err := NewS3SourceWithDomain(explicit, out, "fraud", r); !errors.Is(err, secrets.ErrCrossDomainDenied) {
		t.Fatalf("explicit cross-domain: got %v, want ErrCrossDomainDenied", err)
	}
	if _, err := NewS3SourceWithDomain(cfg, out, "fraud", r); err == nil {
		t.Fatal("unqualified name from another domain was not refused")
	}
	if _, err := NewS3SourceWithDomain(cfg, out, "", r); !errors.Is(err, secrets.ErrNoSecretContext) {
		t.Fatalf("no domain: got %v, want ErrNoSecretContext", err)
	}
	src, err := NewS3SourceWithDomain(cfg, out, "payments", r)
	if err != nil {
		t.Fatalf("own domain: %v", err)
	}
	if src.bucket != "payments-landing" {
		t.Fatalf("bucket not resolved: %q", src.bucket)
	}
}
