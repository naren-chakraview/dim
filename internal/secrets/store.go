package secrets

import (
	"fmt"
	"sync"
)

// SecretEntry represents a secret with its domain scope
type SecretEntry struct {
	Name   string // Secret name
	Value  string // Secret value
	Domain string // Domain scope ("" means global, "@shared" for explicitly shared)
}

// SecretStore defines the interface for secret registration and lookup
type SecretStore interface {
	// Register stores a secret with domain scope
	Register(entry SecretEntry) error

	// Resolve retrieves a secret, checking domain access
	// requestingDomain: the domain of the route requesting the secret
	// secretRef: "name" for global, "domain.name" for cross-domain explicit
	Resolve(requestingDomain, secretRef string) (string, error)

	// ListByDomain returns all secrets in a domain
	ListByDomain(domain string) ([]SecretEntry, error)

	// Clear removes all secrets (for testing)
	Clear()
}

// InMemoryStore implements SecretStore in-memory
type InMemoryStore struct {
	mu      sync.RWMutex
	secrets map[string]SecretEntry // key: "domain/name"
}

// NewInMemoryStore creates a new in-memory secret store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		secrets: make(map[string]SecretEntry),
	}
}

// Register stores a secret with domain scope
func (s *InMemoryStore) Register(entry SecretEntry) error {
	if entry.Name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(entry.Domain, entry.Name)
	s.secrets[key] = entry
	return nil
}

// Resolve retrieves a secret, checking domain access
// Returns error if:
// - secret not found
// - requesting route attempts cross-domain access (blocked by domain-scoping)
func (s *InMemoryStore) Resolve(requestingDomain, secretRef string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Parse secretRef: could be "name" (global) or "domain.name" (explicit cross-domain)
	refDomain, refName := parseSecretRef(secretRef)

	// CRITICAL: Cross-domain access is explicitly blocked, even with explicit syntax
	// The explicit syntax (domain.name) documents intent but does NOT grant access
	// This enforces the domain-scoping isolation required by the governance model
	if refDomain != "" {
		// Attempted cross-domain access
		return "", fmt.Errorf("cross-domain secret access denied: route in domain %q cannot access %q from domain %q",
			requestingDomain, refName, refDomain)
	}

	// Only same-domain and global secrets are accessible
	// Try scoped to requesting domain first
	if requestingDomain != "" {
		key := makeKey(requestingDomain, refName)
		if entry, ok := s.secrets[key]; ok {
			return entry.Value, nil
		}
	}

	// Try global secrets (domain = "")
	key := makeKey("", refName)
	if entry, ok := s.secrets[key]; ok {
		return entry.Value, nil
	}

	return "", fmt.Errorf("secret not found: %s (requesting domain: %s)", refName, requestingDomain)
}

// ListByDomain returns all secrets in a domain
func (s *InMemoryStore) ListByDomain(domain string) ([]SecretEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []SecretEntry
	for _, entry := range s.secrets {
		if entry.Domain == domain {
			results = append(results, entry)
		}
	}
	return results, nil
}

// Clear removes all secrets (for testing)
func (s *InMemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets = make(map[string]SecretEntry)
}

// Helper: make storage key from domain and name
func makeKey(domain, name string) string {
	if domain == "" {
		return "global/" + name
	}
	return domain + "/" + name
}

// Helper: parse secret reference into domain and name
// "name" -> ("", "name")
// "domain.name" -> ("domain", "name")
func parseSecretRef(ref string) (string, string) {
	// Simple split on first dot
	for i, ch := range ref {
		if ch == '.' {
			return ref[:i], ref[i+1:]
		}
	}
	// No dot found - return as name with no domain
	return "", ref
}
