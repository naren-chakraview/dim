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

// ResolveInRoute resolves secrets in the context of a route's domain
// Parses ${SECRET:ref} syntax and looks up via domain-scoped store
// Returns unresolved patterns for any secrets that cannot be resolved
func (r *Resolver) ResolveInRoute(routeDomain string, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	// Regex to find ${SECRET:ref} patterns
	pattern := regexp.MustCompile(`\$\{SECRET:([^}]+)\}`)

	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract ref from ${SECRET:ref}
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		ref := submatches[1]

		// Try domain-scoped store
		value, err := r.store.Resolve(routeDomain, ref)
		if err == nil {
			return value
		}

		// Return original pattern if not found (will be caught by validation)
		// Must NOT fall back to environment variables - a denied cross-domain reference
		// must fail, not silently succeed through a naming coincidence
		return match
	})

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
		refDomain, refName := parseSecretRef(ref)
		if refDomain != "" {
			// Cross-domain access is explicitly forbidden by domain-scoping policy
			return fmt.Errorf("cross-domain secret access denied: route in domain %q cannot access %q from domain %q",
				routeDomain, refName, refDomain)
		}

		// Try to resolve same-domain or global secret
		_, err := r.store.Resolve(routeDomain, ref)
		if err != nil {
			// Secret not found in same domain or global
			return fmt.Errorf("secret not found: %s (route domain: %s)", ref, routeDomain)
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
