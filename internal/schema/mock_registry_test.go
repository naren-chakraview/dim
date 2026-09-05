package schema

import (
	"context"
	"testing"
)

// TestMockRegistryRegisterAndFetch tests the mock registry (R18.2).
func TestMockRegistryRegisterAndFetch(t *testing.T) {
	registry := NewMockRegistry()
	ctx := context.Background()

	schema := []byte(`{"type": "object", "properties": {"id": {"type": "string"}}}`)

	versionID, err := registry.RegisterSchema(ctx, "test-group", "TestSubject", schema)
	if err != nil {
		t.Fatalf("RegisterSchema failed: %v", err)
	}

	if versionID == "" {
		t.Fatal("version ID is empty")
	}

	fetched, fetchedVersion, err := registry.GetSchema(ctx, "test-group", "TestSubject", "latest")
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	if string(fetched) != string(schema) {
		t.Errorf("schema mismatch: expected %s, got %s", schema, fetched)
	}

	if fetchedVersion != versionID {
		t.Errorf("version mismatch: expected %s, got %s", versionID, fetchedVersion)
	}
}

// TestMockRegistryDefaultGroup tests default group handling (R18.2).
func TestMockRegistryDefaultGroup(t *testing.T) {
	registry := NewMockRegistry()
	ctx := context.Background()

	schema := []byte(`{"type": "string"}`)

	versionID, err := registry.RegisterSchema(ctx, "", "DefaultGroupSubject", schema)
	if err != nil {
		t.Fatalf("RegisterSchema with empty group failed: %v", err)
	}

	_, fetchedVersion, err := registry.GetSchema(ctx, "default", "DefaultGroupSubject", "latest")
	if err != nil {
		t.Fatalf("GetSchema from default group failed: %v", err)
	}

	if fetchedVersion != versionID {
		t.Errorf("version mismatch: expected %s, got %s", versionID, fetchedVersion)
	}
}

// TestMockRegistryGetLatestVersion tests GetLatestVersion (R18.2).
func TestMockRegistryGetLatestVersion(t *testing.T) {
	registry := NewMockRegistry()
	ctx := context.Background()

	schema1 := []byte(`{"version": 1}`)
	schema2 := []byte(`{"version": 2}`)

	v1, err := registry.RegisterSchema(ctx, "test", "Subject", schema1)
	if err != nil {
		t.Fatalf("RegisterSchema v1 failed: %v", err)
	}

	v2, err := registry.RegisterSchema(ctx, "test", "Subject", schema2)
	if err != nil {
		t.Fatalf("RegisterSchema v2 failed: %v", err)
	}

	latest, err := registry.GetLatestVersion(ctx, "test", "Subject")
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	if latest != v2 {
		t.Errorf("latest version mismatch: expected %s, got %s", v2, latest)
	}

	// Should be able to fetch previous version by version ID
	prevSchema, prevVersion, err := registry.GetSchema(ctx, "test", "Subject", v1)
	if err != nil {
		t.Fatalf("GetSchema with specific version failed: %v", err)
	}

	if prevVersion != v1 {
		t.Errorf("version mismatch: expected %s, got %s", v1, prevVersion)
	}

	if string(prevSchema) != string(schema1) {
		t.Errorf("schema mismatch for v1: expected %s, got %s", schema1, prevSchema)
	}
}

// TestMockRegistryHealth tests health check (R18.2).
func TestMockRegistryHealth(t *testing.T) {
	registry := NewMockRegistry()
	ctx := context.Background()

	if err := registry.Health(ctx); err != nil {
		t.Fatalf("Health check failed when healthy: %v", err)
	}

	registry.SetUnhealthy(true)

	if err := registry.Health(ctx); err == nil {
		t.Fatal("Health check should fail when unhealthy")
	}
}

// TestMockRegistryUnhealthyErrors tests that operations fail when unhealthy (R18.2).
func TestMockRegistryUnhealthyErrors(t *testing.T) {
	registry := NewMockRegistry()
	ctx := context.Background()

	registry.SetUnhealthy(true)

	schema := []byte(`{"type": "string"}`)

	_, err := registry.RegisterSchema(ctx, "test", "Subject", schema)
	if err == nil {
		t.Fatal("RegisterSchema should fail when unhealthy")
	}

	_, _, err = registry.GetSchema(ctx, "test", "Subject", "latest")
	if err == nil {
		t.Fatal("GetSchema should fail when unhealthy")
	}

	_, err = registry.GetLatestVersion(ctx, "test", "Subject")
	if err == nil {
		t.Fatal("GetLatestVersion should fail when unhealthy")
	}
}

// TestMockRegistryNotFound tests error handling for missing subjects (R18.2).
func TestMockRegistryNotFound(t *testing.T) {
	registry := NewMockRegistry()
	ctx := context.Background()

	_, _, err := registry.GetSchema(ctx, "nonexistent", "Subject", "latest")
	if err == nil {
		t.Fatal("GetSchema should fail for nonexistent group")
	}

	schema := []byte(`{"type": "string"}`)
	registry.RegisterSchema(ctx, "test", "Subject1", schema)

	_, _, err = registry.GetSchema(ctx, "test", "NonexistentSubject", "latest")
	if err == nil {
		t.Fatal("GetSchema should fail for nonexistent subject")
	}

	_, err = registry.GetLatestVersion(ctx, "nonexistent", "Subject")
	if err == nil {
		t.Fatal("GetLatestVersion should fail for nonexistent group")
	}
}
