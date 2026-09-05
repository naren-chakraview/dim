package schema

import "context"

// AlternativeRegistry is a second implementation of RegistryBackend to demonstrate
// BYO support. This is a simplified implementation that wraps MockRegistry to show
// that multiple implementations can conform to the same interface (R18.4).
type AlternativeRegistry struct {
	inner *MockRegistry
}

// NewAlternativeRegistry creates a new alternative registry for testing.
func NewAlternativeRegistry() *AlternativeRegistry {
	return &AlternativeRegistry{
		inner: NewMockRegistry(),
	}
}

// Health checks the registry health.
func (ar *AlternativeRegistry) Health(ctx context.Context) error {
	return ar.inner.Health(ctx)
}

// RegisterSchema registers a schema.
func (ar *AlternativeRegistry) RegisterSchema(ctx context.Context, group, subject string, schema []byte) (string, error) {
	return ar.inner.RegisterSchema(ctx, group, subject, schema)
}

// GetSchema fetches a schema.
func (ar *AlternativeRegistry) GetSchema(ctx context.Context, group, subject, version string) ([]byte, string, error) {
	return ar.inner.GetSchema(ctx, group, subject, version)
}

// GetLatestVersion gets the latest version ID.
func (ar *AlternativeRegistry) GetLatestVersion(ctx context.Context, group, subject string) (string, error) {
	return ar.inner.GetLatestVersion(ctx, group, subject)
}
