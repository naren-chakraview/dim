package secrets

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// An unreachable endpoint is a provider failure, not a missing secret.
func TestAWSProviderUnreachableIsUnavailable(t *testing.T) {
	saved := providerCallTimeout
	providerCallTimeout = 500 * time.Millisecond
	defer func() { providerCallTimeout = saved }()
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	p, err := NewAWSProvider(context.Background(), "us-east-1", "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Get(context.Background(), "dim/payments/db")
	if !errors.Is(err, ErrProviderUnavailable) || errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("unreachable endpoint: got %v, want ErrProviderUnavailable only", err)
	}
}

// Live against LocalStack Secrets Manager (deploy/docker-compose.e2e.yml). Runs only
// when DIM_LIVE_LOCALSTACK is set, so the default test run needs no containers.
func TestAWSProviderLiveLocalStack(t *testing.T) {
	endpoint := os.Getenv("DIM_LIVE_LOCALSTACK")
	if endpoint == "" {
		t.Skip("set DIM_LIVE_LOCALSTACK=http://localhost:4566 to run against LocalStack")
	}
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	ctx := context.Background()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("us-east-1"))
	if err != nil {
		t.Fatal(err)
	}
	admin := secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) { o.BaseEndpoint = aws.String(endpoint) })
	for name, value := range map[string]string{
		"dim/payments/live-db": "live-payments-pass",
		"dim/fraud/live-db":    "live-fraud-pass",
	} {
		_, err := admin.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:               aws.String(name),
			SecretString:       aws.String(value),
			ClientRequestToken: aws.String("6f1d2a3b-0001-4c5d-8e9f-000000000001"),
		})
		if err != nil {
			// The secret may already exist from an earlier run; update it instead.
			if _, err2 := admin.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
				SecretId:           aws.String(name),
				SecretString:       aws.String(value),
				ClientRequestToken: aws.String("6f1d2a3b-0002-4c5d-8e9f-000000000002"),
			}); err2 != nil {
				t.Fatalf("create/update %s: %v / %v", name, err, err2)
			}
		}
	}

	p, err := NewAWSProvider(ctx, "us-east-1", endpoint)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewProviderStore(p, "dim")
	if err != nil {
		t.Fatal(err)
	}

	if v, err := store.Resolve("payments", "live-db"); err != nil || v != "live-payments-pass" {
		t.Fatalf("own domain from LocalStack: got %q, %v", v, err)
	}
	if _, err := store.Resolve("payments", "fraud.live-db"); !errors.Is(err, ErrCrossDomainDenied) {
		t.Fatalf("explicit cross-domain from LocalStack: got %v, want ErrCrossDomainDenied", err)
	}
	if _, err := store.Resolve("fraud", "missing-secret"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("missing secret: got %v, want ErrSecretNotFound", err)
	}
	entries, err := store.ListByDomain("payments")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Name == "live-db" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ListByDomain(payments) = %+v, want live-db", entries)
	}
}
