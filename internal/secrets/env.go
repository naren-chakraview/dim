package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Environment variables that choose the secret provider.
const (
	EnvProvider   = "DIM_SECRET_PROVIDER" // localstack (default), aws, vault, openbao
	EnvPrefix     = "DIM_SECRET_PREFIX"   // path prefix for secrets (default "dim")
	EnvEndpoint   = "DIM_SECRET_ENDPOINT" // Secrets Manager endpoint; localstack default http://localhost:4566
	EnvRegion     = "DIM_SECRET_REGION"   // AWS region (default us-east-1)
	EnvVaultAddr  = "DIM_VAULT_ADDR"      // Vault or OpenBao address, for example http://127.0.0.1:8200
	EnvVaultToken = "DIM_VAULT_TOKEN"     // Vault or OpenBao token
	EnvVaultMount = "DIM_VAULT_MOUNT"     // KV v2 mount (default secret)
)

// localStackEndpoint is where the e2e stack's LocalStack listens.
const localStackEndpoint = "http://localhost:4566"

// ResolverFromEnv builds the secret resolver for this process from the environment.
// The provider defaults to LocalStack, which the local e2e stack runs; set
// DIM_SECRET_PROVIDER to aws, vault or openbao for other environments. A provider that
// is named but misconfigured is an error: nothing falls back to another provider or to
// the environment.
func ResolverFromEnv(ctx context.Context) (*Resolver, error) {
	prefix := envOr(EnvPrefix, "dim")
	var provider Provider

	switch name := strings.ToLower(envOr(EnvProvider, "localstack")); name {
	case "localstack", "aws":
		endpoint := os.Getenv(EnvEndpoint)
		if name == "localstack" && endpoint == "" {
			endpoint = localStackEndpoint
		}
		p, err := NewAWSProvider(ctx, envOr(EnvRegion, "us-east-1"), endpoint)
		if err != nil {
			return nil, fmt.Errorf("%s=%s: %w", EnvProvider, name, err)
		}
		provider = p
	case "vault", "openbao":
		p, err := NewVaultProvider(os.Getenv(EnvVaultAddr), os.Getenv(EnvVaultToken), os.Getenv(EnvVaultMount))
		if err != nil {
			return nil, fmt.Errorf("%s=%s: %w", EnvProvider, name, err)
		}
		provider = p
	default:
		return nil, fmt.Errorf("%s=%q: unknown secret provider (want localstack, aws, vault or openbao)", EnvProvider, name)
	}

	store, err := NewProviderStore(provider, prefix)
	if err != nil {
		return nil, err
	}
	return NewResolver(store), nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
