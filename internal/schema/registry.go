package schema

import (
	"context"
	"fmt"
)

// RegistryBackend is a pluggable schema registry interface. A registry "subject"
// is a named schema group (e.g., "OrderSchema"), distinct from a lineage/privacy
// "subject_id" (a data subject for privacy purposes). This naming collision is
// resolved by always using the full qualified name "registry subject" in code
// and documentation when referring to the registry concept.
type RegistryBackend interface {
	// RegisterSchema registers or updates a schema, returning the version ID.
	// The registry performs compatibility checks; RegisterSchema returns an error
	// if the schema is incompatible per the registry's rules.
	RegisterSchema(ctx context.Context, group, subject string, schema []byte) (versionID string, err error)

	// GetSchema fetches a schema by group, subject, and version.
	// If version is "latest", fetches the most recent version.
	GetSchema(ctx context.Context, group, subject, version string) (schema []byte, versionID string, err error)

	// GetLatestVersion returns the version ID of the latest schema for a subject.
	GetLatestVersion(ctx context.Context, group, subject string) (versionID string, err error)

	// Health checks if the registry is accessible and healthy.
	Health(ctx context.Context) error
}

// SchemaReference holds a reference to a schema in a registry.
type SchemaReference struct {
	Type    string // "apicurio", "confluent", "aws-glue", "azure", etc.
	URL     string
	Group   string // registry group (e.g., "order-schemas")
	Subject string // registry subject: a named schema group, NOT a data subject
	Version string // "latest" or specific version ID
}

// NewRegistryClient creates a registry client for the given backend type.
func NewRegistryClient(ctx context.Context, ref *SchemaReference) (RegistryBackend, error) {
	switch ref.Type {
	case "apicurio":
		return NewApicurioClient(ref.URL)
	case "mock":
		// Mock registry for testing (R18.4)
		return NewMockRegistry(), nil
	case "alternative":
		// Alternative registry implementation for BYO testing (R18.4)
		return NewAlternativeRegistry(), nil
	default:
		return nil, fmt.Errorf("unsupported registry type: %q", ref.Type)
	}
}
