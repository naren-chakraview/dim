package secrets

import (
	"fmt"
	"os"
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
// Falls back to environment variables for backward compatibility during migration
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

		// Try domain-scoped store first
		value, err := r.store.Resolve(routeDomain, ref)
		if err == nil {
			return value
		}

		// Fall back to environment variables for migration period
		if envVal := os.Getenv(ref); envVal != "" {
			return envVal
		}

		// Return original if not found (will be caught by validation)
		return match
	})

	return result, nil
}

// ValidateSecretReferences checks that all ${SECRET:ref} references are resolvable
// Returns error if any secrets cannot be resolved in the route's domain
func (r *Resolver) ValidateSecretReferences(routeDomain string, text string) error {
	if text == "" {
		return nil
	}

	pattern := regexp.MustCompile(`\$\{SECRET:([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		ref := match[1]

		// Try to resolve
		_, err := r.store.Resolve(routeDomain, ref)
		if err != nil {
			// Check if it's a cross-domain reference (contains a dot)
			refDomain, _ := parseSecretRef(ref)
			if refDomain != "" {
				// Explicit cross-domain syntax — needs to be audited
				continue
			}

			// Implicit cross-domain access attempt
			return fmt.Errorf("secret resolution failed: %s (route domain: %s). "+
				"If this secret is in a different domain, use explicit syntax: ${SECRET:domain.name}",
				ref, routeDomain)
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
