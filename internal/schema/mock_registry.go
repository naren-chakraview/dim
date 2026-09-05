package schema

import (
	"context"
	"fmt"
	"sync"
)

// MockRegistry is a RegistryBackend implementation for testing and BYO support verification.
// It demonstrates that the registry interface is truly pluggable and not tied to Apicurio.
type MockRegistry struct {
	mu       sync.RWMutex
	schemas  map[string]map[string][]schemaVersion // group -> subject -> versions
	nextVer  map[string]int                         // group:subject -> next version ID
	healthy  bool
}

// schemaVersion holds a schema and its version ID.
type schemaVersion struct {
	versionID string
	content   []byte
}

// NewMockRegistry creates a new mock registry for testing.
func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		schemas: make(map[string]map[string][]schemaVersion),
		nextVer: make(map[string]int),
		healthy: true,
	}
}

// Health returns nil if healthy, error if unhealthy.
func (mr *MockRegistry) Health(ctx context.Context) error {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	if !mr.healthy {
		return fmt.Errorf("mock registry unhealthy")
	}
	return nil
}

// SetUnhealthy marks the registry as unhealthy for testing failure scenarios.
func (mr *MockRegistry) SetUnhealthy(unhealthy bool) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.healthy = !unhealthy
}

// RegisterSchema registers a new schema version.
func (mr *MockRegistry) RegisterSchema(ctx context.Context, group, subject string, schema []byte) (string, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if group == "" {
		group = "default"
	}

	if !mr.healthy {
		return "", fmt.Errorf("mock registry unhealthy")
	}

	if _, ok := mr.schemas[group]; !ok {
		mr.schemas[group] = make(map[string][]schemaVersion)
	}

	key := group + ":" + subject
	versionID := fmt.Sprintf("%d", mr.nextVer[key]+1)
	mr.nextVer[key]++

	if _, ok := mr.schemas[group][subject]; !ok {
		mr.schemas[group][subject] = []schemaVersion{}
	}

	// Prepend to list so "latest" is first
	mr.schemas[group][subject] = append([]schemaVersion{
		{versionID: versionID, content: schema},
	}, mr.schemas[group][subject]...)

	return versionID, nil
}

// GetSchema retrieves a schema by group, subject, and version.
func (mr *MockRegistry) GetSchema(ctx context.Context, group, subject, version string) ([]byte, string, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	if group == "" {
		group = "default"
	}

	if !mr.healthy {
		return nil, "", fmt.Errorf("mock registry unhealthy")
	}

	subjects, ok := mr.schemas[group]
	if !ok {
		return nil, "", fmt.Errorf("group not found: %s", group)
	}

	versions, ok := subjects[subject]
	if !ok || len(versions) == 0 {
		return nil, "", fmt.Errorf("subject not found: %s/%s", group, subject)
	}

	if version == "" || version == "latest" {
		// Return latest (first in list)
		return versions[0].content, versions[0].versionID, nil
	}

	// Find specific version
	for _, v := range versions {
		if v.versionID == version {
			return v.content, v.versionID, nil
		}
	}

	return nil, "", fmt.Errorf("version not found: %s/%s:%s", group, subject, version)
}

// GetLatestVersion returns the version ID of the latest schema for a subject.
func (mr *MockRegistry) GetLatestVersion(ctx context.Context, group, subject string) (string, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	if group == "" {
		group = "default"
	}

	if !mr.healthy {
		return "", fmt.Errorf("mock registry unhealthy")
	}

	subjects, ok := mr.schemas[group]
	if !ok {
		return "", fmt.Errorf("group not found: %s", group)
	}

	versions, ok := subjects[subject]
	if !ok || len(versions) == 0 {
		return "", fmt.Errorf("subject not found: %s/%s", group, subject)
	}

	return versions[0].versionID, nil
}

// ListSubjects returns all subjects in a group (for testing/listing).
func (mr *MockRegistry) ListSubjects(group string) []string {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	if group == "" {
		group = "default"
	}

	subjects, ok := mr.schemas[group]
	if !ok {
		return []string{}
	}

	result := make([]string, 0, len(subjects))
	for subject := range subjects {
		result = append(result, subject)
	}
	return result
}
