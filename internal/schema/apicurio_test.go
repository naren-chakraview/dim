package schema

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestApicurioHealth tests basic connectivity to Apicurio if available.
// This test is skipped in CI unless APICURIO_URL is set.
func TestApicurioHealth(t *testing.T) {
	baseURL := os.Getenv("APICURIO_URL")
	if baseURL == "" {
		t.Skip("APICURIO_URL not set; skipping live Apicurio tests")
	}

	client, err := NewApicurioClient(baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Health(ctx); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

// TestApicurioRegisterAndFetch tests schema registration and retrieval (R18.1).
// Requires a running Apicurio instance at APICURIO_URL.
func TestApicurioRegisterAndFetch(t *testing.T) {
	baseURL := os.Getenv("APICURIO_URL")
	if baseURL == "" {
		t.Skip("APICURIO_URL not set; skipping live Apicurio tests")
	}

	client, err := NewApicurioClient(baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Register a schema
	schema := []byte(`{
		"type": "object",
		"properties": {
			"order_id": {"type": "string"},
			"amount": {"type": "number"}
		},
		"required": ["order_id"]
	}`)

	group := "test-schemas"
	subject := "OrderSchema"

	versionID, err := client.RegisterSchema(ctx, group, subject, schema)
	if err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	if versionID == "" {
		t.Fatal("empty version ID returned")
	}

	t.Logf("Registered schema with version ID: %s", versionID)

	// Fetch the schema back
	fetched, fetchedVersion, err := client.GetSchema(ctx, group, subject, "latest")
	if err != nil {
		t.Fatalf("failed to fetch schema: %v", err)
	}

	if string(fetched) != string(schema) {
		t.Errorf("fetched schema does not match original\nexpected: %s\ngot: %s", schema, fetched)
	}

	if fetchedVersion != versionID {
		t.Errorf("version mismatch: expected %s, got %s", versionID, fetchedVersion)
	}

	// Get latest version
	latestVersion, err := client.GetLatestVersion(ctx, group, subject)
	if err != nil {
		t.Fatalf("failed to get latest version: %v", err)
	}

	if latestVersion != versionID {
		t.Errorf("latest version mismatch: expected %s, got %s", versionID, latestVersion)
	}
}

// TestApicurioDefaultGroup tests that empty group defaults to "default" (R18.1).
func TestApicurioDefaultGroup(t *testing.T) {
	baseURL := os.Getenv("APICURIO_URL")
	if baseURL == "" {
		t.Skip("APICURIO_URL not set; skipping live Apicurio tests")
	}

	client, err := NewApicurioClient(baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	schema := []byte(`{"type": "object", "properties": {"id": {"type": "string"}}}`)

	// Register with empty group (should default to "default")
	versionID, err := client.RegisterSchema(ctx, "", "DefaultGroupSchema", schema)
	if err != nil {
		t.Fatalf("failed to register schema with default group: %v", err)
	}

	if versionID == "" {
		t.Fatal("empty version ID returned")
	}

	// Fetch with explicit "default" group
	fetched, _, err := client.GetSchema(ctx, "default", "DefaultGroupSchema", "latest")
	if err != nil {
		t.Fatalf("failed to fetch schema from default group: %v", err)
	}

	if string(fetched) != string(schema) {
		t.Errorf("fetched schema mismatch: expected %s, got %s", schema, fetched)
	}
}
