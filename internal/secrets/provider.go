package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrProviderUnavailable wraps any failure to reach or read the secret service.
// It is never treated as "secret not found": a resolution that cannot reach the
// service fails closed.
var ErrProviderUnavailable = errors.New("secret provider unavailable")

// ErrReadOnly is returned by operations a provider-backed store does not support.
// Secrets are created and rotated in the provider, not through dim.
var ErrReadOnly = errors.New("secret store is read-only: create and rotate secrets in the provider")

// Provider reads secret values from an external secret service (AWS Secrets Manager,
// HashiCorp Vault or OpenBao, and so on). Paths use "/" separators.
type Provider interface {
	// Get returns the value stored at path. It returns ErrSecretNotFound when the path
	// does not exist, and an error wrapping ErrProviderUnavailable otherwise.
	Get(ctx context.Context, path string) (string, error)

	// List returns the paths under prefix (prefix ends with "/").
	List(ctx context.Context, prefix string) ([]string, error)
}

// ProviderStore is a SecretStore backed by a Provider. Secrets for a domain live at
// <prefix>/<domain>/<name>; global secrets at <prefix>/_global/<name>.
//
// Resolution follows the same rules as InMemoryStore: an unqualified name is looked up
// in the requesting domain, then globally, and an explicit "domain.name" reference is
// always denied. A provider failure is returned as an error, never as not found.
type ProviderStore struct {
	provider Provider
	prefix   string
}

// NewProviderStore returns a store that reads from provider, under prefix.
func NewProviderStore(provider Provider, prefix string) (*ProviderStore, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider store needs a provider")
	}
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return nil, fmt.Errorf("provider store needs a non-empty prefix")
	}
	return &ProviderStore{provider: provider, prefix: prefix}, nil
}

// domainPath returns the provider path of a secret in a domain ("" means global).
func (s *ProviderStore) domainPath(domain, name string) string {
	if domain == "" {
		domain = "_global"
	}
	return s.prefix + "/" + domain + "/" + name
}

// Register is not supported: this store is read-only.
func (s *ProviderStore) Register(entry SecretEntry) error {
	return ErrReadOnly
}

// Resolve returns the value of secretRef for a route in requestingDomain.
func (s *ProviderStore) Resolve(requestingDomain, secretRef string) (string, error) {
	refDomain, refName, err := parseSecretRef(secretRef)
	if err != nil {
		return "", err
	}
	if refDomain != "" {
		return "", fmt.Errorf("%w: route in domain %q cannot access %q from domain %q",
			ErrCrossDomainDenied, requestingDomain, refName, refDomain)
	}

	ctx := context.Background()
	if requestingDomain != "" {
		value, err := s.provider.Get(ctx, s.domainPath(requestingDomain, refName))
		if err == nil {
			return value, nil
		}
		if !errors.Is(err, ErrSecretNotFound) {
			return "", err
		}
	}

	value, err := s.provider.Get(ctx, s.domainPath("", refName))
	if err == nil {
		return value, nil
	}
	if errors.Is(err, ErrSecretNotFound) {
		return "", fmt.Errorf("%w: %s (requesting domain: %s)", ErrSecretNotFound, refName, requestingDomain)
	}
	return "", err
}

// ListByDomain returns the names of the secrets in a domain. Values are not returned.
func (s *ProviderStore) ListByDomain(domain string) ([]SecretEntry, error) {
	paths, err := s.provider.List(context.Background(), s.prefix+"/"+domain+"/")
	if err != nil {
		return nil, err
	}
	entries := make([]SecretEntry, 0, len(paths))
	for _, p := range paths {
		name := p[strings.LastIndex(p, "/")+1:]
		entries = append(entries, SecretEntry{Name: name, Domain: domain})
	}
	return entries, nil
}

// Clear is not supported: this store is read-only.
func (s *ProviderStore) Clear() {}
