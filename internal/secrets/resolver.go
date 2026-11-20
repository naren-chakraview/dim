package secrets

import (
	"fmt"
	"regexp"
)

// Resolver handles domain-aware secret resolution
type Resolver struct {
	store SecretStore
}

// NewResolver creates a new domain-aware resolver
func NewResolver(store SecretStore) *Resolver {
	return &Resolver{store: store}
}

// ResolveInRoute resolves ${SECRET:ref} patterns in the context of a route's domain.
// It fails closed: if any reference cannot be resolved, it returns ("", err) where err
// wraps ErrCrossDomainDenied (explicit cross-domain reference) or ErrSecretNotFound.
// Callers can distinguish a denial from a missing secret with errors.Is.
// It never returns partially resolved text, and never falls back to environment variables.
func (r *Resolver) ResolveInRoute(routeDomain string, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	pattern := regexp.MustCompile(`\$\{SECRET:([^}]+)\}`)

	var firstErr error
	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		if firstErr != nil {
			return match
		}
		ref := pattern.FindStringSubmatch(match)[1]

		value, err := r.store.Resolve(routeDomain, ref)
		if err != nil {
			firstErr = fmt.Errorf("resolve ${SECRET:%s}: %w", ref, err)
			return match
		}
		return value
	})

	if firstErr != nil {
		return "", firstErr
	}
	return result, nil
}

// ValidateSecretReferences checks that all ${SECRET:ref} references are resolvable
// Returns error if any secrets cannot be resolved in the route's domain
// T1.13 Authorization: Cross-domain access is blocked entirely, including explicit syntax
func (r *Resolver) ValidateSecretReferences(routeDomain string, text string) error {
	if text == "" {
		return nil
	}

	pattern := regexp.MustCompile(`\$\{SECRET:([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		ref := match[1]

		// Check for cross-domain syntax early
		refDomain, refName, err := parseSecretRef(ref)
		if err != nil {
			return err
		}
		if refDomain != "" {
			// Cross-domain access is explicitly forbidden by domain-scoping policy
			return fmt.Errorf("%w: route in domain %q cannot access %q from domain %q", ErrCrossDomainDenied,
				routeDomain, refName, refDomain)
		}

		// Try to resolve same-domain or global secret
		_, err = r.store.Resolve(routeDomain, ref)
		if err != nil {
			// Secret not found in same domain or global
			return fmt.Errorf("%w: %s (route domain: %s)", ErrSecretNotFound, ref, routeDomain)
		}
	}

	return nil
}

// ListUnresolvedSecrets returns all ${SECRET:ref} patterns in text that cannot be resolved
func (r *Resolver) ListUnresolvedSecrets(routeDomain string, text string) ([]string, error) {
	if text == "" {
		return nil, nil
	}

	pattern := regexp.MustCompile(`\$\{SECRET:([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	var unresolved []string
	for _, match := range matches {
		ref := match[1]

		_, err := r.store.Resolve(routeDomain, ref)
		if err != nil {
			unresolved = append(unresolved, ref)
		}
	}

	return unresolved, nil
}
