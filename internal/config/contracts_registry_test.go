package config

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/naren-chakraview/dim/internal/schema"
)

// TestLoadRegistryBackedContract tests loading a registry-backed contract (R18.3).
func TestLoadRegistryBackedContract(t *testing.T) {
	// Use mock registry for testing
	mockRegistry := schema.NewMockRegistry()
	ctx := context.Background()

	// Register a schema in the mock registry
	schemaJSON := `{"type": "object", "properties": {"order_id": {"type": "string"}}}`
	_, err := mockRegistry.RegisterSchema(ctx, "test-group", "OrderSchema", []byte(schemaJSON))
	if err != nil {
		t.Fatalf("failed to register schema in mock: %v", err)
	}

	// Create a contract spec that references the registry schema
	contract := ContractSpec{
		ID:      "order-contract",
		Version: "1.0.0",
		Registry: &RegistryRefSpec{
			Type:    "mock", // Mock type — won't work with NewRegistryClient yet
			URL:     "http://mock",
			Group:   "test-group",
			Subject: "OrderSchema",
			Version: "latest",
		},
	}

	// Compute contract version tag
	contractVersion := computeRegistryContractVersion(contract.Registry)
	if contractVersion == "" {
		t.Fatal("contract version is empty")
	}

	// Verify format
	expectedPrefix := "mock:test-group:OrderSchema:"
	if contractVersion != expectedPrefix+"latest" {
		t.Errorf("contract version format mismatch: expected prefix %s, got %s", expectedPrefix, contractVersion)
	}

	t.Logf("Registry-backed contract version: %s", contractVersion)
}

// TestLoadInlineContract tests computing version for inline contracts (R18.3).
func TestLoadInlineContract(t *testing.T) {
	schemaJSON := `{"type": "object", "properties": {"id": {"type": "string"}}}`

	contractVersion := computeInlineContractVersion(schemaJSON)
	if contractVersion == "" {
		t.Fatal("contract version is empty")
	}

	// Verify SHA256 format
	if len(contractVersion) < len("sha256:") {
		t.Fatalf("contract version too short: %s", contractVersion)
	}

	if contractVersion[:7] != "sha256:" {
		t.Errorf("contract version format mismatch: expected sha256: prefix, got %s", contractVersion)
	}

	// Verify deterministic hashing
	contractVersion2 := computeInlineContractVersion(schemaJSON)
	if contractVersion != contractVersion2 {
		t.Errorf("contract version non-deterministic: %s vs %s", contractVersion, contractVersion2)
	}

	// Verify different schema produces different version
	differentSchema := `{"type": "object", "properties": {"name": {"type": "string"}}}`
	contractVersion3 := computeInlineContractVersion(differentSchema)
	if contractVersion == contractVersion3 {
		t.Errorf("different schema should produce different version")
	}

	t.Logf("Inline contract version: %s", contractVersion)
}

// TestContractStoreLoadInlineContract tests loading inline contracts into store (R18.3).
func TestContractStoreLoadInlineContract(t *testing.T) {
	cs := NewContractStore()

	schemaJSON := `{"type": "object", "properties": {"id": {"type": "string"}, "name": {"type": "string"}}}`
	var schemaObj interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaObj); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	contract := ContractSpec{
		ID:      "test-contract",
		Version: "1.0.0",
		Schema:  schemaObj,
	}

	err := cs.LoadContracts("test-route", []ContractSpec{contract})
	if err != nil {
		t.Fatalf("failed to load contract: %v", err)
	}

	// Verify contract was stored
	stored, err := cs.GetContractByID("test-route", "test-contract")
	if err != nil {
		t.Fatalf("failed to get contract: %v", err)
	}

	// Verify contract_version was computed
	if stored.ContractVersion == "" {
		t.Fatal("contract_version not computed")
	}

	if stored.ContractVersion[:7] != "sha256:" {
		t.Errorf("expected sha256: prefix, got %s", stored.ContractVersion)
	}
}

// TestContractVersionDeterminism tests that contract versions are deterministic (R18.3).
func TestContractVersionDeterminism(t *testing.T) {
	// Same schema should produce same version
	schema1 := `{"type": "object", "properties": {"id": {"type": "string"}}}`
	v1 := computeInlineContractVersion(schema1)
	v2 := computeInlineContractVersion(schema1)

	if v1 != v2 {
		t.Errorf("inline contract version not deterministic: %s vs %s", v1, v2)
	}

	// Registry contract version is deterministic
	ref := &RegistryRefSpec{
		Type:    "apicurio",
		URL:     "http://localhost:8080",
		Group:   "schemas",
		Subject: "TestSubject",
		Version: "3",
	}

	regV1 := computeRegistryContractVersion(ref)
	regV2 := computeRegistryContractVersion(ref)

	if regV1 != regV2 {
		t.Errorf("registry contract version not deterministic: %s vs %s", regV1, regV2)
	}
}

// TestContractVersionInLineage verifies contract_version would be available for lineage (R18.3).
// This is a placeholder test that verifies the contract_version field is populated.
func TestContractVersionInLineage(t *testing.T) {
	cs := NewContractStore()

	schemaJSON := `{"type": "object", "properties": {"id": {"type": "string"}}}`
	var schemaObj interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaObj); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	contract := ContractSpec{
		ID:      "lineage-test",
		Version: "1.0.0",
		Schema:  schemaObj,
	}

	err := cs.LoadContracts("test-route", []ContractSpec{contract})
	if err != nil {
		t.Fatalf("failed to load contract: %v", err)
	}

	stored, err := cs.GetContractByID("test-route", "lineage-test")
	if err != nil {
		t.Fatalf("failed to get contract: %v", err)
	}

	// Verify contract_version is available for lineage records
	if stored.ContractVersion == "" {
		t.Fatal("contract_version not set, won't be available for lineage/dead-letter records")
	}

	// This would be used in lineage like: record.ContractVersion = contract.ContractVersion
	t.Logf("Contract version for lineage: %s", stored.ContractVersion)
}
