package schema

import (
	"context"
	"testing"
)

// ConformanceTestSuite runs a set of tests against any RegistryBackend implementation.
// This allows testing that any registry implementation (Apicurio, custom, etc.) conforms
// to the expected interface and behavior (R18.4: BYO registry support).
type ConformanceTestSuite struct {
	Backend RegistryBackend
	T       *testing.T
}

// TestRegisterAndFetch tests basic register/fetch functionality.
func (cts *ConformanceTestSuite) TestRegisterAndFetch() {
	ctx := context.Background()

	schema := []byte(`{"type": "object", "properties": {"id": {"type": "string"}}}`)
	group := "test"
	subject := "TestSchema"

	versionID, err := cts.Backend.RegisterSchema(ctx, group, subject, schema)
	if err != nil {
		cts.T.Fatalf("RegisterSchema failed: %v", err)
	}

	if versionID == "" {
		cts.T.Fatal("version ID is empty")
	}

	fetched, fetchedVersion, err := cts.Backend.GetSchema(ctx, group, subject, "latest")
	if err != nil {
		cts.T.Fatalf("GetSchema failed: %v", err)
	}

	if string(fetched) != string(schema) {
		cts.T.Errorf("schema mismatch: expected %s, got %s", schema, fetched)
	}

	if fetchedVersion != versionID {
		cts.T.Errorf("version mismatch: expected %s, got %s", versionID, fetchedVersion)
	}
}

// TestGetLatestVersion tests fetching the latest version of a schema.
func (cts *ConformanceTestSuite) TestGetLatestVersion() {
	ctx := context.Background()

	schema1 := []byte(`{"type": "string"}`)
	schema2 := []byte(`{"type": "integer"}`)

	group := "test"
	subject := "VersionedSchema"

	v1, err := cts.Backend.RegisterSchema(ctx, group, subject, schema1)
	if err != nil {
		cts.T.Fatalf("RegisterSchema v1 failed: %v", err)
	}

	v2, err := cts.Backend.RegisterSchema(ctx, group, subject, schema2)
	if err != nil {
		cts.T.Fatalf("RegisterSchema v2 failed: %v", err)
	}

	// GetLatestVersion should return v2
	latest, err := cts.Backend.GetLatestVersion(ctx, group, subject)
	if err != nil {
		cts.T.Fatalf("GetLatestVersion failed: %v", err)
	}

	if latest != v2 {
		cts.T.Errorf("GetLatestVersion returned %s, expected %s", latest, v2)
	}

	// GetSchema with "latest" should return v2
	latestSchema, latestVersion, err := cts.Backend.GetSchema(ctx, group, subject, "latest")
	if err != nil {
		cts.T.Fatalf("GetSchema latest failed: %v", err)
	}

	if string(latestSchema) != string(schema2) {
		cts.T.Errorf("latest schema mismatch: expected %s, got %s", schema2, latestSchema)
	}

	if latestVersion != v2 {
		cts.T.Errorf("latest version mismatch: expected %s, got %s", v2, latestVersion)
	}

	// GetSchema with v1 should return first version
	v1Schema, v1Version, err := cts.Backend.GetSchema(ctx, group, subject, v1)
	if err != nil {
		cts.T.Fatalf("GetSchema v1 failed: %v", err)
	}

	if string(v1Schema) != string(schema1) {
		cts.T.Errorf("v1 schema mismatch: expected %s, got %s", schema1, v1Schema)
	}

	if v1Version != v1 {
		cts.T.Errorf("v1 version mismatch: expected %s, got %s", v1, v1Version)
	}
}

// TestDefaultGroup tests that empty group defaults to "default".
func (cts *ConformanceTestSuite) TestDefaultGroup() {
	ctx := context.Background()

	schema := []byte(`{"type": "string"}`)
	subject := "DefaultGroupSchema"

	versionID, err := cts.Backend.RegisterSchema(ctx, "", subject, schema)
	if err != nil {
		cts.T.Fatalf("RegisterSchema with empty group failed: %v", err)
	}

	if versionID == "" {
		cts.T.Fatal("version ID is empty")
	}

	// Should be able to fetch using "default" group
	fetched, _, err := cts.Backend.GetSchema(ctx, "default", subject, "latest")
	if err != nil {
		cts.T.Fatalf("GetSchema from default group failed: %v", err)
	}

	if string(fetched) != string(schema) {
		cts.T.Errorf("schema mismatch: expected %s, got %s", schema, fetched)
	}
}

// TestHealth tests the health check endpoint.
func (cts *ConformanceTestSuite) TestHealth() {
	ctx := context.Background()

	if err := cts.Backend.Health(ctx); err != nil {
		cts.T.Errorf("Health check failed: %v", err)
	}
}

// TestNotFound tests error handling for missing subjects.
func (cts *ConformanceTestSuite) TestNotFound() {
	ctx := context.Background()

	_, _, err := cts.Backend.GetSchema(ctx, "nonexistent", "NonexistentSubject", "latest")
	if err == nil {
		cts.T.Fatal("GetSchema should fail for nonexistent subject")
	}

	_, err = cts.Backend.GetLatestVersion(ctx, "nonexistent", "NonexistentSubject")
	if err == nil {
		cts.T.Fatal("GetLatestVersion should fail for nonexistent subject")
	}
}

// RunConformanceTests runs all conformance tests against a backend.
// Usage in BYO implementation tests:
//   func TestMyRegistry(t *testing.T) {
//       backend := NewMyRegistry()
//       suite := &ConformanceTestSuite{Backend: backend, T: t}
//       suite.TestRegisterAndFetch()
//       suite.TestGetLatestVersion()
//       suite.TestDefaultGroup()
//       suite.TestHealth()
//       suite.TestNotFound()
//   }
func RunConformanceTests(t *testing.T, backend RegistryBackend) {
	suite := &ConformanceTestSuite{Backend: backend, T: t}
	suite.TestRegisterAndFetch()
	suite.TestGetLatestVersion()
	suite.TestDefaultGroup()
	suite.TestHealth()
	suite.TestNotFound()
}

// TestMockRegistryConformance runs the conformance test suite against MockRegistry (R18.4).
func TestMockRegistryConformance(t *testing.T) {
	backend := NewMockRegistry()
	RunConformanceTests(t, backend)
}

// TestAlternativeRegistryConformance runs the conformance test suite against
// AlternativeRegistry to demonstrate that different implementations can be
// swapped without code changes (R18.4: BYO registry support).
func TestAlternativeRegistryConformance(t *testing.T) {
	backend := NewAlternativeRegistry()
	RunConformanceTests(t, backend)
}
